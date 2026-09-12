package lock

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvePath(t *testing.T) {
	for _, tc := range []struct {
		name, xdgStateHome, home, want, wantError string
	}{
		{"absolute XDG state ignores relative HOME", "/state", "relative-home", filepath.Join("/state", "skills", ".skill-lock.json"), ""},
		{"relative XDG state does not fall back to HOME", "relative-state", "/home", "", "global lock directory must be absolute"},
		{"HOME fallback", "", "/home", filepath.Join("/home", ".agents", ".skill-lock.json"), ""},
		{"relative HOME", "", "relative-home", "", "global lock directory must be absolute"},
		{"missing HOME", "", "", "", "HOME is not set"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolvePath(tc.xdgStateHome, tc.home)
			if got != tc.want || (err != nil && err.Error() != tc.wantError) || (err == nil && tc.wantError != "") {
				t.Fatalf("ResolvePath() = (%q, %v), want (%q, %q)", got, err, tc.want, tc.wantError)
			}
		})
	}
}

func TestPathReadsStateEnvironment(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/state")
	t.Setenv("HOME", "/home")

	got, err := Path()
	want := filepath.Join("/state", "skills", ".skill-lock.json")
	if err != nil || got != want {
		t.Fatalf("Path() = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestReadObservedDistinguishesMissingFromInvalid(t *testing.T) {
	dir := t.TempDir()
	missingPath := filepath.Join(dir, "missing.json")

	observed, err := ReadObserved(missingPath)
	if err != nil || observed == nil || !observed.Missing || observed.Version != 3 || len(observed.Skills) != 0 {
		t.Fatalf("ReadObserved(missing) = (%#v, %v), want marked empty v3 lock", observed, err)
	}

	for _, tc := range []struct {
		name, content, wantError string
	}{
		{"malformed JSON", `{`, "decode global lock"},
		{"trailing JSON", `{"version":3,"skills":{}} {}`, "trailing JSON"},
		{"unsupported version", `{"version":2,"skills":{}}`, "unsupported global lock version 2"},
		{"missing skills", `{"version":3}`, "global lock has no skills object"},
		{"null skills", `{"version":3,"skills":null}`, "global lock has no skills object"},
		{"empty name", `{"version":3,"skills":{"":{"source":"org/repo","sourceType":"github"}}}`, `invalid lock entry ""`},
		{"empty source", `{"version":3,"skills":{"x":{"source":"","sourceType":"github"}}}`, `invalid lock entry "x"`},
		{"empty source type", `{"version":3,"skills":{"x":{"source":"org/repo","sourceType":""}}}`, `invalid lock entry "x"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, strings.ReplaceAll(tc.name, " ", "-")+".json")
			if err := os.WriteFile(path, []byte(tc.content), 0o600); err != nil {
				t.Fatal(err)
			}

			got, err := ReadObserved(path)
			if err == nil || !strings.Contains(err.Error(), tc.wantError) || got != nil {
				t.Fatalf("ReadObserved() = (%#v, %v), want error containing %q", got, err, tc.wantError)
			}
		})
	}

	directoryPath := filepath.Join(dir, "directory")
	if err := os.Mkdir(directoryPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if got, err := ReadObserved(directoryPath); err == nil || errors.Is(err, os.ErrNotExist) || got != nil {
		t.Fatalf("ReadObserved(directory) = (%#v, %v), want filesystem error", got, err)
	}
}

func TestReadObservedReadsValidLockAndFutureFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock.json")
	content := `{"version":3,"skills":{"x":{"source":"example.com","sourceType":"well-known","sourceUrl":"https://cdn.example.com/x.tgz","sourceBaseUrl":"https://example.com/skills","ref":"v1","skillPath":"skills/x/SKILL.md","agents":["codex"]}},"futureField":true}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	observed, err := ReadObserved(path)
	if err != nil {
		t.Fatal(err)
	}
	if observed == nil {
		t.Fatal("ReadObserved() returned a nil lock without an error")
	}
	entry := observed.Skills["x"]
	if observed.Missing || entry.SourceBaseURL != "https://example.com/skills" || entry.Ref != "v1" || entry.SkillPath != "skills/x/SKILL.md" || len(entry.Agents) != 1 || entry.Agents[0] != "codex" {
		t.Fatalf("ReadObserved() = %#v, want decoded existing lock", observed)
	}
}

func TestReadObservedDistinguishesValidAndDanglingSymlinks(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.json")
	if err := os.WriteFile(target, []byte(`{"version":3,"skills":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	validLink := filepath.Join(dir, "valid-link.json")
	if err := os.Symlink(target, validLink); err != nil {
		t.Fatal(err)
	}
	if observed, err := ReadObserved(validLink); err != nil || observed.Missing {
		t.Fatalf("ReadObserved(valid symlink) = (%#v, %v), want existing lock", observed, err)
	}

	danglingLink := filepath.Join(dir, "dangling-link.json")
	if err := os.Symlink(filepath.Join(dir, "absent.json"), danglingLink); err != nil {
		t.Fatal(err)
	}
	if observed, err := ReadObserved(danglingLink); err == nil || !strings.Contains(err.Error(), "dangling symlink") || observed != nil {
		t.Fatalf("ReadObserved(dangling symlink) = (%#v, %v), want dangling-symlink error", observed, err)
	}
}

func TestInstallSourceSupportsOnlyRestorableSources(t *testing.T) {
	for _, tc := range []struct {
		name  string
		entry Entry
		want  string
		ok    bool
	}{
		{"GitHub", Entry{Source: "org/repo", SourceType: "github"}, "org/repo", true},
		{"Git", Entry{Source: "repo", SourceType: "git", SourceURL: "git@example.com:repo.git"}, "git@example.com:repo.git", true},
		{"GitLab", Entry{Source: "repo", SourceType: "gitlab", SourceURL: "https://gitlab.com/org/repo.git"}, "https://gitlab.com/org/repo.git", true},
		{"well-known base URL", Entry{SourceType: "well-known", SourceURL: "https://wrong.example.com/.well-known/skills/x/SKILL.md", SourceBaseURL: "https://example.com/skills"}, "https://example.com/skills", true},
		{"well-known URL fallback", Entry{SourceType: "well-known", SourceURL: "https://example.com/.well-known/skills/x/SKILL.md"}, "https://example.com", true},
		{"well-known without origin", Entry{SourceType: "well-known", SourceURL: "https://cdn.example.com/x.tgz"}, "", false},
		{"local", Entry{Source: "/tmp/skill", SourceType: "local"}, "", false},
		{"node modules", Entry{Source: "pkg", SourceType: "node_modules"}, "", false},
		{"unknown", Entry{Source: "future", SourceType: "future-type"}, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := tc.entry.InstallSource()
			if got != tc.want || ok != tc.ok {
				t.Fatalf("InstallSource() = (%q, %t), want (%q, %t)", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestMatchesSourceUsesGitHubRepositoryIdentity(t *testing.T) {
	entry := Entry{Source: "org/repo", SourceType: "github"}
	for _, source := range []string{
		"org/repo",
		"github:org/repo",
		"https://github.com/org/repo.git",
		"git@github.com:org/repo.git",
		"ssh://git@github.com/org/repo.git",
	} {
		if !entry.MatchesSource(source) {
			t.Errorf("did not match equivalent GitHub source %q", source)
		}
	}
	for _, source := range []string{
		"other/repo",
		"https://github.com/org/repo?mirror=other",
		"https://github.com/org/repo#v1",
		"ssh://git@github.com:2222/org/repo.git",
		"https://GITHUB.COM/org/repo",
		"https://github.com/%6Frg/repo",
	} {
		if entry.MatchesSource(source) {
			t.Errorf("matched non-equivalent GitHub source %q", source)
		}
	}
}

func TestMatchesSourceUsesGitLabRepositoryIdentity(t *testing.T) {
	entry := Entry{SourceURL: "https://gitlab.com/org/group/repo.git", SourceType: "gitlab"}
	for _, source := range []string{
		"https://gitlab.com/org/group/repo",
		"http://gitlab.com/org/group/repo/",
		"ssh://git@gitlab.com/org/group/repo.git",
		"gitlab.com/org/group/repo",
		"gitlab:org/group/repo",
	} {
		if !entry.MatchesSource(source) {
			t.Errorf("did not match equivalent GitLab source %q", source)
		}
	}
	for _, source := range []string{
		"https://gitlab.com/other/group/repo",
		"https://gitlab.com:8443/org/group/repo",
		"https://GITLAB.COM/org/group/repo",
		"https://gitlab.com/%6Frg/group/repo",
		"git@gitlab.com:org/group/repo.git",
	} {
		if entry.MatchesSource(source) {
			t.Errorf("matched non-equivalent GitLab source %q", source)
		}
	}
}

func TestMatchesSourceDoesNotNormalizeGenericOrUnrestorableSources(t *testing.T) {
	generic := Entry{SourceURL: "https://example.com/org/repo.git", SourceType: "git"}
	if !generic.MatchesSource(generic.SourceURL) {
		t.Fatal("generic Git source did not match exactly")
	}
	if generic.MatchesSource("https://example.com/org/repo") {
		t.Fatal("generic Git source was normalized")
	}
	if (Entry{Source: "/tmp/skill", SourceType: "local"}).MatchesSource("/tmp/skill") {
		t.Fatal("unrestorable local source matched")
	}
}
