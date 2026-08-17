package allocation

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/camdenwithrow/redwood/internal/repository"
)

const (
	directoryName = "redwood"
	fileName      = "allocations.toml"
	formatVersion = 1
)

// State is the persisted mapping from branch names to stable numeric slots.
type State struct {
	Version int            `toml:"version"`
	Slots   map[string]int `toml:"slots"`
}

// Store persists allocation state inside a repository's shared Git directory.
type Store struct {
	path string
}

// NewStore creates an allocation store for repo.
func NewStore(repo repository.Repository) Store {
	return Store{path: filepath.Join(repo.GitDir, directoryName, fileName)}
}

// Path returns the allocation file path.
func (store Store) Path() string {
	return store.path
}

// Load reads allocation state, returning an empty state when no file exists.
func (store Store) Load() (State, error) {
	loaded := newState()
	metadata, err := toml.DecodeFile(store.path, &loaded)
	if errors.Is(err, os.ErrNotExist) {
		return loaded, nil
	}
	if err != nil {
		return State{}, fmt.Errorf("load allocations from %s: %w", store.path, err)
	}

	if undecoded := metadata.Undecoded(); len(undecoded) > 0 {
		keys := make([]string, 0, len(undecoded))
		for _, key := range undecoded {
			keys = append(keys, key.String())
		}
		sort.Strings(keys)

		return State{}, fmt.Errorf("load allocations from %s: unknown field(s): %s", store.path, strings.Join(keys, ", "))
	}
	if loaded.Slots == nil {
		loaded.Slots = make(map[string]int)
	}
	if err := loaded.validate(); err != nil {
		return State{}, fmt.Errorf("load allocations from %s: %w", store.path, err)
	}

	return loaded, nil
}

// Save atomically writes allocation state to the shared Git directory.
func (store Store) Save(state State) error {
	if err := state.validate(); err != nil {
		return fmt.Errorf("save allocations to %s: %w", store.path, err)
	}

	directory := filepath.Dir(store.path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create allocation directory %s: %w", directory, err)
	}

	temporaryFile, err := os.CreateTemp(directory, "allocations-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary allocation file: %w", err)
	}
	temporaryPath := temporaryFile.Name()
	defer os.Remove(temporaryPath)
	defer temporaryFile.Close()

	if err := temporaryFile.Chmod(0o644); err != nil {
		return fmt.Errorf("set allocation file permissions: %w", err)
	}
	if err := toml.NewEncoder(temporaryFile).Encode(state); err != nil {
		return fmt.Errorf("encode allocations: %w", err)
	}
	if err := temporaryFile.Sync(); err != nil {
		return fmt.Errorf("sync allocations: %w", err)
	}
	if err := temporaryFile.Close(); err != nil {
		return fmt.Errorf("close allocations: %w", err)
	}
	if err := os.Rename(temporaryPath, store.path); err != nil {
		return fmt.Errorf("replace allocation file %s: %w", store.path, err)
	}

	return nil
}

func newState() State {
	return State{
		Version: formatVersion,
		Slots:   make(map[string]int),
	}
}

func (state State) validate() error {
	if state.Version != formatVersion {
		return fmt.Errorf("unsupported allocation version %d", state.Version)
	}

	branches := make([]string, 0, len(state.Slots))
	for branch := range state.Slots {
		branches = append(branches, branch)
	}
	sort.Strings(branches)

	usedSlots := make(map[int]string, len(state.Slots))
	for _, branch := range branches {
		slot := state.Slots[branch]
		if strings.TrimSpace(branch) == "" {
			return fmt.Errorf("branch name must not be empty")
		}
		if slot < 0 {
			return fmt.Errorf("slot for branch %q must not be negative", branch)
		}
		if existingBranch, exists := usedSlots[slot]; exists {
			return fmt.Errorf("branches %q and %q both use slot %d", existingBranch, branch, slot)
		}
		usedSlots[slot] = branch
	}

	return nil
}
