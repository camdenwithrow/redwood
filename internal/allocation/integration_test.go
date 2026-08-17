package allocation

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camdenwithrow/redwood/internal/config"
	"github.com/camdenwithrow/redwood/internal/repository"
)

func TestMultipleWorktreeAllocationsRemainStableWithUniquePorts(t *testing.T) {
	repositoryRoot := initializeGitRepository(t)
	featureAPath := filepath.Join(t.TempDir(), "feature-a")
	featureBPath := filepath.Join(t.TempDir(), "feature-b")
	runGitCommand(t, repositoryRoot, "worktree", "add", "-b", "feature/a", featureAPath)
	runGitCommand(t, repositoryRoot, "worktree", "add", "-b", "feature/b", featureBPath)

	repo, err := repository.DiscoverFrom(repositoryRoot)
	if err != nil {
		t.Fatalf("DiscoverFrom() error = %v", err)
	}
	worktrees, err := repository.ListWorktrees(repo)
	if err != nil {
		t.Fatalf("ListWorktrees() error = %v", err)
	}
	if len(worktrees) != 3 {
		t.Fatalf("ListWorktrees() returned %d worktrees, want 3", len(worktrees))
	}

	firstInvocation := NewStore(repo)
	firstState, err := firstInvocation.Reconcile(worktrees)
	if err != nil {
		t.Fatalf("first Reconcile() error = %v", err)
	}

	secondInvocation := NewStore(repo)
	secondState, err := secondInvocation.Reconcile(worktrees)
	if err != nil {
		t.Fatalf("second Reconcile() error = %v", err)
	}
	for _, branch := range []string{"main", "feature/a", "feature/b"} {
		if firstState.Slots[branch] != secondState.Slots[branch] {
			t.Fatalf(
				"slot for %q changed from %d to %d",
				branch,
				firstState.Slots[branch],
				secondState.Slots[branch],
			)
		}
	}

	configuration := config.Config{
		PortStride: 100,
		Ports: map[string]int{
			"frontend": 3000,
			"backend":  8080,
		},
	}
	usedPorts := make(map[int]string)
	for branch, slot := range secondState.Slots {
		ports, err := CalculatePorts(configuration, slot)
		if err != nil {
			t.Fatalf("CalculatePorts() for %q error = %v", branch, err)
		}
		for label, port := range ports {
			if existing, used := usedPorts[port]; used {
				t.Fatalf("port %d for %s/%s conflicts with %s", port, branch, label, existing)
			}
			usedPorts[port] = branch + "/" + label
		}
	}
	if len(usedPorts) != 6 {
		t.Fatalf("calculated %d unique ports, want 6", len(usedPorts))
	}
}

func initializeGitRepository(t *testing.T) string {
	t.Helper()

	repositoryRoot := filepath.Join(t.TempDir(), "repository")
	command := exec.Command("git", "init", "-b", "main", repositoryRoot)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("initialize Git repository: %v: %s", err, output)
	}

	readmePath := filepath.Join(repositoryRoot, "README.md")
	if err := os.WriteFile(readmePath, []byte("integration test\n"), 0o644); err != nil {
		t.Fatalf("write repository fixture: %v", err)
	}
	runGitCommand(t, repositoryRoot, "add", "README.md")
	runGitCommand(
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

func runGitCommand(t *testing.T, directory string, args ...string) {
	t.Helper()

	commandArgs := append([]string{"-C", directory}, args...)
	command := exec.Command("git", commandArgs...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
}
