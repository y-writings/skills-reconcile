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

			location, err := Resolve(Overrides{WorkspaceDir: paths[tc.flag]})
			if err != nil {
				t.Fatal(err)
			}
			want := paths[tc.want]
			if location.WorkspaceDir != want || location.ManifestPath != filepath.Join(want, "skills-manifest.json") {
				t.Fatalf("resolved (%q, %q), want workspace %q and its default manifest", location.WorkspaceDir, location.ManifestPath, want)
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

			location, err := Resolve(Overrides{})
			want := base
			if tc.wantHome {
				want = homeWorkspace
			}
			if err != nil || location.WorkspaceDir != want {
				t.Fatalf("workspace = %q, error = %v; want %q", location.WorkspaceDir, err, want)
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
		{"flag with relative config", "flag", true},
		{"environment with relative config", "environment", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := isolateResolve(t)
			if tc.relativeConfig {
				t.Setenv("XDG_CONFIG_HOME", "relative-config")
				t.Setenv("HOME", "relative-home")
			}
			writeConfig(t, os.Getenv("XDG_CONFIG_HOME"), `{broken`)
			var flag string
			switch tc.selector {
			case "flag":
				flag = base
			case "environment":
				t.Setenv("SKILLS_RECONCILE_WORKSPACE", base)
			}
			if location, err := Resolve(Overrides{WorkspaceDir: flag}); err != nil || location.WorkspaceDir != base {
				t.Fatalf("selected workspace = %q, error = %v; want %q without reading config", location.WorkspaceDir, err, base)
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

			location, err := Resolve(Overrides{})
			if err == nil || err.Error() != "config directory must be absolute" || location != (Location{}) {
				t.Fatalf("relative config directory resolved (%q, %q), error = %v; want rejection before reading config", location.WorkspaceDir, location.ManifestPath, err)
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

	location, err := Resolve(Overrides{})
	if err != nil || location.WorkspaceDir != want || location.ManifestPath != filepath.Join(want, "skills-manifest.json") {
		t.Fatalf("absolute XDG config resolved (%q, %q), error = %v; want workspace %q without consulting HOME", location.WorkspaceDir, location.ManifestPath, err, want)
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
			location, err := Resolve(Overrides{WorkspaceDir: flag})
			if err == nil || !strings.Contains(err.Error(), "configured workspace must be absolute") || location != (Location{}) {
				t.Fatalf("relative %s resolved (%q, %q), error = %v; want absolute-path rejection", selector, location.WorkspaceDir, location.ManifestPath, err)
			}
		})
	}
}

func TestResolveRejectsDuplicateWorkspaceConfig(t *testing.T) {
	base := isolateResolve(t)
	writeConfig(t, os.Getenv("XDG_CONFIG_HOME"), fmt.Sprintf(`{"workspace":%q,"workspace":""}`, base))

	location, err := Resolve(Overrides{})
	if err == nil || err.Error() != "invalid skills-reconcile config" || location != (Location{}) {
		t.Fatalf("duplicate workspace config resolved (%q, %q), error = %v; want invalid-config rejection without fallback", location.WorkspaceDir, location.ManifestPath, err)
	}
}

func TestResolveReportsConfigReadFailure(t *testing.T) {
	isolateResolve(t)
	// A directory fails ReadFile even when tests run as root in the container.
	makeDir(t, filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "skills-reconcile", "config.json"))
	location, err := Resolve(Overrides{})
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) || errors.Is(err, os.ErrNotExist) || location != (Location{}) {
		t.Fatalf("unreadable config resolved (%q, %q), error = %v; want filesystem error", location.WorkspaceDir, location.ManifestPath, err)
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
		location, err := Resolve(Overrides{WorkspaceDir: link + "/./"})
		if err != nil || location.WorkspaceDir != target || location.ManifestPath != manifestPath {
			t.Fatalf("resolved (%q, %q), error = %v; want (%q, %q)", location.WorkspaceDir, location.ManifestPath, err, target, manifestPath)
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

func TestResolvePreservesSymlinkAwareParentTraversal(t *testing.T) {
	base := isolateResolve(t)
	parent := filepath.Join(base, "parent")
	target := filepath.Join(parent, "target")
	makeDir(t, target)
	link := filepath.Join(base, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	location, err := Resolve(Overrides{WorkspaceDir: link + string(filepath.Separator) + ".."})
	if err != nil || location.WorkspaceDir != parent {
		t.Fatalf("workspace = %q, error = %v; want symlink-aware parent %q", location.WorkspaceDir, err, parent)
	}
}

func TestResolveRejectsNonDirectoryWithoutFallback(t *testing.T) {
	for _, tc := range []struct {
		name, selector string
		symlink        bool
	}{
		{"workspace file", "workspace", false},
		{"environment file", "environment", false},
		{"config file", "config", false},
		{"workspace file symlink", "workspace", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := isolateResolve(t)
			path := filepath.Join(base, "file")
			if err := os.WriteFile(path, []byte("synthetic"), 0o600); err != nil {
				t.Fatal(err)
			}
			if tc.symlink {
				link := filepath.Join(base, "link")
				if err := os.Symlink(path, link); err != nil {
					t.Fatal(err)
				}
				path = link
			}
			writeConfig(t, os.Getenv("XDG_CONFIG_HOME"), fmt.Sprintf(`{"workspace":%q}`, base))
			var overrides Overrides
			switch tc.selector {
			case "workspace":
				overrides.WorkspaceDir = path
				t.Setenv("SKILLS_RECONCILE_WORKSPACE", base)
			case "environment":
				t.Setenv("SKILLS_RECONCILE_WORKSPACE", path)
			case "config":
				writeConfig(t, os.Getenv("XDG_CONFIG_HOME"), fmt.Sprintf(`{"workspace":%q}`, path))
			}
			location, err := Resolve(overrides)
			if err == nil || !strings.Contains(err.Error(), "workspace must be a directory") || location != (Location{}) {
				t.Fatalf("non-directory resolved %+v, error = %v; want directory rejection without fallback", location, err)
			}
		})
	}
}

func TestResolveRejectsMissingWorkspace(t *testing.T) {
	base := isolateResolve(t)
	missing := filepath.Join(base, "missing")
	location, err := Resolve(Overrides{WorkspaceDir: missing})
	if !errors.Is(err, os.ErrNotExist) || location != (Location{}) {
		t.Fatalf("missing workspace resolved (%q, %q), error = %v", location.WorkspaceDir, location.ManifestPath, err)
	}
	if _, err := os.Stat(missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("resolver created missing directory: %v", err)
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
	location, err := Resolve(Overrides{})
	if err != nil || location.WorkspaceDir != child || location.ManifestPath != filepath.Join(child, "skills-manifest.json") {
		t.Fatalf("resolved (%q, %q), error = %v; want cwd %q without legacy lookup or parent search", location.WorkspaceDir, location.ManifestPath, err, child)
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
