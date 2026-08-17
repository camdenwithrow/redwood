package tmux

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type Client struct {
	run func(args ...string) error
}

type Window struct {
	Name    string
	Command string
}

func NewClient() Client {
	return Client{run: run}
}

func (client Client) StartDetached(name string, windows []Window) error {
	if len(windows) == 0 {
		return fmt.Errorf("tmux session requires at least one window")
	}
	first := windows[0]
	if err := client.run("new-session", "-d", "-s", name, "-n", first.Name, first.Command); err != nil {
		return err
	}
	for _, window := range windows[1:] {
		if err := client.run("new-window", "-d", "-t", name+":", "-n", window.Name, window.Command); err != nil {
			return errors.Join(err, client.run("kill-session", "-t", name))
		}
	}
	return nil
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
	return fmt.Errorf("tmux %s: %s", strings.Join(args, " "), detail)
}
