package workspace

import (
	"errors"
	"fmt"
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
