package workspace

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/y-writings/skills-reconcile/internal/envvars"
)

// Overrides supplies optional paths that take precedence over configured defaults.
type Overrides struct {
	WorkspacePath string
	ManifestPath  string
}

// Location contains the resolved workspace directory and manifest path.
type Location struct {
	// WorkspaceDir is absolute and has its symlinks resolved.
	WorkspaceDir string
	// ManifestPath is absolute; its final component is not read or resolved.
	ManifestPath string
}

// Resolve determines workspace and manifest paths from explicit overrides
// and configured defaults, falling back to the current directory.
func Resolve(overrides Overrides) (Location, error) {
	if overrides.ManifestPath != "" {
		path, err := filepath.Abs(overrides.ManifestPath)
		if err != nil {
			return Location{}, err
		}
		root, err := filepath.EvalSymlinks(filepath.Dir(path))
		if err != nil {
			return Location{}, err
		}
		return Location{WorkspaceDir: root, ManifestPath: filepath.Join(root, filepath.Base(path))}, nil
	}
	root, err := selectWorkspace(overrides.WorkspacePath)
	if err != nil {
		return Location{}, err
	}
	if !filepath.IsAbs(root) {
		return Location{}, errors.New("configured workspace must be absolute")
	}
	root = filepath.Clean(root)
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return Location{}, fmt.Errorf("resolve workspace: %w", err)
	}
	return Location{WorkspaceDir: root, ManifestPath: filepath.Join(root, "skills-manifest.json")}, nil
}

func selectWorkspace(explicitWorkspacePath string) (string, error) {
	if explicitWorkspacePath != "" {
		return explicitWorkspacePath, nil
	}
	if root := os.Getenv(envvars.Workspace); root != "" {
		return root, nil
	}
	root, err := workspaceFromConfig()
	if err != nil || root != "" {
		return root, err
	}
	return os.Getwd()
}

func workspaceFromConfig() (string, error) {
	configHome := os.Getenv(envvars.XDGConfigHome)
	if configHome == "" {
		home := os.Getenv(envvars.Home)
		if home == "" {
			return "", nil
		}
		configHome = filepath.Join(home, ".config")
	}
	if !filepath.IsAbs(configHome) {
		return "", errors.New("config directory must be absolute")
	}
	data, err := os.ReadFile(filepath.Join(configHome, "skills-reconcile", "config.json"))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var config struct {
		Workspace string `json:"workspace"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var trailing any
	if decoder.Decode(&config) != nil || decoder.Decode(&trailing) != io.EOF {
		return "", errors.New("invalid skills-reconcile config")
	}
	return config.Workspace, nil
}
