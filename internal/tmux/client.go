package tmux

import (
	"fmt"
	"os/exec"
	"strings"
)

type Client struct {
	run func(args ...string) error
}

func NewClient() Client {
	return Client{run: run}
}

func (client Client) StartDetached(name string) error {
	return client.run("new-session", "-d", "-s", name)
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
