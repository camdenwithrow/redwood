package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gitexec "github.com/camdenwithrow/redwood/internal/git"
)

type CreatedWorktree struct {
	Worktree      Worktree
	BranchCreated bool
}

func CreateWorktree(repo Repository, branch, path, baseBranch string) (CreatedWorktree, error) {
	if err := ValidateNewWorktree(repo, branch); err != nil {
		return CreatedWorktree{}, err
	}
	if _, err := os.Lstat(path); err == nil {
		return CreatedWorktree{}, fmt.Errorf("worktree path %q already exists", path)
	} else if !os.IsNotExist(err) {
		return CreatedWorktree{}, fmt.Errorf("inspect worktree path %q: %w", path, err)
	}

	runner := gitexec.NewRunner(repo.MainCheckout)
	branchExists, err := LocalBranchExists(repo.MainCheckout, branch)
	if err != nil {
		return CreatedWorktree{}, err
	}
	if branchExists {
		err = runner.Run("worktree", "add", path, branch)
	} else {
		err = runner.Run("worktree", "add", "-b", branch, path, baseBranch)
	}
	if err != nil {
		cause := fmt.Errorf("create worktree for branch %q: %w", branch, err)
		return CreatedWorktree{}, errors.Join(cause, rollbackFailedCreation(repo, path, branch, !branchExists))
	}
	created := CreatedWorktree{BranchCreated: !branchExists}

	worktrees, err := ListWorktrees(repo)
	if err != nil {
		return CreatedWorktree{}, errors.Join(err, RollbackWorktree(repo, path, branch, created.BranchCreated))
	}
	for _, worktree := range worktrees {
		if worktree.Branch == branch {
			created.Worktree = worktree
			return created, nil
		}
	}

	cause := fmt.Errorf("created branch %q but Git did not report its worktree", branch)
	return CreatedWorktree{}, errors.Join(cause, RollbackWorktree(repo, path, branch, created.BranchCreated))
}

func RollbackWorktree(repo Repository, path, branch string, deleteBranch bool) error {
	runner := gitexec.NewRunner(repo.MainCheckout)
	removeErr := runner.Run("worktree", "remove", "--force", path)
	if !deleteBranch {
		return removeErr
	}
	deleteErr := runner.Run("branch", "-D", branch)
	return errors.Join(removeErr, deleteErr)
}

func rollbackFailedCreation(repo Repository, path, branch string, deleteBranch bool) error {
	var rollbackErrors []error
	registered := false
	worktrees, err := ListWorktrees(repo)
	if err != nil {
		rollbackErrors = append(rollbackErrors, err)
	} else {
		for _, worktree := range worktrees {
			if worktree.Branch == branch || filepath.Clean(worktree.Path) == filepath.Clean(path) {
				registered = true
				break
			}
		}
	}

	if registered {
		if err := gitexec.NewRunner(repo.MainCheckout).Run("worktree", "remove", "--force", path); err != nil {
			rollbackErrors = append(rollbackErrors, err)
		}
	}
	if deleteBranch {
		exists, err := LocalBranchExists(repo.MainCheckout, branch)
		if err != nil {
			rollbackErrors = append(rollbackErrors, err)
		} else if exists {
			if err := gitexec.NewRunner(repo.MainCheckout).Run("branch", "-D", branch); err != nil {
				rollbackErrors = append(rollbackErrors, err)
			}
		}
	}

	return errors.Join(rollbackErrors...)
}

func ResolveWorktreePath(repo Repository, template, branch string) (string, error) {
	branchPath := strings.NewReplacer("/", "-", "\\", "-").Replace(branch)
	expanded := strings.NewReplacer(
		"{repo}", repo.Name,
		"{branch}", branchPath,
	).Replace(template)
	if strings.ContainsAny(expanded, "{}") {
		return "", fmt.Errorf("worktree_path contains an unresolved placeholder: %q", expanded)
	}

	if !filepath.IsAbs(expanded) {
		expanded = filepath.Join(repo.MainCheckout, expanded)
	}
	expanded = filepath.Clean(expanded)
	if expanded == filepath.Clean(repo.MainCheckout) {
		return "", fmt.Errorf("worktree_path resolves to the main checkout %q", expanded)
	}

	return expanded, nil
}

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
