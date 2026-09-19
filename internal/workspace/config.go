package workspace

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/y-writings/skills-reconcile/internal/envvars"
	"github.com/y-writings/skills-reconcile/internal/jsondoc"
)

var errInvalidConfig = errors.New("invalid skills-reconcile config")

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
	return decodeWorkspaceConfig(data)
}

func decodeWorkspaceConfig(data []byte) (workspaceDir string, found bool, err error) {
	var config struct {
		Workspace json.RawMessage `json:"workspace"`
	}
	if jsondoc.DecodeObject(data, &config) != nil {
		return "", false, errInvalidConfig
	}
	if config.Workspace == nil {
		return "", false, nil
	}
	var workspace *string
	if json.Unmarshal(config.Workspace, &workspace) != nil || workspace == nil {
		return "", false, errInvalidConfig
	}
	if *workspace == "" {
		return "", false, nil
	}
	return *workspace, true, nil
}
