package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gitexec "github.com/camdenwithrow/redwood/internal/git"
)

func CreateWorktree(repo Repository, branch, path, baseBranch string) (Worktree, error) {
	if err := ValidateNewWorktree(repo, branch); err != nil {
		return Worktree{}, err
	}
	if _, err := os.Lstat(path); err == nil {
		return Worktree{}, fmt.Errorf("worktree path %q already exists", path)
	} else if !os.IsNotExist(err) {
		return Worktree{}, fmt.Errorf("inspect worktree path %q: %w", path, err)
	}

	runner := gitexec.NewRunner(repo.MainCheckout)
	branchExists, err := localBranchExists(repo.MainCheckout, branch)
	if err != nil {
		return Worktree{}, err
	}
	if branchExists {
		err = runner.Run("worktree", "add", path, branch)
	} else {
		err = runner.Run("worktree", "add", "-b", branch, path, baseBranch)
	}
	if err != nil {
		return Worktree{}, fmt.Errorf("create worktree for branch %q: %w", branch, err)
	}

	worktrees, err := ListWorktrees(repo)
	if err != nil {
		return Worktree{}, err
	}
	for _, worktree := range worktrees {
		if worktree.Branch == branch {
			return worktree, nil
		}
	}

	return Worktree{}, fmt.Errorf("created branch %q but Git did not report its worktree", branch)
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
