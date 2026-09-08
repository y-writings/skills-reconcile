package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveWorkspacePrecedence(t *testing.T) {
	for _, tc := range []struct {
		name, flag, env, config, want string
	}{
		{"flag overrides environment and config", "flag", "env", "config", "flag"},
		{"environment overrides config", "", "env", "config", "env"},
		{"config overrides cwd", "", "", "config", "config"},
		{"missing config uses cwd", "", "", "", "cwd"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := isolateResolve(t)
			paths := map[string]string{"": "", "cwd": base}
			for _, name := range []string{"flag", "env", "config"} {
				paths[name] = filepath.Join(base, name)
				makeDir(t, paths[name])
			}
			t.Setenv("SKILLS_RECONCILE_WORKSPACE", paths[tc.env])
			if tc.config != "" {
				writeConfig(t, os.Getenv("XDG_CONFIG_HOME"), fmt.Sprintf(`{"workspace":%q}`, paths[tc.config]))
			}

			root, manifest, err := Resolve(paths[tc.flag], "")
			if err != nil {
				t.Fatal(err)
			}
			want := paths[tc.want]
			if root != want || manifest != filepath.Join(want, "skills-manifest.json") {
				t.Fatalf("resolved (%q, %q), want workspace %q and its default manifest", root, manifest, want)
			}
		})
	}
}

func TestResolveConfigLocation(t *testing.T) {
	for _, tc := range []struct {
		name, xdg, home, config string
		wantHome                bool
	}{
		{"unset XDG uses HOME config", "", "home", "", true},
		{"missing XDG config does not fall back to HOME", "xdg", "home", "", false},
		{"empty workspace uses cwd", "xdg", "home", `{"workspace":""}`, false},
		{"missing workspace uses cwd", "xdg", "home", `{}`, false},
		{"trailing JSON whitespace uses cwd", "xdg", "home", "{} \n\t", false},
		{"no config location uses cwd", "", "", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := isolateResolve(t)
			homeWorkspace := filepath.Join(base, "home-workspace")
			makeDir(t, homeWorkspace)
			writeConfig(t, filepath.Join(base, "home", ".config"), fmt.Sprintf(`{"workspace":%q}`, homeWorkspace))
			for key, value := range map[string]string{"HOME": tc.home, "XDG_CONFIG_HOME": tc.xdg} {
				if value != "" {
					value = filepath.Join(base, value)
				}
				t.Setenv(key, value)
			}
			if tc.config != "" {
				writeConfig(t, os.Getenv("XDG_CONFIG_HOME"), tc.config)
			}

			root, _, err := Resolve("", "")
			want := base
			if tc.wantHome {
				want = homeWorkspace
			}
			if err != nil || root != want {
				t.Fatalf("workspace = %q, error = %v; want %q", root, err, want)
			}
		})
	}
}

func TestResolveExplicitSelectionDoesNotReadLowerPriorityConfig(t *testing.T) {
	for _, tc := range []struct {
		name, selector string
		relativeConfig bool
	}{
		{"flag", "flag", false},
		{"environment", "environment", false},
		{"manifest", "manifest", false},
		{"flag with relative config", "flag", true},
		{"environment with relative config", "environment", true},
		{"manifest with relative config", "manifest", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := isolateResolve(t)
			if tc.relativeConfig {
				t.Setenv("XDG_CONFIG_HOME", "relative-config")
				t.Setenv("HOME", "relative-home")
			}
			writeConfig(t, os.Getenv("XDG_CONFIG_HOME"), `{broken`)
			var flag, manifest string
			switch tc.selector {
			case "flag":
				flag = base
			case "environment":
				t.Setenv("SKILLS_RECONCILE_WORKSPACE", base)
			case "manifest":
				manifest = filepath.Join(base, "skills-manifest.json")
			}
			if root, _, err := Resolve(flag, manifest); err != nil || root != base {
				t.Fatalf("selected workspace = %q, error = %v; want %q without reading config", root, err, base)
			}
		})
	}
}

