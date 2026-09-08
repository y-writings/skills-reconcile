package workspace

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Resolve locates a workspace and manifest without searching parent directories
// or reading the manifest. An explicit manifest path takes precedence over workspace
// selection and may be relative; explicitly selected workspaces must be absolute.
func Resolve(flagValue, explicitManifestPath string) (string, string, error) {
	if explicitManifestPath != "" {
		path, err := filepath.Abs(explicitManifestPath)
		if err != nil {
			return "", "", err
		}
		root, err := filepath.EvalSymlinks(filepath.Dir(path))
		if err != nil {
			return "", "", err
		}
		return root, filepath.Join(root, filepath.Base(path)), nil
	}
	root := flagValue
	if root == "" {
		root = os.Getenv("SKILLS_RECONCILE_WORKSPACE")
	}
	if root == "" {
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			if home := os.Getenv("HOME"); home != "" {
				configHome = filepath.Join(home, ".config")
			}
		}
		if configHome != "" {
			if !filepath.IsAbs(configHome) {
				return "", "", errors.New("config directory must be absolute")
			}
			data, err := os.ReadFile(filepath.Join(configHome, "skills-reconcile", "config.json"))
			if err == nil {
				var config struct {
					Workspace string `json:"workspace"`
				}
				decoder := json.NewDecoder(bytes.NewReader(data))
				decoder.DisallowUnknownFields()
				var trailing any
				if decoder.Decode(&config) != nil || decoder.Decode(&trailing) != io.EOF {
					return "", "", errors.New("invalid skills-reconcile config")
				}
				root = config.Workspace
			}
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return "", "", err
			}
		}
	}
	if root == "" {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return "", "", err
		}
	}
	if !filepath.IsAbs(root) {
		return "", "", errors.New("configured workspace must be absolute")
	}
	root = filepath.Clean(root)
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", "", fmt.Errorf("resolve workspace: %w", err)
	}
	return root, filepath.Join(root, "skills-manifest.json"), nil
}
