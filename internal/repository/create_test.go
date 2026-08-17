package repository

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateNewWorktree(t *testing.T) {
	repositoryRoot := initializeCommittedRepository(t, "main")
	repo, err := DiscoverFrom(repositoryRoot)
	if err != nil {
		t.Fatalf("DiscoverFrom() error = %v", err)
	}
	runGit(t, repositoryRoot, "branch", "feature/available")

	if err := ValidateNewWorktree(repo, "feature/available"); err != nil {
		t.Fatalf("ValidateNewWorktree() error = %v", err)
	}
}

func TestValidateNewWorktreeRejectsInvalidBranch(t *testing.T) {
	repositoryRoot := initializeCommittedRepository(t, "main")
	repo, err := DiscoverFrom(repositoryRoot)
	if err != nil {
		t.Fatalf("DiscoverFrom() error = %v", err)
	}

	err = ValidateNewWorktree(repo, "bad..branch")
	if err == nil {
		t.Fatal("ValidateNewWorktree() error = nil, want invalid branch error")
	}
	if !strings.Contains(err.Error(), `invalid branch name "bad..branch"`) {
		t.Fatalf("ValidateNewWorktree() error = %q, want branch context", err)
	}
}

func TestValidateNewWorktreeRejectsDuplicateWorktree(t *testing.T) {
	repositoryRoot := initializeCommittedRepository(t, "main")
	linkedPath := filepath.Join(t.TempDir(), "feature")
	runGit(t, repositoryRoot, "worktree", "add", "-b", "feature/existing", linkedPath)
	repo, err := DiscoverFrom(repositoryRoot)
	if err != nil {
		t.Fatalf("DiscoverFrom() error = %v", err)
	}

	err = ValidateNewWorktree(repo, "feature/existing")
	if err == nil {
		t.Fatal("ValidateNewWorktree() error = nil, want duplicate worktree error")
	}
	if !strings.Contains(err.Error(), `branch "feature/existing" already has a worktree`) {
		t.Fatalf("ValidateNewWorktree() error = %q, want duplicate branch context", err)
	}
}
