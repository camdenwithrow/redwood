package repository

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestListWorktrees(t *testing.T) {
	repositoryRoot := initializeCommittedRepository(t, "main")
	linkedPath := filepath.Join(t.TempDir(), "feature")
	runGit(t, repositoryRoot, "worktree", "add", "-b", "feature/test", linkedPath)
	canonicalLinkedPath, err := filepath.EvalSymlinks(linkedPath)
	if err != nil {
		t.Fatalf("resolve linked worktree path: %v", err)
	}

	worktrees, err := ListWorktrees(Repository{
		Name:         "repository",
		MainCheckout: repositoryRoot,
		GitDir:       filepath.Join(repositoryRoot, ".git"),
	})
	if err != nil {
		t.Fatalf("ListWorktrees() error = %v", err)
	}
	if len(worktrees) != 2 {
		t.Fatalf("ListWorktrees() returned %d worktrees, want 2", len(worktrees))
	}

	if worktrees[0].Path != repositoryRoot || worktrees[0].Branch != "main" {
		t.Fatalf("ListWorktrees() main worktree = %+v", worktrees[0])
	}
	if worktrees[1].Path != canonicalLinkedPath || worktrees[1].Branch != "feature/test" {
		t.Fatalf("ListWorktrees() linked worktree = %+v", worktrees[1])
	}
	if worktrees[0].Commit == "" || worktrees[1].Commit == "" {
		t.Fatalf("ListWorktrees() returned an empty commit: %+v", worktrees)
	}
}

func TestParseWorktreeList(t *testing.T) {
	const porcelain = `worktree /projects/redwood
HEAD 0123456789abcdef
branch refs/heads/main

worktree /projects/redwood-feature
HEAD fedcba9876543210
detached
locked maintenance
`

	worktrees, err := parseWorktreeList(porcelain)
	if err != nil {
		t.Fatalf("parseWorktreeList() error = %v", err)
	}
	if len(worktrees) != 2 {
		t.Fatalf("parseWorktreeList() returned %d worktrees, want 2", len(worktrees))
	}
	if got := worktrees[0]; got.Path != "/projects/redwood" || got.Commit != "0123456789abcdef" || got.Branch != "main" || got.Detached {
		t.Fatalf("parseWorktreeList() main worktree = %+v", got)
	}
	if got := worktrees[1]; got.Path != "/projects/redwood-feature" || got.Commit != "fedcba9876543210" || got.Branch != "" || !got.Detached {
		t.Fatalf("parseWorktreeList() detached worktree = %+v", got)
	}
}

func TestParseWorktreeListRejectsMalformedOutput(t *testing.T) {
	_, err := parseWorktreeList("HEAD 0123456789abcdef")
	if err == nil {
		t.Fatal("parseWorktreeList() error = nil, want malformed output error")
	}
	if !strings.Contains(err.Error(), "property appears before worktree path") {
		t.Fatalf("parseWorktreeList() error = %q, want property ordering context", err)
	}
}
