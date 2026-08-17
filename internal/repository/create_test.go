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
