package discovery

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// Skills returns the names of Skills directly below root.
func Skills(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read scan root %q: %w", root, err)
	}

	var names []string
	for _, entry := range entries {
		entryPath := filepath.Join(root, entry.Name())
		entryInfo, err := os.Stat(entryPath)
		if err != nil {
			return nil, fmt.Errorf("inspect scan entry %q: %w", entry.Name(), err)
		}
		if !entryInfo.IsDir() {
			continue
		}

		skillInfo, err := os.Lstat(filepath.Join(entryPath, "SKILL.md"))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("inspect SKILL.md for %q: %w", entry.Name(), err)
		}
		if skillInfo.Mode().IsRegular() {
			names = append(names, entry.Name())
		}
	}

	sort.Strings(names)
	return names, nil
}
