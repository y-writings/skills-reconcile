package main_test

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const expectedHelp = `Usage:
  skills-reconcile list
  skills-reconcile --help

Commands:
  list    List Skills in $HOME/.agents/skills.
`

func TestCLI(t *testing.T) {
	binary := buildCLI(t)

	t.Run("prints help without HOME", func(t *testing.T) {
		result := runCLI(t, binary, environmentWithoutHome(), "--help")
		assertResult(t, result, 0, expectedHelp, "")
	})

	t.Run("lists Skills and ignores arguments after list", func(t *testing.T) {
		home := newHome(t)
		writeSkill(t, home, "zed")
		writeSkill(t, home, "alpha")

		before := snapshotTree(t, home)
		result := runCLI(t, binary, environmentWithHome(home), "list", "--help")
		after := snapshotTree(t, home)

		assertResult(t, result, 0, "alpha\nzed\n", "")
		if !reflect.DeepEqual(after, before) {
			t.Fatalf("HOME tree changed after list:\nbefore: %#v\nafter:  %#v", before, after)
		}
	})

	t.Run("succeeds when no Skills match", func(t *testing.T) {
		home := newHome(t)
		result := runCLI(t, binary, environmentWithHome(home), "list")
		assertResult(t, result, 0, "", "")
	})

	t.Run("rejects invalid arguments", func(t *testing.T) {
		home := newHome(t)
		tests := map[string][]string{
			"missing":              nil,
			"unsupported":          {"unknown"},
			"arguments after help": {"--help", "extra"},
		}

		for name, args := range tests {
			t.Run(name, func(t *testing.T) {
				result := runCLI(t, binary, environmentWithHome(home), args...)
				assertFailure(t, result)
			})
		}
	})

	t.Run("rejects invalid HOME", func(t *testing.T) {
		tests := map[string][]string{
			"unset":    environmentWithoutHome(),
			"empty":    environmentWithHome(""),
			"relative": environmentWithHome("relative/home"),
		}

		for name, environment := range tests {
			t.Run(name, func(t *testing.T) {
				result := runCLI(t, binary, environment, "list")
				assertFailure(t, result)
			})
		}
	})

	t.Run("does not print partial results on discovery failure", func(t *testing.T) {
		home := newHome(t)
		writeSkill(t, home, "a-valid")
		brokenLink := filepath.Join(home, ".agents", "skills", "z-invalid")
		if err := os.Symlink(filepath.Join(t.TempDir(), "missing"), brokenLink); err != nil {
			t.Fatal(err)
		}

		result := runCLI(t, binary, environmentWithHome(home), "list")
		assertFailure(t, result)
	})
}

func buildCLI(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "skills-reconcile")
	command := exec.Command("go", "build", "-o", binary, ".")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	return binary
}

type commandResult struct {
	status int
	stdout string
	stderr string
}

func runCLI(t *testing.T, binary string, environment []string, args ...string) commandResult {
	t.Helper()
	command := exec.Command(binary, args...)
	command.Env = environment

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	status := 0
	if err := command.Run(); err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			t.Fatalf("run CLI: %v", err)
		}
		status = exitError.ExitCode()
	}
	return commandResult{status: status, stdout: stdout.String(), stderr: stderr.String()}
}

func environmentWithoutHome() []string {
	environment := make([]string, 0, len(os.Environ()))
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "HOME=") {
			environment = append(environment, entry)
		}
	}
	return environment
}

func environmentWithHome(home string) []string {
	return append(environmentWithoutHome(), "HOME="+home)
}

func newHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".agents", "skills"), 0o700); err != nil {
		t.Fatal(err)
	}
	return home
}

func writeSkill(t *testing.T, home, name string) {
	t.Helper()
	directory := filepath.Join(home, ".agents", "skills", name)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "SKILL.md"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertResult(t *testing.T, result commandResult, status int, stdout, stderr string) {
	t.Helper()
	if result.status != status {
		t.Errorf("status = %d, want %d", result.status, status)
	}
	if result.stdout != stdout {
		t.Errorf("stdout = %q, want %q", result.stdout, stdout)
	}
	if result.stderr != stderr {
		t.Errorf("stderr = %q, want %q", result.stderr, stderr)
	}
}

func assertFailure(t *testing.T, result commandResult) {
	t.Helper()
	if result.status != 1 {
		t.Errorf("status = %d, want 1", result.status)
	}
	if result.stdout != "" {
		t.Errorf("stdout = %q, want empty", result.stdout)
	}
	if result.stderr == "" {
		t.Error("stderr is empty, want diagnostic")
	}
}

type treeEntry struct {
	mode           fs.FileMode
	modificationNS int64
	content        string
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

		entry := treeEntry{mode: info.Mode(), modificationNS: info.ModTime().UnixNano()}
		switch {
		case info.Mode().IsRegular():
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			entry.content = string(contents)
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			entry.content = target
		}
		snapshot[relativePath] = entry
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}
