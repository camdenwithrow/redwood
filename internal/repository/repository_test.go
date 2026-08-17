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
	repositoryRoot := initializeCommittedRepository(t, "main")

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

func TestResolveBaseBranch(t *testing.T) {
	tests := []struct {
		name       string
		initial    string
		additional string
		configured string
		want       string
		wantError  string
	}{
		{name: "detect main", initial: "main", want: "main"},
		{name: "detect master", initial: "master", want: "master"},
		{name: "both candidates", initial: "main", additional: "master", wantError: `both "main" and "master" exist`},
		{name: "neither candidate", initial: "develop", wantError: `neither "main" nor "master" exists`},
		{name: "configured branch exists", initial: "develop", configured: "develop", want: "develop"},
		{name: "configured branch missing", initial: "main", configured: "develop", wantError: `configured base branch "develop" does not exist locally`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repositoryRoot := initializeCommittedRepository(t, test.initial)
			if test.additional != "" {
				runGit(t, repositoryRoot, "branch", test.additional)
			}

			resolved, err := ResolveBaseBranch(
				Repository{Root: repositoryRoot, GitDir: filepath.Join(repositoryRoot, ".git")},
				test.configured,
			)
			if test.wantError != "" {
				if err == nil {
					t.Fatalf("ResolveBaseBranch() error = nil, want %q", test.wantError)
				}
				if !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("ResolveBaseBranch() error = %q, want it to contain %q", err, test.wantError)
				}
				return
			}

			if err != nil {
				t.Fatalf("ResolveBaseBranch() error = %v", err)
			}
			if resolved != test.want {
				t.Fatalf("ResolveBaseBranch() = %q, want %q", resolved, test.want)
			}
		})
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
	return initializeRepositoryWithBranch(t, "main")
}

func initializeRepositoryWithBranch(t *testing.T, branch string) string {
	t.Helper()

	repositoryRoot := filepath.Join(t.TempDir(), "repository")
	command := exec.Command("git", "init", "-b", branch, repositoryRoot)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("initialize Git repository: %v: %s", err, output)
	}

	canonicalRoot, err := filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		t.Fatalf("resolve repository path: %v", err)
	}

	return canonicalRoot
}

func initializeCommittedRepository(t *testing.T, branch string) string {
	t.Helper()

	repositoryRoot := initializeRepositoryWithBranch(t, branch)
	commitFile := filepath.Join(repositoryRoot, "README.md")
	if err := os.WriteFile(commitFile, []byte("test repository\n"), 0o644); err != nil {
		t.Fatalf("write commit fixture: %v", err)
	}
	runGit(t, repositoryRoot, "add", "README.md")
	runGit(t, repositoryRoot, "-c", "user.name=Redwood Tests", "-c", "user.email=redwood@example.com", "commit", "-m", "Initial commit")

	return repositoryRoot
}

func runGit(t *testing.T, directory string, args ...string) {
	t.Helper()

	commandArgs := append([]string{"-C", directory}, args...)
	command := exec.Command("git", commandArgs...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
}
