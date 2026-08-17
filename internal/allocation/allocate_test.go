package allocation

import (
	"path/filepath"
	"testing"

	"github.com/camdenwithrow/redwood/internal/repository"
)

func TestStoreAssignPersistsStableSlots(t *testing.T) {
	gitDirectory := filepath.Join(t.TempDir(), ".git")
	repo := repository.Repository{GitDir: gitDirectory}

	firstInvocation := NewStore(repo)
	firstSlot, err := firstInvocation.Assign("feature/a")
	if err != nil {
		t.Fatalf("first Assign() error = %v", err)
	}
	if firstSlot != 0 {
		t.Fatalf("first Assign() slot = %d, want 0", firstSlot)
	}

	secondInvocation := NewStore(repo)
	repeatedSlot, err := secondInvocation.Assign("feature/a")
	if err != nil {
		t.Fatalf("repeated Assign() error = %v", err)
	}
	if repeatedSlot != firstSlot {
		t.Fatalf("repeated Assign() slot = %d, want stable slot %d", repeatedSlot, firstSlot)
	}

	secondSlot, err := secondInvocation.Assign("feature/b")
	if err != nil {
		t.Fatalf("second branch Assign() error = %v", err)
	}
	if secondSlot != 1 {
		t.Fatalf("second branch Assign() slot = %d, want 1", secondSlot)
	}

	thirdInvocation := NewStore(repo)
	state, err := thirdInvocation.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if state.Slots["feature/a"] != 0 || state.Slots["feature/b"] != 1 {
		t.Fatalf("Load() slots = %v, want persisted allocations", state.Slots)
	}
}

func TestStateAssignsLowestAvailableSlot(t *testing.T) {
	tests := []struct {
		name  string
		slots map[string]int
		want  int
	}{
		{name: "nil state", want: 0},
		{name: "empty state", slots: map[string]int{}, want: 0},
		{name: "next sequential slot", slots: map[string]int{"main": 0, "feature/a": 1}, want: 2},
		{name: "fill lowest gap", slots: map[string]int{"main": 0, "feature/b": 2}, want: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := State{Version: formatVersion, Slots: test.slots}

			slot, err := state.Assign("feature/new")
			if err != nil {
				t.Fatalf("Assign() error = %v", err)
			}
			if slot != test.want {
				t.Fatalf("Assign() slot = %d, want %d", slot, test.want)
			}
			if state.Slots["feature/new"] != test.want {
				t.Fatalf("Assign() stored slot = %d, want %d", state.Slots["feature/new"], test.want)
			}
		})
	}
}

func TestStateAssignReturnsExistingSlot(t *testing.T) {
	state := State{
		Version: formatVersion,
		Slots:   map[string]int{"feature/a": 3},
	}

	slot, err := state.Assign("feature/a")
	if err != nil {
		t.Fatalf("Assign() error = %v", err)
	}
	if slot != 3 {
		t.Fatalf("Assign() slot = %d, want 3", slot)
	}
	if len(state.Slots) != 1 {
		t.Fatalf("Assign() changed slots = %v", state.Slots)
	}
}

func TestStateAssignRejectsEmptyBranch(t *testing.T) {
	state := newState()

	if _, err := state.Assign(" "); err == nil {
		t.Fatal("Assign() error = nil, want empty branch error")
	}
	if len(state.Slots) != 0 {
		t.Fatalf("Assign() changed slots = %v", state.Slots)
	}
}
