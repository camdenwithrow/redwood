package repository

import (
	"os/exec"
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

func TestCreateWorktreeCreatesBranchFromBase(t *testing.T) {
	repositoryRoot := initializeCommittedRepository(t, "main")
	repo, err := DiscoverFrom(repositoryRoot)
	if err != nil {
		t.Fatalf("DiscoverFrom() error = %v", err)
	}
	worktreePath := filepath.Join(t.TempDir(), "feature-new")

	worktree, err := CreateWorktree(repo, "feature/new", worktreePath, "main")
	if err != nil {
		t.Fatalf("CreateWorktree() error = %v", err)
	}
	canonicalPath, err := filepath.EvalSymlinks(worktreePath)
	if err != nil {
		t.Fatalf("resolve worktree path: %v", err)
	}
	if worktree.Path != canonicalPath || worktree.Branch != "feature/new" || worktree.Commit == "" {
		t.Fatalf("CreateWorktree() = %+v", worktree)
	}
	if got, want := gitOutputForTest(t, repositoryRoot, "rev-parse", "feature/new"), gitOutputForTest(t, repositoryRoot, "rev-parse", "main"); got != want {
		t.Fatalf("created branch commit = %q, want base commit %q", got, want)
	}
}

func TestCreateWorktreeChecksOutExistingBranch(t *testing.T) {
	repositoryRoot := initializeCommittedRepository(t, "main")
	runGit(t, repositoryRoot, "branch", "feature/existing")
	repo, err := DiscoverFrom(repositoryRoot)
	if err != nil {
		t.Fatalf("DiscoverFrom() error = %v", err)
	}
	worktreePath := filepath.Join(t.TempDir(), "feature-existing")

	worktree, err := CreateWorktree(repo, "feature/existing", worktreePath, "main")
	if err != nil {
		t.Fatalf("CreateWorktree() error = %v", err)
	}
	if worktree.Branch != "feature/existing" {
		t.Fatalf("CreateWorktree() branch = %q, want feature/existing", worktree.Branch)
	}
}

func TestCreateWorktreeRejectsExistingPath(t *testing.T) {
	repositoryRoot := initializeCommittedRepository(t, "main")
	repo, err := DiscoverFrom(repositoryRoot)
	if err != nil {
		t.Fatalf("DiscoverFrom() error = %v", err)
	}
	existingPath := t.TempDir()

	_, err = CreateWorktree(repo, "feature/new", existingPath, "main")
	if err == nil {
		t.Fatal("CreateWorktree() error = nil, want existing path error")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateWorktree() error = %q, want existing path context", err)
	}
}

func TestResolveWorktreePath(t *testing.T) {
	repo := Repository{
		Name:         "redwood",
		MainCheckout: filepath.Join(string(filepath.Separator), "projects", "redwood"),
	}

	path, err := ResolveWorktreePath(repo, "../{repo}-{branch}", "feature/foo")
	if err != nil {
		t.Fatalf("ResolveWorktreePath() error = %v", err)
	}
	want := filepath.Join(string(filepath.Separator), "projects", "redwood-feature-foo")
	if path != want {
		t.Fatalf("ResolveWorktreePath() = %q, want %q", path, want)
	}
}

func TestResolveWorktreePathKeepsAbsoluteTemplate(t *testing.T) {
	repo := Repository{Name: "redwood", MainCheckout: "/projects/redwood"}
	template := filepath.Join(string(filepath.Separator), "worktrees", "{repo}", "{branch}")

	path, err := ResolveWorktreePath(repo, template, "fix/windows\\path")
	if err != nil {
		t.Fatalf("ResolveWorktreePath() error = %v", err)
	}
	want := filepath.Join(string(filepath.Separator), "worktrees", "redwood", "fix-windows-path")
	if path != want {
		t.Fatalf("ResolveWorktreePath() = %q, want %q", path, want)
	}
}

func TestResolveWorktreePathRejectsUnsafeResult(t *testing.T) {
	repo := Repository{Name: "redwood", MainCheckout: "/projects/redwood"}
	tests := []struct {
		name     string
		template string
		want     string
	}{
		{name: "unknown placeholder", template: "../{repo}-{unknown}-{branch}", want: "unresolved placeholder"},
		{name: "main checkout", template: ".", want: "resolves to the main checkout"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ResolveWorktreePath(repo, test.template, "feature/foo")
			if err == nil {
				t.Fatal("ResolveWorktreePath() error = nil, want path error")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ResolveWorktreePath() error = %q, want it to contain %q", err, test.want)
			}
		})
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

func gitOutputForTest(t *testing.T, directory string, args ...string) string {
	t.Helper()

	commandArgs := append([]string{"-C", directory}, args...)
	command := exec.Command("git", commandArgs...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}
