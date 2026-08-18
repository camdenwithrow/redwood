package worktree

import (
	"fmt"

	"github.com/camdenwithrow/redwood/internal/allocation"
	"github.com/camdenwithrow/redwood/internal/repository"
	"github.com/camdenwithrow/redwood/internal/session"
	"github.com/camdenwithrow/redwood/internal/tmux"
)

type sessionCleaner interface {
	HasSession(name string) (bool, error)
	Stop(name string) error
}

func Remove(repo repository.Repository, branch string) (repository.Worktree, error) {
	return remove(repo, branch, tmux.NewClient())
}

func remove(repo repository.Repository, branch string, cleaner sessionCleaner) (repository.Worktree, error) {
	worktrees, err := repository.ListWorktrees(repo)
	if err != nil {
		return repository.Worktree{}, err
	}

	for _, worktree := range worktrees {
		if worktree.Branch != branch {
			continue
		}
		name := session.Name(repo, worktree)
		running, err := cleaner.HasSession(name)
		if err != nil {
			return repository.Worktree{}, fmt.Errorf("check tmux session for branch %q: %w", branch, err)
		}
		if running {
			if err := cleaner.Stop(name); err != nil {
				return repository.Worktree{}, fmt.Errorf("stop tmux session for branch %q: %w", branch, err)
			}
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
