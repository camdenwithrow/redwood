package session

import "testing"

type fakeStopper struct {
	name string
}

func (stopper *fakeStopper) Stop(name string) error {
	stopper.name = name
	return nil
}

func TestStopUsesWorktreeSession(t *testing.T) {
	repo := initializeRepository(t)
	client := &fakeStopper{}

	name, err := stop(repo, "main", client)
	if err != nil {
		t.Fatalf("stop() error = %v", err)
	}
	if name == "" || client.name != name {
		t.Fatalf("stop() name = %q, client name = %q", name, client.name)
	}
}
