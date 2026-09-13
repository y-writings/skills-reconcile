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
		{"invalid UTF-8", "{\"version\":3,\"skills\":{\"x\":{\"source\":\"org/repo\xff\",\"sourceType\":\"github\"}}}", "decode global lock: invalid UTF-8"},
		{"trailing JSON", `{"version":3,"skills":{}} {}`, "decode global lock: trailing JSON"},
		{"escaped duplicate in future array", `{"version":3,"skills":{},"future":[{"x":1,"\u0078":2}]}`, `decode global lock: duplicate JSON field "x"`},
		{"uppercase top-level field", `{"VERSION":3,"skills":{}}`, `non-canonical JSON field "VERSION"`},
		{"mixed-case provenance field", `{"version":3,"skills":{"x":{"source":"repo","sourceType":"git","SourceUrl":"https://example.com/repo.git"}}}`, `non-canonical JSON field "SourceUrl"`},
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
	if got, err := ReadObserved(directoryPath); err == nil || !strings.Contains(err.Error(), "not a regular file") ||
		errors.Is(err, os.ErrNotExist) || got != nil {
		t.Fatalf("ReadObserved(directory) = (%#v, %v), want non-regular-file error", got, err)
	}
}

func TestReadObservedReadsValidLockAndFutureFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock.json")
	content := `{"version":3,"skills":{"x":{"source":"example.com","sourceType":"well-known","sourceUrl":"https://cdn.example.com/x.tgz","sourceBaseUrl":"https://example.com/skills","ref":"v1","skillPath":"skills/x/SKILL.md","agents":["codex"],"futureEntryField":true},"y":{"source":"org/repo","sourceType":"github"}},"futureField":{"number":1e1000}}`
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
	if observed.Missing || len(observed.Skills) != 2 || entry.SourceBaseURL != "https://example.com/skills" || entry.Ref != "v1" || entry.SkillPath != "skills/x/SKILL.md" || len(entry.Agents) != 1 || entry.Agents[0] != "codex" {
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

	targetDir := filepath.Join(dir, "target-directory")
	if err := os.Mkdir(targetDir, 0o700); err != nil {
		t.Fatal(err)
	}
	validAncestor := filepath.Join(dir, "valid-directory-link")
	if err := os.Symlink(targetDir, validAncestor); err != nil {
		t.Fatal(err)
	}
	if observed, err := ReadObserved(filepath.Join(validAncestor, "missing.json")); err != nil || !observed.Missing {
		t.Fatalf("ReadObserved(valid symlink ancestor) = (%#v, %v), want missing lock", observed, err)
	}

	danglingAncestor := filepath.Join(dir, "dangling-directory-link")
	if err := os.Symlink(filepath.Join(dir, "absent-directory"), danglingAncestor); err != nil {
		t.Fatal(err)
	}
	descendant := filepath.Join(danglingAncestor, "skills", ".skill-lock.json")
	if observed, err := ReadObserved(descendant); err == nil || !strings.Contains(err.Error(), "dangling symlink") || observed != nil {
		t.Fatalf("ReadObserved(dangling symlink ancestor) = (%#v, %v), want dangling-symlink error", observed, err)
	}

	missingAncestor := filepath.Join(dir, "missing-directory", ".skill-lock.json")
	if observed, err := ReadObserved(missingAncestor); err != nil || !observed.Missing {
		t.Fatalf("ReadObserved(missing ancestor) = (%#v, %v), want missing lock", observed, err)
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
		{"GitHub repository punctuation", Entry{Source: "Org/repo_name.test-1", SourceType: "github"}, "Org/repo_name.test-1", true},
		{"GitHub NUL control character", Entry{Source: "org/repo\x00", SourceType: "github"}, "", false},
		{"GitHub shorthand query", Entry{Source: "org/repo?mirror=other", SourceType: "github"}, "", false},
		{"GitHub prefix fragment", Entry{Source: "github:org/repo#v1", SourceType: "github"}, "", false},
		{"GitHub shorthand skill selector", Entry{Source: "org/repo@skill", SourceType: "github"}, "", false},
		{"GitHub prefix ref selector", Entry{Source: "github:org/repo:ref", SourceType: "github"}, "", false},
		{"GitHub encoded path separator", Entry{Source: "org/repo%2Fother", SourceType: "github"}, "", false},
		{"GitHub dot repository", Entry{Source: "org/..", SourceType: "github"}, "", false},
		{"GitHub unsupported URL scheme", Entry{Source: "file://github.com/org/repo", SourceType: "github"}, "", false},
		{"GitHub URL credentials", Entry{Source: "https://person@github.com/org/repo", SourceType: "github"}, "", false},
		{"Git", Entry{Source: "repo", SourceType: "git", SourceURL: "git@example.com:repo.git"}, "git@example.com:repo.git", true},
		{"Git SCP without username", Entry{Source: "repo", SourceType: "git", SourceURL: "example.com:repo.git"}, "example.com:repo.git", true},
		{"Git SCP nested path", Entry{Source: "repo", SourceType: "git", SourceURL: "git@example.com:mirrors/example.org/org/repo.git"}, "git@example.com:mirrors/example.org/org/repo.git", true},
		{"Git SCP reclassified as GitHub", Entry{Source: "repo", SourceType: "git", SourceURL: "git@example.com:mirrors/github.com/org/repo.git"}, "", false},
		{"Git SCP reclassified as GitLab", Entry{Source: "repo", SourceType: "git", SourceURL: "git@example.com:mirrors/gitlab.com/org/repo.git"}, "", false},
		{"Git SCP with empty username", Entry{Source: "repo", SourceType: "git", SourceURL: "@example.com:repo.git"}, "", false},
		{"Git SCP with option-like authority", Entry{Source: "repo", SourceType: "git", SourceURL: "-example.com:repo.git"}, "", false},
		{"Git SCP with option-like username", Entry{Source: "repo", SourceType: "git", SourceURL: "-git@example.com:repo.git"}, "", false},
		{"Git SCP with bracketed option-like host", Entry{Source: "repo", SourceType: "git", SourceURL: "[-example.com]:repo.git"}, "", false},
		{"Git SCP with option-like host after username", Entry{Source: "repo", SourceType: "git", SourceURL: "git@-example.com:repo.git"}, "", false},
		{"Git SCP with empty bracketed host", Entry{Source: "repo", SourceType: "git", SourceURL: "[]:repo.git"}, "", false},
		{"Git SCP with bracketed IPv6 host", Entry{Source: "repo", SourceType: "git", SourceURL: "[::1]:repo.git"}, "[::1]:repo.git", true},
		{"Git SCP with username and bracketed IPv6 host", Entry{Source: "repo", SourceType: "git", SourceURL: "git@[::1]:repo.git"}, "git@[::1]:repo.git", true},
		{"Git SCP with bracketed IPv6 host and separators-only path", Entry{Source: "repo", SourceType: "git", SourceURL: "[::1]:/"}, "", false},
		{"Git SCP without path after bracketed IPv6 host", Entry{Source: "repo", SourceType: "git", SourceURL: "[::1]suffix"}, "", false},
		{"Git SCP with bracketed IPv6 host and suffixed separators-only path", Entry{Source: "repo", SourceType: "git", SourceURL: "[::1]suffix:/"}, "", false},
		{"Git SCP absolute path", Entry{Source: "repo", SourceType: "git", SourceURL: "git@example.com:/repo.git"}, "git@example.com:/repo.git", true},
		{"Git SCP absolute option-like path", Entry{Source: "repo", SourceType: "git", SourceURL: "example.com:/-repo.git"}, "example.com:/-repo.git", true},
		{"Git SCP path with separators only", Entry{Source: "repo", SourceType: "git", SourceURL: "git@example.com:/"}, "", false},
		{"Git SCP option-like relative path", Entry{Source: "repo", SourceType: "git", SourceURL: "example.com:-repo.git"}, "", false},
		{"Git remote-helper syntax", Entry{Source: "repo", SourceType: "git", SourceURL: "ext::/usr/bin/false"}, "", false},
		{"Git SCP NUL control character", Entry{Source: "repo", SourceType: "git", SourceURL: "git@example.com:repo.git\x00"}, "", false},
		{"Git SCP authority control character", Entry{Source: "repo", SourceType: "git", SourceURL: "git\x1b@example.com:repo.git"}, "", false},
		{"Git SCP reserved provider host", Entry{Source: "repo", SourceType: "git", SourceURL: "git@github.com:org/repo.git"}, "", false},
		{"Git HTTPS", Entry{Source: "repo", SourceType: "git", SourceURL: "https://example.com/repo.git"}, "https://example.com/repo.git", true},
		{"Git HTTPS with maximum port", Entry{Source: "repo", SourceType: "git", SourceURL: "https://example.com:65535/repo.git"}, "https://example.com:65535/repo.git", true},
		{"Git HTTPS with port zero", Entry{Source: "repo", SourceType: "git", SourceURL: "https://example.com:0/repo.git"}, "", false},
		{"Git HTTPS out-of-range port", Entry{Source: "repo", SourceType: "git", SourceURL: "https://example.com:65536/repo.git"}, "", false},
		{"Git SSH", Entry{Source: "repo", SourceType: "git", SourceURL: "ssh://git@example.com/repo"}, "ssh://git@example.com/repo", true},
		{"Git SSH with option-like username", Entry{Source: "repo", SourceType: "git", SourceURL: "ssh://-git@example.com/repo"}, "", false},
		{"Git SSH with option-like host", Entry{Source: "repo", SourceType: "git", SourceURL: "ssh://git@-example.com/repo"}, "", false},
		{"Git SSH URL with encoded separator and repository", Entry{Source: "repo", SourceType: "git", SourceURL: "ssh://git@example.com/%2Frepo"}, "ssh://git@example.com/%2Frepo", true},
		{"Git protocol", Entry{Source: "repo", SourceType: "git", SourceURL: "git://example.com/repo"}, "git://example.com/repo", true},
		{"Git local file URL", Entry{Source: "repo", SourceType: "git", SourceURL: "file:///tmp/repo"}, "", false},
		{"Git URL credentials", Entry{Source: "repo", SourceType: "git", SourceURL: "https://person@example.com/repo.git"}, "", false},
		{"Git unsupported URL scheme", Entry{Source: "repo", SourceType: "git", SourceURL: "ftp://example.com/repo.git"}, "", false},
		{"Git URL fragment", Entry{Source: "repo", SourceType: "git", SourceURL: "https://example.com/repo.git#main"}, "", false},
		{"Git SSH URL without repository", Entry{Source: "repo", SourceType: "git", SourceURL: "ssh://git@example.com"}, "", false},
		{"Git SSH URL with encoded separators only", Entry{Source: "repo", SourceType: "git", SourceURL: "ssh://git@example.com/%2F"}, "", false},
		{"Git protocol URL without repository", Entry{Source: "repo", SourceType: "git", SourceURL: "git://example.com"}, "", false},
		{"Git protocol URL with encoded separators only", Entry{Source: "repo", SourceType: "git", SourceURL: "git://example.com/%2F"}, "", false},
		{"Git URL reclassified as well-known", Entry{Source: "repo", SourceType: "git", SourceURL: "https://example.com/repo"}, "", false},
		{"Git URL reclassified as GitHub", Entry{Source: "repo", SourceType: "git", SourceURL: "https://github.com/org/repo.git"}, "", false},
		{"Git URL with noncanonical GitHub host", Entry{Source: "repo", SourceType: "git", SourceURL: "https://GitHub.com/org/repo.git"}, "", false},
		{"Git URL with noncanonical GitLab host", Entry{Source: "repo", SourceType: "git", SourceURL: "https://GitLab.com/org/repo.git"}, "", false},
		{"Git URL reclassified as hosted artifact", Entry{Source: "repo", SourceType: "git", SourceURL: "https://codeload.github.com/org/repo.git"}, "", false},
		{"Git URL reclassified as self-hosted GitLab", Entry{Source: "repo", SourceType: "git", SourceURL: "https://example.com/org/repo/-/tree/main.git"}, "", false},
		{"GitLab", Entry{Source: "repo", SourceType: "gitlab", SourceURL: "https://gitlab.com/org/repo.git"}, "https://gitlab.com/org/repo.git", true},
		{"GitLab nested repository punctuation", Entry{Source: "repo", SourceType: "gitlab", SourceURL: "gitlab:org-name/group_name/repo.test"}, "gitlab:org-name/group_name/repo.test", true},
		{"GitLab control character", Entry{Source: "repo", SourceType: "gitlab", SourceURL: "gitlab:org/repo\x1b"}, "", false},
		{"GitLab prefix query", Entry{Source: "repo", SourceType: "gitlab", SourceURL: "gitlab:org/repo?mirror=other"}, "", false},
		{"GitLab shorthand fragment", Entry{Source: "repo", SourceType: "gitlab", SourceURL: "gitlab.com/org/repo#v1"}, "", false},
		{"GitLab skill selector", Entry{Source: "repo", SourceType: "gitlab", SourceURL: "gitlab:org/repo@skill"}, "", false},
		{"GitLab ref selector", Entry{Source: "repo", SourceType: "gitlab", SourceURL: "gitlab:org/repo:ref"}, "", false},
		{"GitLab punctuation outside slug grammar", Entry{Source: "repo", SourceType: "gitlab", SourceURL: "gitlab:org/repo;mirror"}, "", false},
		{"GitLab dot path component", Entry{Source: "repo", SourceType: "gitlab", SourceURL: "gitlab:org/../repo"}, "", false},
		{"GitLab unsupported URL scheme", Entry{Source: "repo", SourceType: "gitlab", SourceURL: "file://gitlab.com/org/repo"}, "", false},
		{"GitLab URL credentials", Entry{Source: "repo", SourceType: "gitlab", SourceURL: "https://person@gitlab.com/org/repo"}, "", false},
		{"well-known base URL", Entry{SourceType: "well-known", SourceURL: "https://wrong.example.com/.well-known/skills/x/SKILL.md", SourceBaseURL: "https://example.com/skills"}, "https://example.com/skills", true},
		{"well-known base URL control character", Entry{SourceType: "well-known", SourceBaseURL: "https://example.com/skills\u0080"}, "", false},
		{"well-known HTTP base URL with port", Entry{SourceType: "well-known", SourceBaseURL: "http://localhost:8080/scope"}, "http://localhost:8080/scope", true},
		{"well-known base URL with out-of-range port", Entry{SourceType: "well-known", SourceBaseURL: "https://example.com:65536/scope"}, "", false},
		{"well-known uppercase scheme", Entry{SourceType: "well-known", SourceBaseURL: "HTTP://example.com/scope"}, "", false},
		{"well-known file base URL", Entry{SourceType: "well-known", SourceBaseURL: "file:///tmp/skill"}, "", false},
		{"well-known base URL without host", Entry{SourceType: "well-known", SourceBaseURL: "https:///skills"}, "", false},
		{"well-known base URL credentials", Entry{SourceType: "well-known", SourceBaseURL: "https://person@example.com/skills"}, "", false},
		{"well-known base URL query", Entry{SourceType: "well-known", SourceBaseURL: "https://example.com/skills?version=1"}, "", false},
		{"well-known base URL fragment", Entry{SourceType: "well-known", SourceBaseURL: "https://example.com/skills#v1"}, "", false},
		{"well-known base URL whitespace", Entry{SourceType: "well-known", SourceBaseURL: "https://example.com/path with space"}, "", false},
		{"well-known base URL backslash", Entry{SourceType: "well-known", SourceBaseURL: `https://example.com/path\scope`}, "", false},
		{"well-known hosted provider base URL", Entry{SourceType: "well-known", SourceBaseURL: "https://github.com/org/repo"}, "", false},
		{"well-known GitLab base URL", Entry{SourceType: "well-known", SourceBaseURL: "https://gitlab.com/org/repo"}, "", false},
		{"well-known raw artifact base URL", Entry{SourceType: "well-known", SourceBaseURL: "https://raw.githubusercontent.com/org/repo/main/SKILL.md"}, "", false},
		{"well-known codeload artifact base URL", Entry{SourceType: "well-known", SourceBaseURL: "https://codeload.github.com/org/repo/tar.gz/main"}, "", false},
		{"well-known objects artifact base URL", Entry{SourceType: "well-known", SourceBaseURL: "https://objects.githubusercontent.com/object"}, "", false},
		{"well-known Hugging Face base URL", Entry{SourceType: "well-known", SourceBaseURL: "https://huggingface.co/org/repo"}, "https://huggingface.co/org/repo", true},
		{"well-known embedded GitHub source", Entry{SourceType: "well-known", SourceBaseURL: "https://example.com/path/github.com/org/repo"}, "", false},
		{"well-known embedded GitLab source", Entry{SourceType: "well-known", SourceBaseURL: "https://example.com/path/gitlab.com/org/repo"}, "", false},
		{"well-known repository base URL", Entry{SourceType: "well-known", SourceBaseURL: "https://example.com/repo.git"}, "", false},
		{"well-known invalid base does not fall back", Entry{SourceType: "well-known", SourceURL: "https://example.com/.well-known/skills/x/SKILL.md", SourceBaseURL: "file:///tmp/skill"}, "", false},
		{"well-known skills URL fallback", Entry{SourceType: "well-known", SourceURL: "https://example.com/.well-known/skills/x/SKILL.md"}, "https://example.com", true},
		{"well-known hyphenated skill URL fallback", Entry{SourceType: "well-known", SourceURL: "https://example.com/.well-known/skills/x-y/SKILL.md"}, "https://example.com", true},
		{"well-known 64-character skill URL fallback", Entry{SourceType: "well-known", SourceURL: "https://example.com/.well-known/skills/" + strings.Repeat("a", 64) + "/SKILL.md"}, "https://example.com", true},
		{"well-known agent skills URL fallback with scope", Entry{SourceType: "well-known", SourceURL: "https://example.com:8443/scope/.well-known/agent-skills/x/SKILL.md"}, "https://example.com:8443/scope", true},
		{"well-known file URL fallback", Entry{SourceType: "well-known", SourceURL: "file:///tmp/.well-known/skills/x/SKILL.md"}, "", false},
		{"well-known URL fallback credentials", Entry{SourceType: "well-known", SourceURL: "https://person@example.com/.well-known/skills/x/SKILL.md"}, "", false},
		{"well-known URL fallback query", Entry{SourceType: "well-known", SourceURL: "https://example.com/.well-known/skills/x/SKILL.md?version=1"}, "", false},
		{"well-known URL fallback invalid name", Entry{SourceType: "well-known", SourceURL: "https://example.com/.well-known/skills/Bad_Name/SKILL.md"}, "", false},
		{"well-known URL fallback leading hyphen", Entry{SourceType: "well-known", SourceURL: "https://example.com/.well-known/skills/-x/SKILL.md"}, "", false},
		{"well-known URL fallback trailing hyphen", Entry{SourceType: "well-known", SourceURL: "https://example.com/.well-known/skills/x-/SKILL.md"}, "", false},
		{"well-known URL fallback double hyphen", Entry{SourceType: "well-known", SourceURL: "https://example.com/.well-known/skills/x--y/SKILL.md"}, "", false},
		{"well-known URL fallback 65-character name", Entry{SourceType: "well-known", SourceURL: "https://example.com/.well-known/skills/" + strings.Repeat("a", 65) + "/SKILL.md"}, "", false},
		{"well-known URL fallback embedded GitHub source", Entry{SourceType: "well-known", SourceURL: "https://example.com/path/github.com/org/repo/.well-known/skills/x/SKILL.md"}, "", false},
		{"well-known URL fallback missing SKILL file", Entry{SourceType: "well-known", SourceURL: "https://example.com/.well-known/skills/x"}, "", false},
		{"well-known URL fallback extra tail", Entry{SourceType: "well-known", SourceURL: "https://example.com/.well-known/skills/x/SKILL.md/archive"}, "", false},
		{"well-known empty URL fallback origin", Entry{SourceType: "well-known", SourceURL: "/.well-known/skills/x/SKILL.md"}, "", false},
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
		"http://github.com/org/repo/",
		"https://github.com/org/repo.git",
		"git@github.com:org/repo.git",
		"ssh://git@github.com/org/repo.git",
		"ssh://alice@github.com/org/repo.git",
	} {
		if !entry.MatchesSource(source) {
			t.Errorf("did not match equivalent GitHub source %q", source)
		}
	}
	for _, source := range []string{
		"other/repo",
		"file://github.com/org/repo",
		"//github.com/org/repo",
		"https://github.com/org/repo?mirror=other",
		"https://github.com/org/repo#v1",
		"ssh://git@github.com:2222/org/repo.git",
		"https://GITHUB.COM/org/repo",
		"https://github.com/%6Frg/repo",
		"https://person@github.com/org/repo",
		"ssh://git:password@github.com/org/repo.git",
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
		"ssh://deploy@gitlab.com/org/group/repo.git",
		"gitlab.com/org/group/repo",
		"gitlab:org/group/repo",
	} {
		if !entry.MatchesSource(source) {
			t.Errorf("did not match equivalent GitLab source %q", source)
		}
	}
	for _, source := range []string{
		"https://gitlab.com/other/group/repo",
		"file://gitlab.com/org/group/repo",
		"//gitlab.com/org/group/repo",
		"https://gitlab.com:8443/org/group/repo",
		"https://GITLAB.COM/org/group/repo",
		"https://gitlab.com/%6Frg/group/repo",
		"https://person@gitlab.com/org/group/repo",
		"ssh://git:password@gitlab.com/org/group/repo.git",
		"git@gitlab.com:org/group/repo.git",
	} {
		if entry.MatchesSource(source) {
			t.Errorf("matched non-equivalent GitLab source %q", source)
		}
	}
}

