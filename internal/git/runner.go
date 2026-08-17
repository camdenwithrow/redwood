package git

import (
	"fmt"
	"os/exec"
	"strings"
)

type Runner struct {
	directory string
}

type CommandError struct {
	Directory string
	Args      []string
	Output    string
	Err       error
}

func NewRunner(directory string) Runner {
	return Runner{directory: directory}
}

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

func (commandError *CommandError) Unwrap() error {
	return commandError.Err
}
