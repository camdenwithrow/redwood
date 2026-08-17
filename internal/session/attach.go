package session

import (
	"fmt"

	"github.com/camdenwithrow/redwood/internal/repository"
	"github.com/camdenwithrow/redwood/internal/tmux"
)

type attacher interface {
	Attach(name string) error
}

func Attach(repo repository.Repository, branch string) error {
	return attach(repo, branch, tmux.NewClient())
}

func attach(repo repository.Repository, branch string, client attacher) error {
	worktree, err := worktreeForBranch(repo, branch)
	if err != nil {
		return err
	}
	name := Name(repo, worktree)
	if err := client.Attach(name); err != nil {
		return fmt.Errorf("attach to tmux session for branch %q: %w", branch, err)
	}
	return nil
}
