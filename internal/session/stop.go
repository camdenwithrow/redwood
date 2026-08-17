package session

import (
	"fmt"

	"github.com/camdenwithrow/redwood/internal/repository"
	"github.com/camdenwithrow/redwood/internal/tmux"
)

type stopper interface {
	Stop(name string) error
}

func Stop(repo repository.Repository, branch string) (string, error) {
	return stop(repo, branch, tmux.NewClient())
}

func stop(repo repository.Repository, branch string, client stopper) (string, error) {
	worktree, err := worktreeForBranch(repo, branch)
	if err != nil {
		return "", err
	}
	name := Name(repo, worktree)
	if err := client.Stop(name); err != nil {
		return "", fmt.Errorf("stop tmux session for branch %q: %w", branch, err)
	}
	return name, nil
}
