package tmux

import (
	"errors"
	"os/exec"
	"slices"
	"testing"
)

func TestStartDetached(t *testing.T) {
	var got [][]string
	client := Client{run: func(args ...string) error {
		got = append(got, append([]string(nil), args...))
		if args[0] == "has-session" {
			return missingSessionError(t)
		}
		return nil
	}}
	windows := []Window{
		{Name: "backend", Command: "just dev-server", Directory: "/repo-feature-a", Port: intPointer(8180)},
		{Name: "frontend", Command: "just dev-web", Directory: "/repo-feature-a", Port: intPointer(3100)},
	}

	if _, err := client.StartDetached("rw-redwood-feature-a-123456789abc", windows); err != nil {
		t.Fatalf("StartDetached() error = %v", err)
	}
	want := [][]string{
		{"has-session", "-t", "=rw-redwood-feature-a-123456789abc"},
		{"new-session", "-d", "-s", "rw-redwood-feature-a-123456789abc", "-n", "backend", "-c", "/repo-feature-a", "-e", "RW_PORT=8180", "just dev-server"},
		{"new-window", "-d", "-t", "rw-redwood-feature-a-123456789abc:", "-n", "frontend", "-c", "/repo-feature-a", "-e", "RW_PORT=3100", "just dev-web"},
	}
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("StartDetached() args = %v, want %v", got, want)
	}
}

func TestStartDetachedShellWithoutPortOrCommand(t *testing.T) {
	var got [][]string
	client := Client{run: func(args ...string) error {
		got = append(got, append([]string(nil), args...))
		if args[0] == "has-session" {
			return missingSessionError(t)
		}
		return nil
	}}

	if _, err := client.StartDetached("session", []Window{{Name: "shell", Directory: "/repo-feature-a"}}); err != nil {
		t.Fatalf("StartDetached() error = %v", err)
	}
	want := [][]string{
		{"has-session", "-t", "=session"},
		{"new-session", "-d", "-s", "session", "-n", "shell", "-c", "/repo-feature-a"},
	}
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("StartDetached() args = %v, want %v", got, want)
	}
}

func TestStartDetachedRequiresWindow(t *testing.T) {
	client := Client{run: func(args ...string) error {
		if args[0] == "has-session" {
			return missingSessionError(t)
		}
		return nil
	}}
	if _, err := client.StartDetached("session", nil); err == nil {
		t.Fatal("StartDetached() error = nil, want missing window error")
	}
}

func TestStartDetachedKillsPartialSession(t *testing.T) {
	var commands []string
	client := Client{run: func(args ...string) error {
		commands = append(commands, args[0])
		if args[0] == "has-session" {
			return missingSessionError(t)
		}
		if args[0] == "new-window" {
			return errors.New("window failed")
		}
		return nil
	}}
	windows := []Window{{Name: "one", Command: "first"}, {Name: "two", Command: "second"}}

	if _, err := client.StartDetached("session", windows); err == nil {
		t.Fatal("StartDetached() error = nil, want window error")
	}
	want := []string{"has-session", "new-session", "new-window", "kill-session"}
	if !slices.Equal(commands, want) {
		t.Fatalf("StartDetached() commands = %v, want %v", commands, want)
	}
}

func TestStartDetachedReturnsExistingSession(t *testing.T) {
	var commands []string
	client := Client{run: func(args ...string) error {
		commands = append(commands, args[0])
		return nil
	}}

	alreadyRunning, err := client.StartDetached("session", []Window{{Name: "web"}})
	if err != nil {
		t.Fatalf("StartDetached() error = %v", err)
	}
	if !alreadyRunning {
		t.Fatal("StartDetached() alreadyRunning = false, want true")
	}
	if !slices.Equal(commands, []string{"has-session"}) {
		t.Fatalf("StartDetached() commands = %v, want only has-session", commands)
	}
}

func TestAttach(t *testing.T) {
	var got string
	client := Client{
		run: func(...string) error { return nil },
		attach: func(name string) error {
			got = name
			return nil
		},
	}

	if err := client.Attach("session"); err != nil {
		t.Fatalf("Attach() error = %v", err)
	}
	if got != "session" {
		t.Fatalf("Attach() name = %q, want session", got)
	}
}

func TestAttachSwitchesClientInsideTmux(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-session,1,0")
	var got [][]string
	attachCalled := false
	client := NewClient()
	client.run = func(args ...string) error {
		got = append(got, append([]string(nil), args...))
		return nil
	}
	client.attach = func(string) error {
		attachCalled = true
		return nil
	}

	if err := client.Attach("session"); err != nil {
		t.Fatalf("Attach() error = %v", err)
	}
	want := [][]string{
		{"has-session", "-t", "=session"},
		{"switch-client", "-t", "=session"},
	}
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("Attach() args = %v, want %v", got, want)
	}
	if attachCalled {
		t.Fatal("Attach() called attach-session while already inside tmux")
	}
}

func TestStop(t *testing.T) {
	var got [][]string
	client := Client{run: func(args ...string) error {
		got = append(got, append([]string(nil), args...))
		return nil
	}}

	if err := client.Stop("session"); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	want := [][]string{{"has-session", "-t", "=session"}, {"kill-session", "-t", "=session"}}
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("Stop() args = %v, want %v", got, want)
	}
}

func TestAttachAndStopRejectMissingSession(t *testing.T) {
	client := Client{
		run:    func(...string) error { return missingSessionError(t) },
		attach: func(string) error { return errors.New("attach should not run") },
	}

	for operation, run := range map[string]func() error{
		"attach": func() error { return client.Attach("missing") },
		"stop":   func() error { return client.Stop("missing") },
	} {
		t.Run(operation, func(t *testing.T) {
			var sessionError SessionNotFoundError
			if err := run(); !errors.As(err, &sessionError) {
				t.Fatalf("error = %v, want SessionNotFoundError", err)
			}
		})
	}
}

func TestUnavailableError(t *testing.T) {
	err := newCommandError(
		[]string{"has-session"},
		"",
		&exec.Error{Name: "tmux", Err: exec.ErrNotFound},
	)
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("newCommandError() = %v, want ErrUnavailable", err)
	}
}

func missingSessionError(t *testing.T) error {
	t.Helper()
	err := exec.Command("sh", "-c", "exit 1").Run()
	exitError, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("create missing session error: %v", err)
	}
	return &CommandError{Err: exitError}
}

func intPointer(value int) *int {
	return &value
}
