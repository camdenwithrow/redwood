package repository

import (
	"fmt"
	"path/filepath"
	"strings"

	gitexec "github.com/camdenwithrow/redwood/internal/git"
)

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
