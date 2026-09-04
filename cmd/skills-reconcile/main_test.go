package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunPrintsHelp(t *testing.T) {
	var stdout bytes.Buffer

	if err := run([]string{"--help"}, &stdout); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != helpText {
		t.Fatalf("stdout = %q, want %q", got, helpText)
	}
}

func TestRunRejectsMissingInput(t *testing.T) {
	var stdout bytes.Buffer

	err := run(nil, &stdout)
	if err == nil || err.Error() != "usage: skills-reconcile --help" {
		t.Fatalf("error = %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

func TestRunRejectsUnsupportedInputs(t *testing.T) {
	for _, input := range []string{
		"doctor",
		"plan",
		"capture",
		"apply",
		"add",
		"remove",
		"adopt",
		"migrate",
		"--version",
	} {
		t.Run(input, func(t *testing.T) {
			var stdout bytes.Buffer

			err := run([]string{input}, &stdout)
			if err == nil || !strings.Contains(err.Error(), "unsupported command or option") {
				t.Fatalf("error = %v", err)
			}
			if !strings.Contains(err.Error(), input) {
				t.Fatalf("error = %q, want input %q", err, input)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
		})
	}
}

func TestRunRejectsArgumentsAfterHelp(t *testing.T) {
	var stdout bytes.Buffer

	err := run([]string{"--help", "apply"}, &stdout)
	if err == nil || !strings.Contains(err.Error(), "unsupported command or option") {
		t.Fatalf("error = %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}
