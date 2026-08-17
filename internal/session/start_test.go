package session

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camdenwithrow/redwood/internal/config"
	"github.com/camdenwithrow/redwood/internal/repository"
	"github.com/camdenwithrow/redwood/internal/tmux"
)

type fakeStarter struct {
	name    string
	windows []tmux.Window
}

func (starter *fakeStarter) StartDetached(name string, windows []tmux.Window) error {
	starter.name = name
	starter.windows = windows
	return nil
}

func TestStartCreatesDetachedSessionForBranch(t *testing.T) {
	repo := initializeRepository(t)
	client := &fakeStarter{}
	configuration := config.Config{Commands: map[string]string{
		"frontend": "just dev-web",
		"backend":  "just dev-server",
	}}

	name, err := start(repo, configuration, "main", client)
	if err != nil {
		t.Fatalf("start() error = %v", err)
	}
	if name == "" || client.name != name {
		t.Fatalf("start() name = %q, client name = %q", name, client.name)
	}
	if len(client.windows) != 2 || client.windows[0].Name != "backend" || client.windows[1].Name != "frontend" {
		t.Fatalf("start() windows = %v, want sorted command windows", client.windows)
	}
}

func TestStartRejectsMissingWorktree(t *testing.T) {
	repo := initializeRepository(t)

	_, err := start(repo, config.Config{}, "feature/missing", &fakeStarter{})
	if err == nil {
		t.Fatal("start() error = nil, want missing worktree error")
	}
	if !strings.Contains(err.Error(), `branch "feature/missing" has no worktree`) {
		t.Fatalf("start() error = %q, want branch context", err)
	}
}

func initializeRepository(t *testing.T) repository.Repository {
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
	repo, err := repository.DiscoverFrom(repositoryRoot)
	if err != nil {
		t.Fatalf("DiscoverFrom() error = %v", err)
	}
	return repo
}

func runGit(t *testing.T, directory string, args ...string) {
	t.Helper()

	commandArgs := append([]string{"-C", directory}, args...)
	if output, err := exec.Command("git", commandArgs...).CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
}
