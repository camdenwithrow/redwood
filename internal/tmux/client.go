package tmux

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type Client struct {
	run    func(args ...string) error
	attach func(name string) error
}

type Window struct {
	Name      string
	Command   string
	Directory string
	Port      int
}

type CommandError struct {
	Args   []string
	Output string
	Err    error
}

func NewClient() Client {
	return Client{run: run, attach: attach}
}

func (client Client) Attach(name string) error {
	return client.attach(name)
}

func (client Client) Stop(name string) error {
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
	first := windows[0]
	if err := client.run(
		"new-session", "-d", "-s", name,
		"-n", first.Name,
		"-c", first.Directory,
		"-e", "RW_PORT="+strconv.Itoa(first.Port),
		first.Command,
	); err != nil {
		return false, err
	}
	for _, window := range windows[1:] {
		if err := client.run(
			"new-window", "-d", "-t", name+":",
			"-n", window.Name,
			"-c", window.Directory,
			"-e", "RW_PORT="+strconv.Itoa(window.Port),
			window.Command,
		); err != nil {
			return false, errors.Join(err, client.run("kill-session", "-t", name))
		}
	}
	return false, nil
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
	return &CommandError{Args: append([]string(nil), args...), Output: detail, Err: err}
}

func attach(name string) error {
	command := exec.Command("tmux", "attach-session", "-t", "="+name)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return &CommandError{Args: []string{"attach-session", "-t", "=" + name}, Err: err, Output: err.Error()}
	}
	return nil
}

func (commandError *CommandError) Error() string {
	return fmt.Sprintf("tmux %s: %s", strings.Join(commandError.Args, " "), commandError.Output)
}

func (commandError *CommandError) Unwrap() error {
	return commandError.Err
}
