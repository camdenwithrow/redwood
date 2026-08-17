package session

import "testing"

type fakeAttacher struct {
	name string
}

func (attacher *fakeAttacher) Attach(name string) error {
	attacher.name = name
	return nil
}

func TestAttachUsesWorktreeSession(t *testing.T) {
	repo := initializeRepository(t)
	client := &fakeAttacher{}

	if err := attach(repo, "main", client); err != nil {
		t.Fatalf("attach() error = %v", err)
	}
	if client.name == "" {
		t.Fatal("attach() session name is empty")
	}
}
