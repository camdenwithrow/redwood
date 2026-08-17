package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camdenwithrow/redwood/internal/allocation"
	"github.com/camdenwithrow/redwood/internal/config"
	"github.com/camdenwithrow/redwood/internal/repository"
)

func TestCreatePersistsSlotAfterWorktreeCreation(t *testing.T) {
	repositoryRoot := initializeRepository(t)
	repo, err := repository.DiscoverFrom(repositoryRoot)
	if err != nil {
		t.Fatalf("DiscoverFrom() error = %v", err)
	}
	configuration := config.Config{
		BaseBranch:   "main",
		WorktreePath: filepath.Join(t.TempDir(), "{repo}-{branch}"),
	}

	created, err := Create(repo, configuration, "feature/a")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Worktree.Branch != "feature/a" || created.Slot != 1 {
		t.Fatalf("Create() = %+v, want feature/a at slot 1", created)
	}
	if _, err := os.Stat(created.Worktree.Path); err != nil {
		t.Fatalf("created worktree path: %v", err)
	}

	state, err := allocation.NewStore(repo).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if state.Slots["main"] != 0 || state.Slots["feature/a"] != 1 {
		t.Fatalf("Load() slots = %v, want main=0 and feature/a=1", state.Slots)
	}
}

func TestCreateDoesNotAllocateWhenGitCreationFails(t *testing.T) {
	repositoryRoot := initializeRepository(t)
	repo, err := repository.DiscoverFrom(repositoryRoot)
	if err != nil {
		t.Fatalf("DiscoverFrom() error = %v", err)
	}
	existingPath := t.TempDir()
	configuration := config.Config{
		BaseBranch:   "main",
		WorktreePath: existingPath,
	}

	_, err = Create(repo, configuration, "feature/a")
	if err == nil {
		t.Fatal("Create() error = nil, want Git creation error")
	}
	state, err := allocation.NewStore(repo).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(state.Slots) != 0 {
		t.Fatalf("Load() slots = %v, want no allocation", state.Slots)
	}
}

func initializeRepository(t *testing.T) string {
	t.Helper()

	repositoryRoot := filepath.Join(t.TempDir(), "repository")
	command := exec.Command("git", "init", "-b", "main", repositoryRoot)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("initialize Git repository: %v: %s", err, output)
	}
	if err := os.WriteFile(filepath.Join(repositoryRoot, "README.md"), []byte("test\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	runGit(t, repositoryRoot, "add", "README.md")
	runGit(
		t,
		repositoryRoot,
		"-c", "user.name=Redwood Tests",
		"-c", "user.email=redwood@example.com",
		"commit", "-m", "Initial commit",
	)

	canonicalRoot, err := filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		t.Fatalf("resolve repository path: %v", err)
	}
	return canonicalRoot
}

func runGit(t *testing.T, directory string, args ...string) {
	t.Helper()

	commandArgs := append([]string{"-C", directory}, args...)
	command := exec.Command("git", commandArgs...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
}
