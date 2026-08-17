// Package git provides a small, contextual wrapper around the Git executable.
package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// Runner executes Git commands from a fixed working directory.
type Runner struct {
	directory string
}

// CommandError describes a failed Git invocation.
type CommandError struct {
	Directory string
	Args      []string
	Output    string
	Err       error
}

// NewRunner creates a Git runner rooted at directory.
func NewRunner(directory string) Runner {
	return Runner{directory: directory}
}

// Output runs Git and returns its trimmed combined output.
func (runner Runner) Output(args ...string) (string, error) {
	commandArgs := append([]string{"-C", runner.directory}, args...)
	output, err := exec.Command("git", commandArgs...).CombinedOutput()
	trimmedOutput := strings.TrimSpace(string(output))
	if err != nil {
		return "", &CommandError{
			Directory: runner.directory,
			Args:      append([]string(nil), args...),
			Output:    trimmedOutput,
			Err:       err,
		}
	}

	return trimmedOutput, nil
}

// Run executes Git when the caller does not need its output.
func (runner Runner) Run(args ...string) error {
	_, err := runner.Output(args...)
	return err
}

func (commandError *CommandError) Error() string {
	detail := commandError.Output
	if detail == "" {
		detail = commandError.Err.Error()
	}

	return fmt.Sprintf(
		"git %s failed in %q: %s",
		strings.Join(commandError.Args, " "),
		commandError.Directory,
		detail,
	)
}

// Unwrap exposes the underlying execution error, including its exit code.
func (commandError *CommandError) Unwrap() error {
	return commandError.Err
}
