package worktree

import (
	"fmt"

	"github.com/camdenwithrow/redwood/internal/allocation"
	"github.com/camdenwithrow/redwood/internal/repository"
)

type Info struct {
	Worktree repository.Worktree
	Slot     *int
}

func List(repo repository.Repository) ([]Info, error) {
	worktrees, err := repository.ListWorktrees(repo)
	if err != nil {
		return nil, err
	}

	state, err := allocation.NewStore(repo).Reconcile(worktrees)
	if err != nil {
		return nil, fmt.Errorf("reconcile worktree slots: %w", err)
	}

	return combine(worktrees, state)
}

func combine(worktrees []repository.Worktree, state allocation.State) ([]Info, error) {
	listed := make([]Info, 0, len(worktrees))
	for _, discovered := range worktrees {
		entry := Info{Worktree: discovered}
		if !discovered.Detached {
			slot, exists := state.Slots[discovered.Branch]
			if !exists {
				return nil, fmt.Errorf("branch %q has no allocated slot", discovered.Branch)
			}
			entry.Slot = &slot
		}
		listed = append(listed, entry)
	}

	return listed, nil
}
