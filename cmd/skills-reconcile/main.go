package main

import (
	"errors"
	"fmt"
	"io"
	"os"
)

const helpText = `Usage: skills-reconcile --help

No commands are available in this migration phase.
`

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "skills-reconcile:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: skills-reconcile --help")
	}
	if len(args) == 1 && args[0] == "--help" {
		_, err := io.WriteString(stdout, helpText)
		return err
	}
	return fmt.Errorf("unsupported command or option %q", args[0])
}
