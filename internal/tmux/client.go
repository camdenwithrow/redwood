package tmux

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

type Client struct {
	run           func(args ...string) error
	attach        func(name string) error
	insideSession func() bool
}

type Window struct {
	Name        string
	Arguments   []string
	Shell       string
	Directory   string
	Environment map[string]string
}

type CommandError struct {
	Args   []string
	Output string
	Err    error
}

var ErrUnavailable = errors.New("tmux is not installed or not available on PATH")

type SessionNotFoundError struct {
	Name string
}

func NewClient() Client {
	return Client{
		run:    run,
		attach: attach,
		insideSession: func() bool {
			return os.Getenv("TMUX") != ""
		},
	}
}

func (client Client) Attach(name string) error {
	running, err := client.HasSession(name)
	if err != nil {
		return err
	}
	if !running {
		return SessionNotFoundError{Name: name}
	}
	if client.insideSession != nil && client.insideSession() {
		return client.run("switch-client", "-t", "="+name)
	}
	return client.attach(name)
}

func (client Client) Stop(name string) error {
	running, err := client.HasSession(name)
	if err != nil {
		return err
	}
	if !running {
		return SessionNotFoundError{Name: name}
	}
	return client.run("kill-session", "-t", "="+name)
}

func (client Client) StartDetached(name string, windows []Window) (bool, error) {
	running, err := client.HasSession(name)
	if err != nil {
		return false, err
	}
	if running {
		return true, nil
	}
	if len(windows) == 0 {
		return false, fmt.Errorf("tmux session requires at least one window")
	}
	commands, err := StartArguments(name, windows)
	if err != nil {
		return false, err
	}
	if err := client.run(commands[0]...); err != nil {
		return false, err
	}
	for _, command := range commands[1:] {
		if err := client.run(command...); err != nil {
			return false, errors.Join(err, client.run("kill-session", "-t", name))
		}
	}
	return false, nil
}

// StartArguments returns the tmux arguments used to create a detached session.
// The executable name itself is intentionally omitted from each argument list.
func StartArguments(name string, windows []Window) ([][]string, error) {
	if len(windows) == 0 {
		return nil, fmt.Errorf("tmux session requires at least one window")
	}

	first := windows[0]
	commands := [][]string{windowArgs(first,
		"new-session", "-d", "-s", name,
		"-n", first.Name,
		"-c", first.Directory,
	)}
	for _, window := range windows[1:] {
		commands = append(commands, windowArgs(window,
			"new-window", "-d", "-t", name+":",
			"-n", window.Name,
			"-c", window.Directory,
		))
	}
	return commands, nil
}

func windowArgs(window Window, args ...string) []string {
	result := append([]string(nil), args...)
	environmentNames := make([]string, 0, len(window.Environment))
	for name := range window.Environment {
		environmentNames = append(environmentNames, name)
	}
	sort.Strings(environmentNames)
	for _, name := range environmentNames {
		result = append(result, "-e", name+"="+window.Environment[name])
	}
	if len(window.Arguments) > 0 {
		result = append(result, window.Arguments...)
	} else if window.Shell != "" {
		result = append(result, window.Shell)
	}
	return result
}

func (client Client) HasSession(name string) (bool, error) {
	err := client.run("has-session", "-t", "="+name)
	if err == nil {
		return true, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

func run(args ...string) error {
	output, err := exec.Command("tmux", args...).CombinedOutput()
	if err == nil {
		return nil
	}
	detail := strings.TrimSpace(string(output))
	if detail == "" {
		detail = err.Error()
	}
	return newCommandError(args, detail, err)
}

func attach(name string) error {
	command := exec.Command("tmux", "attach-session", "-t", "="+name)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return newCommandError([]string{"attach-session", "-t", "=" + name}, "", err)
	}
	return nil
}

func (commandError *CommandError) Error() string {
	detail := commandError.Output
	if detail == "" {
		detail = commandError.Err.Error()
	}
	return fmt.Sprintf("tmux %s: %s", strings.Join(commandError.Args, " "), detail)
}

func (commandError *CommandError) Unwrap() error {
	return commandError.Err
}

func (sessionError SessionNotFoundError) Error() string {
	return fmt.Sprintf("tmux session %q is not running", sessionError.Name)
}

func newCommandError(args []string, output string, err error) error {
	var executableError *exec.Error
	if errors.As(err, &executableError) {
		return ErrUnavailable
	}
	return &CommandError{Args: append([]string(nil), args...), Output: output, Err: err}
}
