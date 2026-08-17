package allocation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camdenwithrow/redwood/internal/repository"
)

func TestNewStoreUsesSharedGitDirectory(t *testing.T) {
	repositoryRoot := t.TempDir()
	gitDirectory := filepath.Join(repositoryRoot, ".git")
	store := NewStore(repository.Repository{
		Name:         "redwood",
		MainCheckout: repositoryRoot,
		GitDir:       gitDirectory,
	})

	want := filepath.Join(gitDirectory, "redwood", "allocations.toml")
	if store.Path() != want {
		t.Fatalf("Path() = %q, want %q", store.Path(), want)
	}
}

func TestStoreLoadMissingFile(t *testing.T) {
	store := testStore(t)

	state, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if state.Version != formatVersion {
		t.Fatalf("Load() version = %d, want %d", state.Version, formatVersion)
	}
	if len(state.Slots) != 0 {
		t.Fatalf("Load() slots = %v, want empty map", state.Slots)
	}
}

func TestStoreRoundTrip(t *testing.T) {
	store := testStore(t)
	want := State{
		Version: formatVersion,
		Slots: map[string]int{
			"feature/a": 1,
			"feature/b": 2,
		},
	}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Version != want.Version || len(got.Slots) != len(want.Slots) {
		t.Fatalf("Load() = %+v, want %+v", got, want)
	}
	for branch, slot := range want.Slots {
		if got.Slots[branch] != slot {
			t.Fatalf("Load() slot for %q = %d, want %d", branch, got.Slots[branch], slot)
		}
	}

	contents, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatalf("read allocation file: %v", err)
	}
	for _, expected := range []string{"version = 1", "[slots]", `"feature/a" = 1`, `"feature/b" = 2`} {
		if !strings.Contains(string(contents), expected) {
			t.Fatalf("allocation file = %q, want it to contain %q", contents, expected)
		}
	}
}

func TestStoreRejectsDuplicateSlots(t *testing.T) {
	store := testStore(t)
	state := State{
		Version: formatVersion,
		Slots: map[string]int{
			"feature/a": 1,
			"feature/b": 1,
		},
	}

	err := store.Save(state)
	if err == nil {
		t.Fatal("Save() error = nil, want duplicate slot error")
	}
	if !strings.Contains(err.Error(), "both use slot 1") {
		t.Fatalf("Save() error = %q, want duplicate slot context", err)
	}
}

func testStore(t *testing.T) Store {
	t.Helper()

	gitDirectory := filepath.Join(t.TempDir(), ".git")
	return NewStore(repository.Repository{GitDir: gitDirectory})
}
