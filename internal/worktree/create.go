package worktree

import (
	"fmt"

	"github.com/camdenwithrow/redwood/internal/allocation"
	"github.com/camdenwithrow/redwood/internal/config"
	"github.com/camdenwithrow/redwood/internal/repository"
)

type Created struct {
	Worktree repository.Worktree
	Slot     int
}

func Create(repo repository.Repository, configuration config.Config, branch string) (Created, error) {
	path, err := repository.ResolveWorktreePath(repo, configuration.WorktreePath, branch)
	if err != nil {
		return Created{}, err
	}

	createdWorktree, err := repository.CreateWorktree(repo, branch, path, configuration.BaseBranch)
	if err != nil {
		return Created{}, err
	}

	worktrees, err := repository.ListWorktrees(repo)
	if err != nil {
		return Created{}, err
	}
	state, err := allocation.NewStore(repo).Reconcile(worktrees)
	if err != nil {
		return Created{}, fmt.Errorf("allocate worktree slot: %w", err)
	}
	slot, exists := state.Slots[branch]
	if !exists {
		return Created{}, fmt.Errorf("allocation state has no slot for branch %q", branch)
	}

	return Created{Worktree: createdWorktree, Slot: slot}, nil
}
