package repository

import (
	"fmt"
	"strings"

	gitexec "github.com/camdenwithrow/redwood/internal/git"
)

const localBranchPrefix = "refs/heads/"

// Worktree describes one entry reported by git worktree list.
type Worktree struct {
	Path     string
	Commit   string
	Branch   string
	Detached bool
}

// ListWorktrees discovers all worktrees that share repo's Git directory.
func ListWorktrees(repo Repository) ([]Worktree, error) {
	output, err := gitexec.NewRunner(repo.MainCheckout).Output("worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("list Git worktrees: %w", err)
	}

	worktrees, err := parseWorktreeList(output)
	if err != nil {
		return nil, fmt.Errorf("parse Git worktrees: %w", err)
	}

	return worktrees, nil
}

func parseWorktreeList(output string) ([]Worktree, error) {
	if strings.TrimSpace(output) == "" {
		return nil, nil
	}

	var worktrees []Worktree
	var current *Worktree
	for lineNumber, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			if current != nil {
				updatedWorktrees, err := appendWorktree(worktrees, current)
				if err != nil {
					return nil, err
				}
				worktrees = updatedWorktrees
			}
			path := strings.TrimPrefix(line, "worktree ")
			if path == "" {
				return nil, fmt.Errorf("line %d: worktree path is empty", lineNumber+1)
			}
			current = &Worktree{Path: path}
		case line == "":
			if current != nil {
				updatedWorktrees, err := appendWorktree(worktrees, current)
				if err != nil {
					return nil, err
				}
				worktrees = updatedWorktrees
				current = nil
			}
		case current == nil:
			return nil, fmt.Errorf("line %d: property appears before worktree path", lineNumber+1)
		case strings.HasPrefix(line, "HEAD "):
			current.Commit = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			current.Branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), localBranchPrefix)
		case line == "detached":
			current.Detached = true
		}
	}

	if current != nil {
		updatedWorktrees, err := appendWorktree(worktrees, current)
		if err != nil {
			return nil, err
		}
		worktrees = updatedWorktrees
	}

	return worktrees, nil
}

func appendWorktree(worktrees []Worktree, worktree *Worktree) ([]Worktree, error) {
	if worktree.Path == "" {
		return nil, fmt.Errorf("worktree path is empty")
	}
	if worktree.Commit == "" {
		return nil, fmt.Errorf("worktree %q has no HEAD commit", worktree.Path)
	}
	if worktree.Branch == "" && !worktree.Detached {
		return nil, fmt.Errorf("worktree %q has neither a branch nor detached state", worktree.Path)
	}
	if worktree.Branch != "" && worktree.Detached {
		return nil, fmt.Errorf("worktree %q cannot have both a branch and detached state", worktree.Path)
	}

	return append(worktrees, *worktree), nil
}
