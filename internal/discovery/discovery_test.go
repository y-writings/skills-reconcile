package discovery

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
)

func TestSkillNamesRecognizesDirectSkillDirectories(t *testing.T) {
	root := newScanRoot(t)
	writeSkill(t, root, "middle", "---\nname: middle\n---\n")
	writeSkill(t, root, "zed", "")
	writeSkill(t, root, "alpha", "not validated")

	mustWriteFile(t, filepath.Join(root, "plain-file"), []byte("ignored"), 0o600)
	if err := syscall.Mkfifo(filepath.Join(root, "special-file"), 0o600); err != nil {
		t.Fatal(err)
	}
	mustMkdirAll(t, filepath.Join(root, "missing-skill"), 0o700)
	mustMkdirAll(t, filepath.Join(root, "skill-md-directory", "SKILL.md"), 0o700)
	mustMkdirAll(t, filepath.Join(root, "nested-only", "child"), 0o700)
	mustWriteFile(t, filepath.Join(root, "nested-only", "child", "SKILL.md"), nil, 0o600)

	symlinkTarget := filepath.Join(t.TempDir(), "target.md")
	mustWriteFile(t, symlinkTarget, nil, 0o600)
	symlinkSkill := filepath.Join(root, "skill-md-symlink")
	mustMkdirAll(t, symlinkSkill, 0o700)
	mustSymlink(t, symlinkTarget, filepath.Join(symlinkSkill, "SKILL.md"))

	got, err := SkillNames(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alpha", "middle", "zed"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SkillNames() = %q, want %q", got, want)
	}
}

func TestSkillNamesAcceptsOnlyContractNameSyntax(t *testing.T) {
	root := newScanRoot(t)
	maximumLengthName := strings.Repeat("a", 64)
	for _, name := range []string{"0", "a-b", maximumLengthName} {
		writeSkill(t, root, name, "")
	}
	for _, name := range []string{
		"Uppercase",
		"with_underscore",
		"with.dot",
		"-leading",
		"trailing-",
		"double--hyphen",
		strings.Repeat("a", 65),
		"non-ascii-é",
	} {
		writeSkill(t, root, name, "")
	}

	got, err := SkillNames(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"0", "a-b", maximumLengthName}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SkillNames() = %q, want %q", got, want)
	}
}

func TestSkillNamesIgnoresInvalidNamesBeforeInspection(t *testing.T) {
	root := newScanRoot(t)
	mustSymlink(t, filepath.Join(t.TempDir(), "missing"), filepath.Join(root, "broken_link"))

	got, err := SkillNames(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("SkillNames() = %q, want empty", got)
	}
}

func TestSkillNamesReturnsEmptyWhenNoSkillsMatch(t *testing.T) {
	tests := map[string]func(*testing.T, string){
		"empty root": func(_ *testing.T, _ string) {},
		"ignored entries": func(t *testing.T, root string) {
			mustWriteFile(t, filepath.Join(root, "plain-file"), nil, 0o600)
			mustMkdirAll(t, filepath.Join(root, "missing-skill"), 0o700)
		},
	}

	for name, populate := range tests {
		t.Run(name, func(t *testing.T) {
			root := newScanRoot(t)
			populate(t, root)

			got, err := SkillNames(root)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 0 {
				t.Fatalf("SkillNames() = %q, want empty", got)
			}
		})
	}
}

func TestSkillNamesUsesRootEntryNameForDirectorySymlink(t *testing.T) {
	root := newScanRoot(t)
	targetParent := t.TempDir()
	writeSkill(t, targetParent, "target-name", "outside the scan root")
	mustSymlink(t, filepath.Join(targetParent, "target-name"), filepath.Join(root, "linked-name"))

	fileTarget := filepath.Join(targetParent, "file-target")
	mustWriteFile(t, fileTarget, nil, 0o600)
	mustSymlink(t, fileTarget, filepath.Join(root, "file-link"))

	got, err := SkillNames(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"linked-name"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SkillNames() = %q, want %q", got, want)
	}
}

func TestSkillNamesFollowsSymlinkedScanRoot(t *testing.T) {
	temp := t.TempDir()
	realRoot := filepath.Join(temp, "real-root")
	mustMkdirAll(t, realRoot, 0o700)
	writeSkill(t, realRoot, "example", "")

	linkedRoot := filepath.Join(temp, "linked-root")
	mustSymlink(t, realRoot, linkedRoot)

	got, err := SkillNames(linkedRoot)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"example"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SkillNames() = %q, want %q", got, want)
	}
}

