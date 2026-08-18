package worktree

import (
	"fmt"

	"github.com/camdenwithrow/redwood/internal/allocation"
	"github.com/camdenwithrow/redwood/internal/repository"
)

func Remove(repo repository.Repository, branch string) (repository.Worktree, error) {
	worktrees, err := repository.ListWorktrees(repo)
	if err != nil {
		return repository.Worktree{}, err
	}

	for _, worktree := range worktrees {
		if worktree.Branch != branch {
			continue
		}
		if err := repository.RemoveWorktree(repo, worktree.Path); err != nil {
			return repository.Worktree{}, err
		}

		remaining, err := repository.ListWorktrees(repo)
		if err != nil {
			return repository.Worktree{}, err
		}
		if _, err := allocation.NewStore(repo).Reconcile(remaining); err != nil {
			return repository.Worktree{}, fmt.Errorf("reconcile worktree slots: %w", err)
		}
		return worktree, nil
	}

	return repository.Worktree{}, fmt.Errorf("branch %q has no worktree", branch)
}
