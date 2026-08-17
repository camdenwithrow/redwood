package repository

import (
	"fmt"

	gitexec "github.com/camdenwithrow/redwood/internal/git"
)

func ValidateNewWorktree(repo Repository, branch string) error {
	if err := gitexec.NewRunner(repo.MainCheckout).Run("check-ref-format", "--branch", branch); err != nil {
		return fmt.Errorf("invalid branch name %q: %w", branch, err)
	}

	worktrees, err := ListWorktrees(repo)
	if err != nil {
		return err
	}
	for _, worktree := range worktrees {
		if worktree.Branch == branch {
			return fmt.Errorf("branch %q already has a worktree at %q", branch, worktree.Path)
		}
	}

	return nil
}
