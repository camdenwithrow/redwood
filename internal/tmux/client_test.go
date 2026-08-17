package tmux

import (
	"slices"
	"testing"
)

func TestStartDetached(t *testing.T) {
	var got []string
	client := Client{run: func(args ...string) error {
		got = append([]string(nil), args...)
		return nil
	}}

	if err := client.StartDetached("rw-redwood-feature-a-123456789abc"); err != nil {
		t.Fatalf("StartDetached() error = %v", err)
	}
	want := []string{"new-session", "-d", "-s", "rw-redwood-feature-a-123456789abc"}
	if !slices.Equal(got, want) {
		t.Fatalf("StartDetached() args = %v, want %v", got, want)
	}
}
