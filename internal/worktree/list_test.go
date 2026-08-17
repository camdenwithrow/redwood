package worktree

import (
	"errors"
	"slices"
	"testing"

	"github.com/camdenwithrow/redwood/internal/allocation"
	"github.com/camdenwithrow/redwood/internal/repository"
	"github.com/camdenwithrow/redwood/internal/session"
)

type sessionCheckerFunc func(name string) (bool, error)

func (check sessionCheckerFunc) HasSession(name string) (bool, error) {
	return check(name)
}

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

func TestAddRunningStateChecksEveryWorktreeSession(t *testing.T) {
	repo := repository.Repository{Name: "redwood", GitDir: "/repo/.git"}
	listed := []Info{
		{Worktree: repository.Worktree{Path: "/repo", Branch: "main"}},
		{Worktree: repository.Worktree{Path: "/repo-feature-a", Branch: "feature/a"}},
	}
	runningName := session.Name(repo, listed[1].Worktree)
	var checked []string
	checker := sessionCheckerFunc(func(name string) (bool, error) {
		checked = append(checked, name)
		return name == runningName, nil
	})

	withState, err := addRunningState(repo, listed, checker)
	if err != nil {
		t.Fatalf("addRunningState() error = %v", err)
	}
	wantNames := []string{
		session.Name(repo, listed[0].Worktree),
		session.Name(repo, listed[1].Worktree),
	}
	if !slices.Equal(checked, wantNames) {
		t.Fatalf("addRunningState() checked = %v, want %v", checked, wantNames)
	}
	if withState[0].Running || !withState[1].Running {
		t.Fatalf("addRunningState() running states = %v, %v; want false, true", withState[0].Running, withState[1].Running)
	}
}

func TestAddRunningStateReportsTmuxError(t *testing.T) {
	listed := []Info{{Worktree: repository.Worktree{Path: "/repo", Branch: "main"}}}
	checker := sessionCheckerFunc(func(string) (bool, error) {
		return false, errors.New("tmux unavailable")
	})

	_, err := addRunningState(repository.Repository{}, listed, checker)
	if err == nil {
		t.Fatal("addRunningState() error = nil, want tmux error")
	}
	if got, want := err.Error(), `check tmux session for worktree "/repo": tmux unavailable`; got != want {
		t.Fatalf("addRunningState() error = %q, want %q", got, want)
	}
}