func TestResolveRejectsRelativeConfigDirectory(t *testing.T) {
	for _, tc := range []struct{ name, variable, value, content string }{
		{"XDG config exists", "XDG_CONFIG_HOME", "config", "valid"},
		{"HOME config exists", "HOME", "home", "valid"},
		{"XDG config missing", "XDG_CONFIG_HOME", "missing", ""},
		{"HOME config missing", "HOME", "missing", ""},
		{"dot directory before JSON decode", "XDG_CONFIG_HOME", ".", `{broken`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := isolateResolve(t)
			t.Setenv("XDG_CONFIG_HOME", "")
			t.Setenv(tc.variable, tc.value)
			configDir := tc.value
			if tc.variable == "HOME" {
				configDir = filepath.Join(configDir, ".config")
			}
			content := tc.content
			if content == "valid" {
				content = fmt.Sprintf(`{"workspace":%q}`, base)
			}
			if content != "" {
				writeConfig(t, configDir, content)
			}

			root, manifest, err := Resolve("", "")
			if err == nil || err.Error() != "config directory must be absolute" || root != "" || manifest != "" {
				t.Fatalf("relative config directory resolved (%q, %q), error = %v; want rejection before reading config", root, manifest, err)
			}
		})
	}
}

func TestResolveAbsoluteXDGConfigIgnoresRelativeHOME(t *testing.T) {
	base := isolateResolve(t)
	t.Setenv("HOME", "relative-home")
	want := filepath.Join(base, "configured-workspace")
	makeDir(t, want)
	writeConfig(t, os.Getenv("XDG_CONFIG_HOME"), fmt.Sprintf(`{"workspace":%q}`, want))

	root, manifest, err := Resolve("", "")
	if err != nil || root != want || manifest != filepath.Join(want, "skills-manifest.json") {
		t.Fatalf("absolute XDG config resolved (%q, %q), error = %v; want workspace %q without consulting HOME", root, manifest, err, want)
	}
}

func TestResolveRejectsRelativeWorkspaceWithoutFallback(t *testing.T) {
	for _, selector := range []string{"flag", "environment", "config"} {
		t.Run(selector, func(t *testing.T) {
			base := isolateResolve(t)
			writeConfig(t, os.Getenv("XDG_CONFIG_HOME"), fmt.Sprintf(`{"workspace":%q}`, base))
			var flag string
			switch selector {
			case "flag":
				flag = "relative"
				t.Setenv("SKILLS_RECONCILE_WORKSPACE", base)
			case "environment":
				t.Setenv("SKILLS_RECONCILE_WORKSPACE", "relative")
			case "config":
				writeConfig(t, os.Getenv("XDG_CONFIG_HOME"), `{"workspace":"relative"}`)
			}
			root, manifest, err := Resolve(flag, "")
			if err == nil || !strings.Contains(err.Error(), "configured workspace must be absolute") || root != "" || manifest != "" {
				t.Fatalf("relative %s resolved (%q, %q), error = %v; want absolute-path rejection", selector, root, manifest, err)
			}
		})
	}
}

func TestResolveRejectsInvalidConfig(t *testing.T) {
	for _, tc := range []struct{ name, content string }{
		{"empty file", ""},
		{"malformed JSON", `{`},
		{"wrong workspace type", `{"workspace":42}`},
		{"trailing JSON", `{} {}`},
		{"trailing non-JSON data", `{} trailing`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateResolve(t)
			writeConfig(t, os.Getenv("XDG_CONFIG_HOME"), tc.content)
			root, manifest, err := Resolve("", "")
			if err == nil || err.Error() != "invalid skills-reconcile config" || root != "" || manifest != "" {
				t.Fatalf("invalid config resolved (%q, %q), error = %v", root, manifest, err)
			}
		})
	}
}

