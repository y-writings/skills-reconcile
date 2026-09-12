package workspace

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/y-writings/skills-reconcile/internal/envvars"
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
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return "", false, errInvalidConfig
	}

	// Token iteration preserves duplicate members that Unmarshal would discard.
	workspaceSeen := false
	for decoder.More() {
		token, err = decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok || key != "workspace" || workspaceSeen {
			return "", false, errInvalidConfig
		}
		workspaceSeen = true

		var workspace *string
		if decoder.Decode(&workspace) != nil || workspace == nil {
			return "", false, errInvalidConfig
		}
		workspaceDir = *workspace
	}

	token, err = decoder.Token()
	if err != nil || token != json.Delim('}') || decoder.Decode(&struct{}{}) != io.EOF {
		return "", false, errInvalidConfig
	}
	if workspaceDir == "" {
		return "", false, nil
	}
	return workspaceDir, true, nil
}
