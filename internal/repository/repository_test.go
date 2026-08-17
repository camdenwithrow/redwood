package repository

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverFromMainCheckout(t *testing.T) {
	repositoryRoot := initializeRepository(t)
	nestedDirectory := filepath.Join(repositoryRoot, "one", "two")
	if err := os.MkdirAll(nestedDirectory, 0o755); err != nil {
		t.Fatalf("create nested directory: %v", err)
	}

	discovered, err := DiscoverFrom(nestedDirectory)
	if err != nil {
		t.Fatalf("DiscoverFrom() error = %v", err)
	}
	if discovered.Root != repositoryRoot {
		t.Fatalf("DiscoverFrom() root = %q, want %q", discovered.Root, repositoryRoot)
	}
	if want := filepath.Join(repositoryRoot, ".git"); discovered.GitDir != want {
		t.Fatalf("DiscoverFrom() GitDir = %q, want %q", discovered.GitDir, want)
	}
}

func TestDiscoverFromRejectsLinkedWorktree(t *testing.T) {
	repositoryRoot := initializeRepository(t)
	commitFile := filepath.Join(repositoryRoot, "README.md")
	if err := os.WriteFile(commitFile, []byte("test repository\n"), 0o644); err != nil {
		t.Fatalf("write commit fixture: %v", err)
	}
	runGit(t, repositoryRoot, "add", "README.md")
	runGit(t, repositoryRoot, "-c", "user.name=Redwood Tests", "-c", "user.email=redwood@example.com", "commit", "-m", "Initial commit")

	worktreePath := filepath.Join(t.TempDir(), "linked")
	runGit(t, repositoryRoot, "worktree", "add", "-b", "feature/test", worktreePath)

	_, err := DiscoverFrom(worktreePath)
	if err == nil {
		t.Fatal("DiscoverFrom() error = nil, want linked worktree error")
	}
	if !strings.Contains(err.Error(), "is not the main checkout") {
		t.Fatalf("DiscoverFrom() error = %q, want main checkout guidance", err)
	}
	if !strings.Contains(err.Error(), repositoryRoot) {
		t.Fatalf("DiscoverFrom() error = %q, want main checkout path %q", err, repositoryRoot)
	}
}

func TestDiscoverFromRejectsNonRepository(t *testing.T) {
	_, err := DiscoverFrom(t.TempDir())
	if err == nil {
		t.Fatal("DiscoverFrom() error = nil, want repository error")
	}
	if !strings.Contains(err.Error(), "locate repository root") {
		t.Fatalf("DiscoverFrom() error = %q, want repository guidance", err)
	}
}

func initializeRepository(t *testing.T) string {
	t.Helper()

	repositoryRoot := filepath.Join(t.TempDir(), "repository")
	command := exec.Command("git", "init", "-b", "main", repositoryRoot)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("initialize Git repository: %v: %s", err, output)
	}

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
