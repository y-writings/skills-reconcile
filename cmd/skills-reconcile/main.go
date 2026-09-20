package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/y-writings/skills-reconcile/internal/discovery"
)

const helpText = `Usage:
  skills-reconcile list
  skills-reconcile --help

Commands:
  list    List Skills in $HOME/.agents/skills.
`

func main() {
	if err := run(os.Args[1:], os.Getenv("HOME"), os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "skills-reconcile:", err)
		os.Exit(1)
	}
}

func run(args []string, home string, stdout io.Writer) error {
	if len(args) == 1 && args[0] == "--help" {
		_, err := io.WriteString(stdout, helpText)
		return err
	}
	if len(args) == 0 {
		return errors.New("usage: skills-reconcile list or skills-reconcile --help")
	}
	if args[0] != "list" {
		return fmt.Errorf("unsupported command or option %q", args[0])
	}
	if home == "" || !filepath.IsAbs(home) {
		return errors.New("HOME must be a non-empty absolute path")
	}

	skillNames, err := discovery.SkillNames(filepath.Join(home, ".agents", "skills"))
	if err != nil {
		return fmt.Errorf("discover Skills: %w", err)
	}
	for _, skillName := range skillNames {
		if _, err := fmt.Fprintln(stdout, skillName); err != nil {
			return fmt.Errorf("write Skill name: %w", err)
		}
	}
	return nil
}
