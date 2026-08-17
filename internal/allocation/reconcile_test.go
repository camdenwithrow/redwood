package allocation

import (
	"testing"

	"github.com/camdenwithrow/redwood/internal/repository"
)

func TestStoreReconcileAllocatesDiscoveredBranches(t *testing.T) {
	store := testStore(t)
	worktrees := []repository.Worktree{
		{Path: "/repo", Branch: "main", Commit: "aaa"},
		{Path: "/repo-feature-a", Branch: "feature/a", Commit: "bbb"},
		{Path: "/repo-detached", Commit: "ccc", Detached: true},
	}

	state, err := store.Reconcile(worktrees)
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if len(state.Slots) != 2 {
		t.Fatalf("Reconcile() slots = %v, want two branch allocations", state.Slots)
	}
	if state.Slots["main"] != 0 || state.Slots["feature/a"] != 1 {
		t.Fatalf("Reconcile() slots = %v, want main=0 and feature/a=1", state.Slots)
	}
	if _, exists := state.Slots[""]; exists {
		t.Fatalf("Reconcile() allocated detached worktree: %v", state.Slots)
	}
}

func TestStoreReconcileKeepsStableSlotsAndReusesStaleSlot(t *testing.T) {
	store := testStore(t)
	initial := State{
		Version: formatVersion,
		Slots: map[string]int{
			"main":          0,
			"feature/stale": 1,
			"feature/a":     2,
		},
	}
	if err := store.Save(initial); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	worktrees := []repository.Worktree{
		{Path: "/repo", Branch: "main", Commit: "aaa"},
		{Path: "/repo-feature-a", Branch: "feature/a", Commit: "bbb"},
		{Path: "/repo-feature-b", Branch: "feature/b", Commit: "ccc"},
	}
	state, err := store.Reconcile(worktrees)
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	if _, exists := state.Slots["feature/stale"]; exists {
		t.Fatalf("Reconcile() kept stale allocation: %v", state.Slots)
	}
	if state.Slots["main"] != 0 || state.Slots["feature/a"] != 2 || state.Slots["feature/b"] != 1 {
		t.Fatalf("Reconcile() slots = %v, want stable slots and feature/b=1", state.Slots)
	}

	reloaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(reloaded.Slots) != 3 || reloaded.Slots["feature/b"] != 1 {
		t.Fatalf("Load() slots = %v, want reconciled state", reloaded.Slots)
	}
}

func TestStoreReconcileRejectsBranchlessWorktree(t *testing.T) {
	store := testStore(t)

	_, err := store.Reconcile([]repository.Worktree{{Path: "/repo", Commit: "aaa"}})
	if err == nil {
		t.Fatal("Reconcile() error = nil, want branchless worktree error")
	}
}
