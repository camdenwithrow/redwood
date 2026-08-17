package tmux

import (
	"errors"
	"slices"
	"testing"
)

func TestStartDetached(t *testing.T) {
	var got [][]string
	client := Client{run: func(args ...string) error {
		got = append(got, append([]string(nil), args...))
		return nil
	}}
	windows := []Window{
		{Name: "backend", Command: "just dev-server", Directory: "/repo-feature-a", Port: 8180},
		{Name: "frontend", Command: "just dev-web", Directory: "/repo-feature-a", Port: 3100},
	}

	if err := client.StartDetached("rw-redwood-feature-a-123456789abc", windows); err != nil {
		t.Fatalf("StartDetached() error = %v", err)
	}
	want := [][]string{
		{"new-session", "-d", "-s", "rw-redwood-feature-a-123456789abc", "-n", "backend", "-c", "/repo-feature-a", "-e", "RW_PORT=8180", "just dev-server"},
		{"new-window", "-d", "-t", "rw-redwood-feature-a-123456789abc:", "-n", "frontend", "-c", "/repo-feature-a", "-e", "RW_PORT=3100", "just dev-web"},
	}
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("StartDetached() args = %v, want %v", got, want)
	}
}

func TestStartDetachedRequiresWindow(t *testing.T) {
	client := Client{run: func(...string) error { return nil }}
	if err := client.StartDetached("session", nil); err == nil {
		t.Fatal("StartDetached() error = nil, want missing window error")
	}
}

func TestStartDetachedKillsPartialSession(t *testing.T) {
	var commands []string
	client := Client{run: func(args ...string) error {
		commands = append(commands, args[0])
		if args[0] == "new-window" {
			return errors.New("window failed")
		}
		return nil
	}}
	windows := []Window{{Name: "one", Command: "first"}, {Name: "two", Command: "second"}}

	if err := client.StartDetached("session", windows); err == nil {
		t.Fatal("StartDetached() error = nil, want window error")
	}
	want := []string{"new-session", "new-window", "kill-session"}
	if !slices.Equal(commands, want) {
		t.Fatalf("StartDetached() commands = %v, want %v", commands, want)
	}
}
