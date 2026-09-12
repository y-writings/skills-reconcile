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
	"unicode"
	"unicode/utf8"

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

var lockJSONFields = []string{"version", "skills", "dismissed", "lastSelectedAgents"}
var entryJSONFields = []string{"source", "sourceType", "sourceUrl", "sourceBaseUrl", "ref", "skillPath", "skillFolderHash", "installedAt", "updatedAt", "agents"}

func (lock *Lock) UnmarshalJSON(data []byte) error {
	type plainLock Lock
	return decodeCanonicalJSON(data, lockJSONFields, (*plainLock)(lock))
}

func (entry *Entry) UnmarshalJSON(data []byte) error {
	type plainEntry Entry
	return decodeCanonicalJSON(data, entryJSONFields, (*plainEntry)(entry))
}

func decodeCanonicalJSON(data []byte, canonicalFields []string, destination any) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for field := range fields {
		for _, canonical := range canonicalFields {
			if field != canonical && strings.EqualFold(field, canonical) {
				return fmt.Errorf("non-canonical JSON field %q", field)
			}
		}
	}
	return json.Unmarshal(data, destination)
}

// Path returns the global lock path for the current environment.
func Path() (string, error) {
	return ResolvePath(os.Getenv(envvars.XDGStateHome), os.Getenv(envvars.Home))
}

// ResolvePath selects the XDG state path when configured and otherwise falls back to HOME.
func ResolvePath(xdgStateHome, home string) (string, error) {
	if xdgStateHome != "" {
		if !filepath.IsAbs(xdgStateHome) {
			return "", errors.New("global lock directory must be absolute")
		}
		return filepath.Join(xdgStateHome, "skills", ".skill-lock.json"), nil
	}
	if home == "" {
		return "", errors.New("HOME is not set")
	}
	if !filepath.IsAbs(home) {
		return "", errors.New("global lock directory must be absolute")
	}
	return filepath.Join(home, ".agents", ".skill-lock.json"), nil
}

// Read decodes and validates an existing v3 global lock.
func Read(path string) (*Lock, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("global lock is not a regular file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(data) {
		return nil, errors.New("decode global lock: invalid UTF-8")
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
		missing, inspectErr := inspectMissingLockPath(path)
		if inspectErr != nil {
			return nil, inspectErr
		}
		if !missing {
			return nil, err
		}
		return &Lock{Version: 3, Skills: map[string]Entry{}, Missing: true}, nil
	}
	return observed, err
}

func inspectMissingLockPath(path string) (bool, error) {
	lockPath := filepath.Clean(path)
	for current := lockPath; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		switch {
		case err == nil:
			if info.Mode()&os.ModeSymlink != 0 {
				if _, statErr := os.Stat(current); errors.Is(statErr, os.ErrNotExist) {
					return false, errors.New("global lock path contains a dangling symlink")
				} else if statErr != nil {
					return false, fmt.Errorf("inspect global lock path: %w", statErr)
				}
			}
			return current != lockPath, nil
		case !errors.Is(err, os.ErrNotExist):
			return false, fmt.Errorf("inspect global lock path: %w", err)
		}

		parent := filepath.Dir(current)
		if parent == current {
			return true, nil
		}
	}
}