func TestMatchesSourceRejectsUnsupportedObservedProviderURLs(t *testing.T) {
	for _, tc := range []struct {
		name   string
		entry  Entry
		source string
	}{
		{"GitHub", Entry{Source: "file://github.com/org/repo", SourceType: "github"}, "file://github.com/org/repo"},
		{"GitHub control character", Entry{Source: "org/repo\x00", SourceType: "github"}, "org/repo\x00"},
		{"GitHub shorthand query", Entry{Source: "org/repo?mirror=other", SourceType: "github"}, "org/repo?mirror=other"},
		{"GitHub shorthand skill selector", Entry{Source: "org/repo@skill", SourceType: "github"}, "org/repo@skill"},
		{"GitHub prefix ref selector", Entry{Source: "github:org/repo:ref", SourceType: "github"}, "github:org/repo:ref"},
		{"GitHub credentials", Entry{Source: "https://person@github.com/org/repo", SourceType: "github"}, "https://person@github.com/org/repo"},
		{"GitLab", Entry{Source: "repo", SourceType: "gitlab", SourceURL: "file://gitlab.com/org/repo"}, "file://gitlab.com/org/repo"},
		{"GitLab control character", Entry{Source: "repo", SourceType: "gitlab", SourceURL: "gitlab:org/repo\x1b"}, "gitlab:org/repo\x1b"},
		{"GitLab shorthand fragment", Entry{Source: "repo", SourceType: "gitlab", SourceURL: "gitlab.com/org/repo#v1"}, "gitlab.com/org/repo#v1"},
		{"GitLab skill selector", Entry{Source: "repo", SourceType: "gitlab", SourceURL: "gitlab:org/repo@skill"}, "gitlab:org/repo@skill"},
		{"GitLab credentials", Entry{Source: "repo", SourceType: "gitlab", SourceURL: "https://person@gitlab.com/org/repo"}, "https://person@gitlab.com/org/repo"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.entry.MatchesSource(tc.source) {
				t.Fatalf("matched unsupported provider URL %q", tc.source)
			}
		})
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
	for _, source := range []string{
		"file:///tmp/repo",
		"https://person@example.com/repo.git",
		"ftp://example.com/repo.git",
		"https://example.com/repo.git#main",
		"https://example.com/repo",
		"ssh://git@example.com/%2F",
		"@example.com:repo.git",
		"git@example.com:repo.git\x00",
	} {
		if (Entry{SourceURL: source, SourceType: "git"}).MatchesSource(source) {
			t.Errorf("matched unrestorable generic Git source %q", source)
		}
	}
	if (Entry{Source: "/tmp/skill", SourceType: "local"}).MatchesSource("/tmp/skill") {
		t.Fatal("unrestorable local source matched")
	}
}
