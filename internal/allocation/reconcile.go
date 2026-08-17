package allocation

import (
	"fmt"

	"github.com/camdenwithrow/redwood/internal/repository"
)

func (store Store) Reconcile(worktrees []repository.Worktree) (State, error) {
	state, err := store.Load()
	if err != nil {
		return State{}, err
	}

	activeBranches := make(map[string]struct{}, len(worktrees))
	for _, worktree := range worktrees {
		if worktree.Detached {
			continue
		}
		if worktree.Branch == "" {
			return State{}, fmt.Errorf("worktree %q has no branch", worktree.Path)
		}
		activeBranches[worktree.Branch] = struct{}{}
	}

	changed := false
	for branch := range state.Slots {
		if _, active := activeBranches[branch]; active {
			continue
		}
		delete(state.Slots, branch)
		changed = true
	}

	assignedBranches := make(map[string]struct{}, len(activeBranches))
	for _, worktree := range worktrees {
		if worktree.Detached {
			continue
		}
		if _, assigned := assignedBranches[worktree.Branch]; assigned {
			continue
		}

		_, existed := state.Slots[worktree.Branch]
		if _, err := state.Assign(worktree.Branch); err != nil {
			return State{}, fmt.Errorf("reconcile branch %q: %w", worktree.Branch, err)
		}
		if !existed {
			changed = true
		}
		assignedBranches[worktree.Branch] = struct{}{}
	}

	if changed {
		if err := store.Save(state); err != nil {
			return State{}, err
		}
	}

	return state, nil
}
