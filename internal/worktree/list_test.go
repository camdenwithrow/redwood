package worktree

import (
	"testing"

	"github.com/camdenwithrow/redwood/internal/allocation"
	"github.com/camdenwithrow/redwood/internal/repository"
)

func TestCombineWorktreesWithSlots(t *testing.T) {
	worktrees := []repository.Worktree{
		{Path: "/repo", Branch: "main", Commit: "1111111"},
		{Path: "/repo-feature-a", Branch: "feature/a", Commit: "2222222"},
		{Path: "/repo-detached", Commit: "3333333", Detached: true},
	}
	state := allocation.State{
		Version: 1,
		Slots:   map[string]int{"main": 0, "feature/a": 2},
	}

	listed, err := combine(worktrees, state)
	if err != nil {
		t.Fatalf("combine() error = %v", err)
	}
	if len(listed) != 3 {
		t.Fatalf("combine() returned %d worktrees, want 3", len(listed))
	}
	if listed[0].Slot == nil || *listed[0].Slot != 0 {
		t.Fatalf("combine() main slot = %v, want 0", listed[0].Slot)
	}
	if listed[1].Slot == nil || *listed[1].Slot != 2 {
		t.Fatalf("combine() feature/a slot = %v, want 2", listed[1].Slot)
	}
	if listed[2].Slot != nil {
		t.Fatalf("combine() detached slot = %v, want nil", *listed[2].Slot)
	}
}

func TestCombineRejectsMissingBranchSlot(t *testing.T) {
	worktrees := []repository.Worktree{{Path: "/repo", Branch: "main", Commit: "1111111"}}

	_, err := combine(worktrees, allocation.State{Slots: map[string]int{}})
	if err == nil {
		t.Fatal("combine() error = nil, want missing slot error")
	}
	if got, want := err.Error(), `branch "main" has no allocated slot`; got != want {
		t.Fatalf("combine() error = %q, want %q", got, want)
	}
}
