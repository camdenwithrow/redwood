package git

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunnerOutput(t *testing.T) {
	repositoryRoot := initializeRepository(t)
	runner := NewRunner(repositoryRoot)

	output, err := runner.Output("rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}

	canonicalRoot, err := filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		t.Fatalf("resolve repository path: %v", err)
	}
	if output != canonicalRoot {
		t.Fatalf("Output() = %q, want %q", output, canonicalRoot)
	}
}

func TestRunnerErrorIncludesCommandContext(t *testing.T) {
	repositoryRoot := initializeRepository(t)
	runner := NewRunner(repositoryRoot)

	err := runner.Run("show-ref", "--verify", "--quiet", "refs/heads/missing")
	if err == nil {
		t.Fatal("Run() error = nil, want Git error")
	}

	message := err.Error()
	for _, expected := range []string{
		"git show-ref --verify --quiet refs/heads/missing failed",
		repositoryRoot,
		"exit status 1",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("Run() error = %q, want it to contain %q", message, expected)
		}
	}

	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		t.Fatalf("Run() error type = %T, want wrapped *exec.ExitError", err)
	}
	if exitError.ExitCode() != 1 {
		t.Fatalf("Run() exit code = %d, want 1", exitError.ExitCode())
	}
}

func initializeRepository(t *testing.T) string {
	t.Helper()

	repositoryRoot := filepath.Join(t.TempDir(), "repository")
	command := exec.Command("git", "init", "-b", "main", repositoryRoot)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("initialize Git repository: %v: %s", err, output)
	}

	return repositoryRoot
}
