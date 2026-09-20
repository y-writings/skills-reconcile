package discovery

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// SkillNames returns the names of Skills directly below scanRootPath.
func SkillNames(scanRootPath string) ([]string, error) {
	entries, err := os.ReadDir(scanRootPath)
	if err != nil {
		return nil, fmt.Errorf("read scan root %q: %w", scanRootPath, err)
	}

	var skillNames []string
	for _, entry := range entries {
		entryPath := filepath.Join(scanRootPath, entry.Name())
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
			skillNames = append(skillNames, entry.Name())
		}
	}

	sort.Strings(skillNames)
	return skillNames, nil
}