// InstallSource returns the source that can reproduce this observed installation.
func (e Entry) InstallSource() (string, bool) {
	switch e.SourceType {
	case "github":
		if _, ok := githubRepository(e.Source); !ok {
			return "", false
		}
		return e.Source, true
	case "git":
		if !validGenericGitSource(e.SourceURL) {
			return "", false
		}
		return e.SourceURL, true
	case "gitlab":
		if _, ok := gitlabRepository(e.SourceURL); !ok {
			return "", false
		}
		return e.SourceURL, true
	case "well-known":
		if e.SourceBaseURL != "" {
			if _, ok := parseWellKnownURL(e.SourceBaseURL); !ok {
				return "", false
			}
			return e.SourceBaseURL, true
		}
		return wellKnownBaseURL(e.SourceURL)
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
	if hasUnsafeSourceRune(source) || strings.ContainsAny(source, "?#") {
		return "", false
	}
	repository := source
	if strings.HasPrefix(source, "github:") {
		repository = strings.TrimPrefix(source, "github:")
	} else if strings.HasPrefix(source, "git@github.com:") {
		repository = strings.TrimPrefix(source, "git@github.com:")
	} else if parsed, err := url.Parse(source); err == nil &&
		supportedRepositoryURLScheme(parsed.Scheme) && plainRepositoryURL(source, parsed, "github.com") {
		repository = strings.TrimPrefix(parsed.Path, "/")
	} else if strings.Count(source, "/") != 1 || strings.Contains(source, ":") {
		return "", false
	}
	if strings.ContainsAny(repository, "@:") {
		return "", false
	}

	repository = strings.TrimSuffix(strings.TrimSuffix(repository, "/"), ".git")
	parts := strings.Split(repository, "/")
	if len(parts) != 2 || !validHostedRepositoryPart(parts[0]) || !validHostedRepositoryPart(parts[1]) {
		return "", false
	}
	return strings.ToLower(parts[0] + "/" + parts[1]), true
}

func gitlabRepository(source string) (string, bool) {
	if hasUnsafeSourceRune(source) || strings.ContainsAny(source, "?#") {
		return "", false
	}
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
		if !supportedRepositoryURLScheme(parsed.Scheme) {
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
		if !validHostedRepositoryPart(part) {
			return "", false
		}
	}
	return "gitlab.com/" + repository, true
}

func validHostedRepositoryPart(part string) bool {
	if part == "" || part == "." || part == ".." {
		return false
	}
	return strings.IndexFunc(part, func(character rune) bool {
		return (character < 'a' || character > 'z') &&
			(character < 'A' || character > 'Z') &&
			(character < '0' || character > '9') &&
			character != '.' && character != '_' && character != '-'
	}) < 0
}

func supportedRepositoryURLScheme(scheme string) bool {
	return scheme == "http" || scheme == "https" || scheme == "ssh"
}

func plainRepositoryURL(source string, parsed *url.URL, hostname string) bool {
	return !strings.ContainsAny(source, "?#") &&
		parsed.Hostname() == hostname &&
		parsed.Port() == "" &&
		plainRepositoryUserInfo(parsed) &&
		parsed.RawPath == "" &&
		parsed.RawQuery == "" &&
		parsed.Fragment == ""
}

func plainRepositoryUserInfo(parsed *url.URL) bool {
	if parsed.User == nil {
		return true
	}
	if parsed.Scheme != "ssh" || parsed.User.Username() == "" {
		return false
	}
	_, hasPassword := parsed.User.Password()
	return !hasPassword
}

func parseWellKnownURL(source string) (*url.URL, bool) {
	if !portableRemoteSource(source) ||
		(!strings.HasPrefix(source, "http://") && !strings.HasPrefix(source, "https://")) {
		return nil, false
	}
	parsed, err := url.Parse(source)
	if err != nil || parsed.Hostname() == "" || parsed.User != nil || strings.HasSuffix(source, ".git") ||
		reservedWellKnownHost(parsed.Hostname()) || sourceSelectsDifferentProvider(source, parsed) {
		return nil, false
	}
	return parsed, true
}

func reservedWellKnownHost(hostname string) bool {
	switch strings.ToLower(hostname) {
	case "github.com", "gitlab.com", "raw.githubusercontent.com",
		"codeload.github.com", "objects.githubusercontent.com":
		return true
	default:
		return false
	}
}

func validGenericGitSource(source string) bool {
	if !portableRemoteSource(source) || strings.HasPrefix(source, "github:") ||
		strings.HasPrefix(source, "gitlab:") {
		return false
	}

	parsed, err := url.Parse(source)
	if err == nil && parsed.Hostname() != "" {
		if sourceSelectsDifferentProvider(source, parsed) {
			return false
		}
		switch parsed.Scheme {
		case "http", "https":
			return parsed.User == nil && strings.HasPrefix(source, parsed.Scheme+"://") &&
				strings.HasSuffix(source, ".git")
		case "ssh":
			return strings.HasPrefix(source, "ssh://") && plainRepositoryUserInfo(parsed) &&
				nonEmptyRepositoryPath(parsed)
		case "git":
			return strings.HasPrefix(source, "git://") && parsed.User == nil && nonEmptyRepositoryPath(parsed)
		default:
			return false
		}
	}
	return validSCPGitSource(source)
}

func nonEmptyRepositoryPath(parsed *url.URL) bool {
	return strings.Trim(parsed.EscapedPath(), "/") != ""
}

func validSCPGitSource(source string) bool {
	if strings.Contains(source, "://") {
		return false
	}
	separator := strings.IndexByte(source, ':')
	if separator <= 0 || separator == len(source)-1 {
		return false
	}
	authority := source[:separator]
	if strings.ContainsAny(authority, `/\`) || strings.Count(authority, "@") > 1 {
		return false
	}
	hostname := authority
	if at := strings.LastIndexByte(authority, '@'); at >= 0 {
		hostname = authority[at+1:]
	}
	return hostname != ""
}

func portableRemoteSource(source string) bool {
	if source == "" || source != strings.TrimSpace(source) || hasUnsafeSourceRune(source) ||
		strings.ContainsAny(source, `?#\`) ||
		source == "." || source == ".." || filepath.IsAbs(source) ||
		strings.HasPrefix(source, "./") || strings.HasPrefix(source, "../") ||
		strings.HasPrefix(source, "~/") || strings.HasPrefix(strings.ToLower(source), "file:") ||
		windowsAbsoluteSource(source) {
		return false
	}
	parsed, err := url.Parse(source)
	if err != nil || parsed.User == nil {
		return true
	}
	_, hasPassword := parsed.User.Password()
	return !hasPassword && parsed.Scheme != "http" && parsed.Scheme != "https"
}

func hasUnsafeSourceRune(source string) bool {
	return strings.IndexFunc(source, func(character rune) bool {
		return unicode.IsSpace(character) || unicode.IsControl(character)
	}) >= 0
}

func windowsAbsoluteSource(source string) bool {
	return len(source) >= 3 && ((source[0] >= 'a' && source[0] <= 'z') ||
		(source[0] >= 'A' && source[0] <= 'Z')) && source[1] == ':' && source[2] == '/'
}

func sourceSelectsDifferentProvider(source string, parsed *url.URL) bool {
	switch strings.ToLower(parsed.Hostname()) {
	case "github.com", "gitlab.com", "raw.githubusercontent.com", "codeload.github.com",
		"objects.githubusercontent.com":
		return true
	}
	return strings.Contains(source, "github.com/") || strings.Contains(source, "gitlab.com/") ||
		((parsed.Scheme == "http" || parsed.Scheme == "https") && strings.Contains(parsed.Path, "/-/tree/"))
}

func wellKnownBaseURL(sourceURL string) (string, bool) {
	parsed, ok := parseWellKnownURL(sourceURL)
	if !ok {
		return "", false
	}

	path := parsed.EscapedPath()
	for _, marker := range []string{"/.well-known/agent-skills/", "/.well-known/skills/"} {
		markerIndex := strings.LastIndex(path, marker)
		if markerIndex < 0 || !validWellKnownSkillPath(path[markerIndex+len(marker):]) {
			continue
		}
		return parsed.Scheme + "://" + parsed.Host + path[:markerIndex], true
	}
	return "", false
}

func validWellKnownSkillPath(path string) bool {
	name, skillFile, found := strings.Cut(path, "/")
	return found && skillFile == "SKILL.md" && validWellKnownSkillName(name)
}

func validWellKnownSkillName(name string) bool {
	if len(name) == 0 || len(name) > 64 || name[0] == '-' || name[len(name)-1] == '-' ||
		strings.Contains(name, "--") {
		return false
	}
	for _, character := range name {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' {
			return false
		}
	}
	return true
}
