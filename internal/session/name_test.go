package session

import (
	"regexp"
	"testing"

	"github.com/camdenwithrow/redwood/internal/repository"
)

func TestNameIsDeterministicAndSafe(t *testing.T) {
	repo := repository.Repository{Name: "My Project", GitDir: "/projects/redwood/.git"}
	worktree := repository.Worktree{Path: "/projects/redwood-feature", Branch: "feature/Hello_World"}

	first := Name(repo, worktree)
	second := Name(repo, worktree)
	if first != second {
		t.Fatalf("Name() changed from %q to %q", first, second)
	}
	if !regexp.MustCompile(`^rw-[a-z0-9-]+-[a-f0-9]{12}$`).MatchString(first) {
		t.Fatalf("Name() = %q, want tmux-safe name", first)
	}
}

func TestNameAvoidsRepositoryAndWorktreeCollisions(t *testing.T) {
	worktree := repository.Worktree{Path: "/projects/redwood-feature", Branch: "feature/a"}
	firstRepo := repository.Repository{Name: "redwood", GitDir: "/projects/one/.git"}
	secondRepo := repository.Repository{Name: "redwood", GitDir: "/projects/two/.git"}

	if Name(firstRepo, worktree) == Name(secondRepo, worktree) {
		t.Fatal("Name() collided for repositories with different shared Git directories")
	}

	otherWorktree := worktree
	otherWorktree.Path = "/projects/redwood-feature-copy"
	if Name(firstRepo, worktree) == Name(firstRepo, otherWorktree) {
		t.Fatal("Name() collided for different worktree paths")
	}
}

func TestNameHandlesDetachedAndLongWorktrees(t *testing.T) {
	repo := repository.Repository{Name: "redwood", GitDir: "/projects/redwood/.git"}
	worktree := repository.Worktree{
		Path:   "/projects/redwood-detached",
		Commit: "0123456789abcdef",
		Branch: "feature/a-very-long-branch-name-that-keeps-going",
	}

	name := Name(repo, worktree)
	if len(name) > 52 {
		t.Fatalf("Name() length = %d, want at most 52", len(name))
	}
}
