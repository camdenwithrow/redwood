package worktree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camdenwithrow/redwood/internal/allocation"
	"github.com/camdenwithrow/redwood/internal/repository"
)

func TestRemoveDeletesWorktreeAndReleasesSlot(t *testing.T) {
	repositoryRoot := initializeRepository(t)
	worktreePath := filepath.Join(t.TempDir(), "feature-a")
	runGit(t, repositoryRoot, "worktree", "add", "-b", "feature/a", worktreePath)
	canonicalWorktreePath, err := filepath.EvalSymlinks(worktreePath)
	if err != nil {
		t.Fatalf("resolve worktree path: %v", err)
	}
	repo, err := repository.DiscoverFrom(repositoryRoot)
	if err != nil {
		t.Fatalf("DiscoverFrom() error = %v", err)
	}
	worktrees, err := repository.ListWorktrees(repo)
	if err != nil {
		t.Fatalf("ListWorktrees() error = %v", err)
	}
	if _, err := allocation.NewStore(repo).Reconcile(worktrees); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	removed, err := Remove(repo, "feature/a")
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if removed.Branch != "feature/a" || removed.Path != canonicalWorktreePath {
		t.Fatalf("Remove() = %+v, want feature/a at %q", removed, canonicalWorktreePath)
	}
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Fatalf("removed worktree path error = %v, want not found", err)
	}
	if !branchExists(t, repositoryRoot, "feature/a") {
		t.Fatal("Remove() deleted the worktree branch")
	}
	state, err := allocation.NewStore(repo).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if _, exists := state.Slots["feature/a"]; exists {
		t.Fatalf("Load() slots = %v, want feature/a released", state.Slots)
	}
}

func TestRemovePassesThroughGitErrorForDirtyWorktree(t *testing.T) {
	repositoryRoot := initializeRepository(t)
	worktreePath := filepath.Join(t.TempDir(), "feature-dirty")
	runGit(t, repositoryRoot, "worktree", "add", "-b", "feature/dirty", worktreePath)
	if err := os.WriteFile(filepath.Join(worktreePath, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("modify worktree: %v", err)
	}
	repo, err := repository.DiscoverFrom(repositoryRoot)
	if err != nil {
		t.Fatalf("DiscoverFrom() error = %v", err)
	}

	_, err = Remove(repo, "feature/dirty")
	if err == nil {
		t.Fatal("Remove() error = nil, want Git refusal")
	}
	if !strings.Contains(err.Error(), "git worktree remove") || !strings.Contains(err.Error(), "contains modified or untracked files") {
		t.Fatalf("Remove() error = %q, want Git worktree error", err)
	}
	if _, statErr := os.Stat(worktreePath); statErr != nil {
		t.Fatalf("dirty worktree was removed: %v", statErr)
	}
}

func TestRemoveRejectsMissingWorktree(t *testing.T) {
	repositoryRoot := initializeRepository(t)
	repo, err := repository.DiscoverFrom(repositoryRoot)
	if err != nil {
		t.Fatalf("DiscoverFrom() error = %v", err)
	}

	_, err = Remove(repo, "feature/missing")
	if err == nil || err.Error() != `branch "feature/missing" has no worktree` {
		t.Fatalf("Remove() error = %v, want missing worktree error", err)
	}
}
