package session

import (
	"fmt"

	"github.com/camdenwithrow/redwood/internal/repository"
	"github.com/camdenwithrow/redwood/internal/tmux"
)

type detachedStarter interface {
	StartDetached(name string) error
}

func Start(repo repository.Repository, branch string) (string, error) {
	return start(repo, branch, tmux.NewClient())
}

func start(repo repository.Repository, branch string, client detachedStarter) (string, error) {
	worktrees, err := repository.ListWorktrees(repo)
	if err != nil {
		return "", err
	}
	for _, worktree := range worktrees {
		if worktree.Branch != branch {
			continue
		}

		name := Name(repo, worktree)
		if err := client.StartDetached(name); err != nil {
			return "", fmt.Errorf("start tmux session for branch %q: %w", branch, err)
		}
		return name, nil
	}

	return "", fmt.Errorf("branch %q has no worktree", branch)
}