func TestResolveRejectsUnknownConfigFields(t *testing.T) {
	for _, tc := range []struct{ name, content string }{
		{"unknown field instead of workspace", `{"workpace":%q}`},
		{"unknown field alongside workspace", `{"workspace":%q,"unsupported":true}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := isolateResolve(t)
			writeConfig(t, os.Getenv("XDG_CONFIG_HOME"), fmt.Sprintf(tc.content, base))
			root, manifest, err := Resolve("", "")
			if err == nil || err.Error() != "invalid skills-reconcile config" || root != "" || manifest != "" {
				t.Fatalf("unknown config field resolved (%q, %q), error = %v; want invalid-config rejection without fallback", root, manifest, err)
			}
		})
	}
}

func TestResolveReportsConfigReadFailure(t *testing.T) {
	isolateResolve(t)
	// A directory fails ReadFile even when tests run as root in the container.
	makeDir(t, filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "skills-reconcile", "config.json"))
	root, manifest, err := Resolve("", "")
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) || errors.Is(err, os.ErrNotExist) || root != "" || manifest != "" {
		t.Fatalf("unreadable config resolved (%q, %q), error = %v; want filesystem error", root, manifest, err)
	}
}

func TestResolveCanonicalizesWorkspaceWithoutReadingOrCreatingManifest(t *testing.T) {
	base := isolateResolve(t)
	target := filepath.Join(base, "target")
	makeDir(t, target)
	link := filepath.Join(base, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	for _, content := range []string{"", "not a manifest"} {
		manifestPath := filepath.Join(target, "skills-manifest.json")
		if content != "" {
			if err := os.WriteFile(manifestPath, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		root, manifest, err := Resolve(link+"/./", "")
		if err != nil || root != target || manifest != manifestPath {
			t.Fatalf("resolved (%q, %q), error = %v; want (%q, %q)", root, manifest, err, target, manifestPath)
		}
		data, readErr := os.ReadFile(manifestPath)
		if content == "" {
			if !errors.Is(readErr, os.ErrNotExist) {
				t.Fatalf("resolver created manifest: %v", readErr)
			}
		} else if readErr != nil || string(data) != content {
			t.Fatalf("resolver changed manifest: content = %q, error = %v", data, readErr)
		}
	}
}

func TestResolveExplicitManifestOverridesWorkspaceAndResolvesOnlyParent(t *testing.T) {
	base := isolateResolve(t)
	parent := filepath.Join(base, "parent")
	makeDir(t, parent)
	if err := os.Symlink(parent, filepath.Join(base, "alias")); err != nil {
		t.Fatal(err)
	}
	// A dangling manifest symlink must not be followed by the locator.
	if err := os.Symlink("missing.json", filepath.Join(parent, "inventory.json")); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"alias/inventory.json", filepath.Join(base, "alias", "inventory.json")} {
		root, manifest, err := Resolve("invalid-relative-workspace", path)
		want := filepath.Join(parent, "inventory.json")
		if err != nil || root != parent || manifest != want {
			t.Fatalf("manifest %q resolved (%q, %q), error = %v; want (%q, %q)", path, root, manifest, err, parent, want)
		}
	}
}

func TestResolveRejectsMissingWorkspaceOrManifestParent(t *testing.T) {
	base := isolateResolve(t)
	missing := filepath.Join(base, "missing")
	for _, tc := range []struct{ name, flag, manifest string }{
		{"workspace", missing, ""},
		{"manifest parent", "", filepath.Join(missing, "inventory.json")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, manifest, err := Resolve(tc.flag, tc.manifest)
			if !errors.Is(err, os.ErrNotExist) || root != "" || manifest != "" {
				t.Fatalf("missing %s resolved (%q, %q), error = %v", tc.name, root, manifest, err)
			}
			if _, err := os.Stat(missing); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("resolver created missing directory: %v", err)
			}
		})
	}
}

func TestResolveDoesNotSearchParentsOrReadOldNamespace(t *testing.T) {
	base := isolateResolve(t)
	if err := os.WriteFile(filepath.Join(base, "skills-manifest.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(base, "child")
	makeDir(t, child)
	t.Chdir(child)
	t.Setenv("SKILLS_SYNC_WORKSPACE", base)
	oldConfig := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "skills-sync")
	makeDir(t, oldConfig)
	if err := os.WriteFile(filepath.Join(oldConfig, "config.json"), []byte(`{broken`), 0o600); err != nil {
		t.Fatal(err)
	}
	root, manifest, err := Resolve("", "")
	if err != nil || root != child || manifest != filepath.Join(child, "skills-manifest.json") {
		t.Fatalf("resolved (%q, %q), error = %v; want cwd %q without legacy lookup or parent search", root, manifest, err, child)
	}
}

func isolateResolve(t *testing.T) string {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for key, dir := range map[string]string{
		"HOME": "home", "XDG_CONFIG_HOME": "xdg", "XDG_STATE_HOME": "state", "XDG_CACHE_HOME": "cache",
	} {
		t.Setenv(key, filepath.Join(base, dir))
	}
	t.Setenv("SKILLS_RECONCILE_WORKSPACE", "")
	t.Setenv("SKILLS_SYNC_WORKSPACE", "")
	t.Chdir(base)
	return base
}

func makeDir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatal(err)
	}
}

func writeConfig(t *testing.T, configHome, content string) {
	t.Helper()
	dir := filepath.Join(configHome, "skills-reconcile")
	makeDir(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
