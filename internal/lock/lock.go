// Package lock reads the machine-local state written by the skills CLI.
package lock

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/y-writings/skills-reconcile/internal/envvars"
)

// Lock is the observed v3 global lock written by the skills CLI.
type Lock struct {
	Version            int              `json:"version"`
	Skills             map[string]Entry `json:"skills"`
	Dismissed          map[string]any   `json:"dismissed,omitempty"`
	LastSelectedAgents []string         `json:"lastSelectedAgents,omitempty"`
	Missing            bool             `json:"-"`
}

// Entry describes the observed installation source for one Skill.
type Entry struct {
	Source          string   `json:"source"`
	SourceType      string   `json:"sourceType"`
	SourceURL       string   `json:"sourceUrl,omitempty"`
	SourceBaseURL   string   `json:"sourceBaseUrl,omitempty"`
	Ref             string   `json:"ref,omitempty"`
	SkillPath       string   `json:"skillPath,omitempty"`
	SkillFolderHash string   `json:"skillFolderHash,omitempty"`
	InstalledAt     string   `json:"installedAt,omitempty"`
	UpdatedAt       string   `json:"updatedAt,omitempty"`
	Agents          []string `json:"agents,omitempty"`
}

// Path returns the global lock path for the current environment.
func Path() (string, error) {
	return ResolvePath(os.Getenv(envvars.XDGStateHome), os.Getenv(envvars.Home))
}

// ResolvePath selects the XDG state path when configured and otherwise falls back to HOME.
func ResolvePath(xdgStateHome, home string) (string, error) {
	if xdgStateHome != "" {
		return filepath.Join(xdgStateHome, "skills", ".skill-lock.json"), nil
	}
	if home == "" {
		return "", errors.New("HOME is not set")
	}
	return filepath.Join(home, ".agents", ".skill-lock.json"), nil
}

// Read decodes and validates an existing v3 global lock.
func Read(path string) (*Lock, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	var observed Lock
	if err := decoder.Decode(&observed); err != nil {
		return nil, fmt.Errorf("decode global lock: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, errors.New("decode global lock: trailing JSON")
	}
	if observed.Version != 3 {
		return nil, fmt.Errorf("unsupported global lock version %d", observed.Version)
	}
	if observed.Skills == nil {
		return nil, errors.New("global lock has no skills object")
	}
	for name, entry := range observed.Skills {
		if name == "" || entry.Source == "" || entry.SourceType == "" {
			return nil, fmt.Errorf("invalid lock entry %q", name)
		}
	}
	return &observed, nil
}

// ReadObserved returns an empty marked lock only when the global lock does not exist.
func ReadObserved(path string) (*Lock, error) {
	observed, err := Read(path)
	if errors.Is(err, os.ErrNotExist) {
		info, lstatErr := os.Lstat(path)
		switch {
		case lstatErr == nil && info.Mode()&os.ModeSymlink != 0:
			return nil, errors.New("global lock is a dangling symlink")
		case lstatErr == nil:
			return nil, err
		case !errors.Is(lstatErr, os.ErrNotExist):
			return nil, fmt.Errorf("inspect global lock path: %w", lstatErr)
		}
		return &Lock{Version: 3, Skills: map[string]Entry{}, Missing: true}, nil
	}
	return observed, err
}

// InstallSource returns the source that can reproduce this observed installation.
func (e Entry) InstallSource() (string, bool) {
	switch e.SourceType {
	case "github":
		return e.Source, e.Source != ""
	case "git", "gitlab":
		return e.SourceURL, e.SourceURL != ""
	case "well-known":
		if e.SourceBaseURL != "" {
			return e.SourceBaseURL, true
		}
		if marker := strings.Index(e.SourceURL, "/.well-known/"); marker >= 0 {
			return e.SourceURL[:marker], true
		}
		return "", false
	default:
		return "", false
	}
}

// MatchesSource reports whether source identifies the same restorable repository as the entry.
func (e Entry) MatchesSource(source string) bool {
	observed, restorable := e.InstallSource()
	if !restorable {
		return false
	}
	if observed == source {
		return true
	}

	switch e.SourceType {
	case "github":
		observedRepository, observedGitHub := githubRepository(observed)
		desiredRepository, desiredGitHub := githubRepository(source)
		return observedGitHub && desiredGitHub && observedRepository == desiredRepository
	case "gitlab":
		observedRepository, observedGitLab := gitlabRepository(observed)
		desiredRepository, desiredGitLab := gitlabRepository(source)
		return observedGitLab && desiredGitLab && observedRepository == desiredRepository
	default:
		return false
	}
}

func githubRepository(source string) (string, bool) {
	repository := source
	if strings.HasPrefix(source, "github:") {
		repository = strings.TrimPrefix(source, "github:")
	} else if strings.HasPrefix(source, "git@github.com:") {
		repository = strings.TrimPrefix(source, "git@github.com:")
	} else if parsed, err := url.Parse(source); err == nil && plainRepositoryURL(source, parsed, "github.com") {
		repository = strings.TrimPrefix(parsed.Path, "/")
	} else if strings.Count(source, "/") != 1 || strings.Contains(source, ":") {
		return "", false
	}

	repository = strings.TrimSuffix(strings.TrimSuffix(repository, "/"), ".git")
	parts := strings.Split(repository, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", false
	}
	return strings.ToLower(parts[0] + "/" + parts[1]), true
}

func gitlabRepository(source string) (string, bool) {
	var repository string
	switch {
	case strings.HasPrefix(source, "gitlab:"):
		repository = strings.TrimPrefix(source, "gitlab:")
	case strings.HasPrefix(source, "gitlab.com/"):
		repository = strings.TrimPrefix(source, "gitlab.com/")
	default:
		parsed, err := url.Parse(source)
		if err != nil {
			return "", false
		}
		switch parsed.Scheme {
		case "http", "https", "ssh":
		default:
			return "", false
		}
		if !plainRepositoryURL(source, parsed, "gitlab.com") {
			return "", false
		}
		repository = strings.TrimPrefix(parsed.Path, "/")
	}

	repository = strings.TrimSuffix(strings.TrimSuffix(repository, "/"), ".git")
	parts := strings.Split(repository, "/")
	if len(parts) < 2 {
		return "", false
	}
	for _, part := range parts {
		if part == "" {
			return "", false
		}
	}
	return "gitlab.com/" + repository, true
}

func plainRepositoryURL(source string, parsed *url.URL, hostname string) bool {
	return !strings.ContainsAny(source, "?#") &&
		parsed.Hostname() == hostname &&
		parsed.Port() == "" &&
		parsed.RawPath == "" &&
		parsed.RawQuery == "" &&
		parsed.Fragment == ""
}
