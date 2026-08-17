package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"help"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("Run() exit code = %d, want 0", exitCode)
	}
	if stdout.String() != usage {
		t.Fatalf("Run() stdout = %q, want usage text", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("Run() stderr = %q, want empty", stderr.String())
	}
}

func TestRunUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing command", want: "rw: no command provided"},
		{name: "unknown command", args: []string{"launch"}, want: `rw: unknown command "launch"`},
		{name: "missing branch", args: []string{"create"}, want: "rw: create requires <branch>"},
		{name: "extra branch", args: []string{"start", "one", "two"}, want: "rw: start accepts exactly one <branch> argument"},
		{name: "list argument", args: []string{"list", "feature/a"}, want: "rw: list does not accept arguments"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			exitCode := Run(test.args, &stdout, &stderr)

			if exitCode != 2 {
				t.Fatalf("Run() exit code = %d, want 2", exitCode)
			}
			if !strings.Contains(stderr.String(), test.want) {
				t.Fatalf("Run() stderr = %q, want it to contain %q", stderr.String(), test.want)
			}
			if stdout.Len() != 0 {
				t.Fatalf("Run() stdout = %q, want empty", stdout.String())
			}
		})
	}
}

func TestRunDispatchesKnownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"attach", "feature/a"}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("Run() exit code = %d, want 1", exitCode)
	}
	if got := stderr.String(); got != "rw: attach is not implemented yet\n" {
		t.Fatalf("Run() stderr = %q, want not-implemented error", got)
	}
}
