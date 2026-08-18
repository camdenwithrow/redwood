package repository

import gitexec "github.com/camdenwithrow/redwood/internal/git"

func RemoveWorktree(repo Repository, path string) error {
	return gitexec.NewRunner(repo.MainCheckout).Run("worktree", "remove", path)
}