func TestSkillNamesRejectsInvalidScanRoot(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "missing")
		assertDiscoveryFailure(t, root)
	})

	t.Run("not a directory", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "file")
		mustWriteFile(t, root, nil, 0o600)
		assertDiscoveryFailure(t, root)
	})

	t.Run("unreadable", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("permission checks are ineffective as root")
		}
		root := newScanRoot(t)
		if err := os.Chmod(root, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := os.Chmod(root, 0o700); err != nil {
				t.Errorf("restore permissions: %v", err)
			}
		})

		assertDiscoveryFailure(t, root)
	})
}

func TestSkillNamesFailsWithoutPartialResultsForInvalidSymlink(t *testing.T) {
	tests := map[string]func(*testing.T, string){
		"broken": func(t *testing.T, path string) {
			mustSymlink(t, filepath.Join(t.TempDir(), "missing"), path)
		},
		"loop": func(t *testing.T, path string) {
			mustSymlink(t, path, path)
		},
	}

	for name, createInvalidEntry := range tests {
		t.Run(name, func(t *testing.T) {
			root := newScanRoot(t)
			writeSkill(t, root, "a-valid", "")
			createInvalidEntry(t, filepath.Join(root, "z-invalid"))

			assertDiscoveryFailure(t, root)
		})
	}
}

func TestSkillNamesFailsWithoutPartialResultsForUnreadableEntry(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission checks are ineffective as root")
	}

	root := newScanRoot(t)
	writeSkill(t, root, "a-valid", "")
	locked := filepath.Join(root, "z-locked")
	writeSkill(t, root, "z-locked", "")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(locked, 0o700); err != nil {
			t.Errorf("restore permissions: %v", err)
		}
	})

	assertDiscoveryFailure(t, root)
}

func TestSkillNamesDoesNotModifyInputTree(t *testing.T) {
	temp := t.TempDir()
	root := filepath.Join(temp, "root")
	mustMkdirAll(t, root, 0o700)
	writeSkill(t, root, "local", "local contents")
	writeSkill(t, temp, "outside", "outside contents")
	mustSymlink(t, filepath.Join(temp, "outside"), filepath.Join(root, "linked"))

	before := snapshotTree(t, temp)
	if _, err := SkillNames(root); err != nil {
		t.Fatal(err)
	}
	after := snapshotTree(t, temp)
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("tree changed after discovery:\nbefore: %#v\nafter:  %#v", before, after)
	}
}

func newScanRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".agents", "skills")
	mustMkdirAll(t, root, 0o700)
	return root
}

func writeSkill(t *testing.T, parent, name, contents string) {
	t.Helper()
	directory := filepath.Join(parent, name)
	mustMkdirAll(t, directory, 0o700)
	mustWriteFile(t, filepath.Join(directory, "SKILL.md"), []byte(contents), 0o600)
}

func assertDiscoveryFailure(t *testing.T, root string) {
	t.Helper()
	got, err := SkillNames(root)
	if err == nil {
		t.Fatal("SkillNames() error = nil, want failure")
	}
	if got != nil {
		t.Fatalf("SkillNames() = %q on failure, want nil", got)
	}
}

func mustMkdirAll(t *testing.T, path string, mode fs.FileMode) {
	t.Helper()
	if err := os.MkdirAll(path, mode); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, path string, contents []byte, mode fs.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, contents, mode); err != nil {
		t.Fatal(err)
	}
}

func mustSymlink(t *testing.T, target, path string) {
	t.Helper()
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
}

type treeEntry struct {
	Mode           fs.FileMode
	ModificationNS int64
	Content        string
}

func snapshotTree(t *testing.T, root string) map[string]treeEntry {
	t.Helper()
	snapshot := make(map[string]treeEntry)
	err := filepath.WalkDir(root, func(path string, _ fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		entry := treeEntry{
			Mode:           info.Mode(),
			ModificationNS: info.ModTime().UnixNano(),
		}
		switch {
		case info.Mode().IsRegular():
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			entry.Content = string(contents)
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			entry.Content = target
		}
		snapshot[relativePath] = entry
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}
