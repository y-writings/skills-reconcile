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

// Overrides supplies an optional workspace directory that takes precedence over configured defaults.
type Overrides struct {
	WorkspaceDir string
}

// Location contains the resolved workspace directory and manifest path.
type Location struct {
	// WorkspaceDir is absolute and has its symlinks resolved.
	WorkspaceDir string
	// ManifestPath is absolute; its final component is not read or resolved.
	ManifestPath string
}

// Resolve determines the workspace and its manifest path from an explicit override
// and configured defaults, falling back to the current directory.
func Resolve(overrides Overrides) (Location, error) {
	selectedDir, err := selectWorkspaceDir(overrides.WorkspaceDir)
	if err != nil {
		return Location{}, err
	}
	workspaceDir, err := resolveWorkspaceDir(selectedDir)
	if err != nil {
		return Location{}, fmt.Errorf("resolve workspace: %w", err)
	}
	return Location{WorkspaceDir: workspaceDir, ManifestPath: filepath.Join(workspaceDir, "skills-manifest.json")}, nil
}

func resolveWorkspaceDir(selectedDir string) (string, error) {
	workspaceDir, err := filepath.EvalSymlinks(selectedDir)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(workspaceDir)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("workspace must be a directory: %s", workspaceDir)
	}
	return workspaceDir, nil
}

func selectWorkspaceDir(explicitWorkspaceDir string) (string, error) {
	if explicitWorkspaceDir != "" {
		return requireAbsoluteWorkspaceDir(explicitWorkspaceDir)
	}

	if environmentWorkspaceDir := os.Getenv(envvars.Workspace); environmentWorkspaceDir != "" {
		return requireAbsoluteWorkspaceDir(environmentWorkspaceDir)
	}

	configuredWorkspaceDir, found, err := lookupWorkspaceDirFromConfig()
	if err != nil {
		return "", err
	}
	if found {
		return requireAbsoluteWorkspaceDir(configuredWorkspaceDir)
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return requireAbsoluteWorkspaceDir(currentDir)
}

func requireAbsoluteWorkspaceDir(workspaceDir string) (string, error) {
	if !filepath.IsAbs(workspaceDir) {
		return "", errors.New("configured workspace must be absolute")
	}
	return workspaceDir, nil
}

func lookupWorkspaceDirFromConfig() (workspaceDir string, found bool, err error) {
	configHome := os.Getenv(envvars.XDGConfigHome)
	if configHome == "" {
		home := os.Getenv(envvars.Home)
		if home == "" {
			return "", false, nil
		}
		configHome = filepath.Join(home, ".config")
	}
	if !filepath.IsAbs(configHome) {
		return "", false, errors.New("config directory must be absolute")
	}
	data, err := os.ReadFile(filepath.Join(configHome, "skills-reconcile", "config.json"))
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	var config struct {
		Workspace string `json:"workspace"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var trailing any
	if decoder.Decode(&config) != nil || decoder.Decode(&trailing) != io.EOF {
		return "", false, errors.New("invalid skills-reconcile config")
	}
	if config.Workspace == "" {
		return "", false, nil
	}
	return config.Workspace, true, nil
}
