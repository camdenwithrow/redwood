package session

import (
	"maps"
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
	name           string
	windows        []tmux.Window
	alreadyRunning bool
}

func (starter *fakeStarter) StartDetached(name string, windows []tmux.Window) (bool, error) {
	starter.name = name
	starter.windows = windows
	return starter.alreadyRunning, nil
}

func TestStartCreatesDetachedSessionForBranch(t *testing.T) {
	repo := initializeRepository(t)
	client := &fakeStarter{}
	configuration := config.Config{Commands: map[string]config.Command{
		"frontend": {Run: []string{"just", "dev-web"}},
		"backend":  {Shell: "just dev-server"},
	}, Ports: map[string]int{
		"frontend": 3000,
		"backend":  8080,
	}, PortStride: 100}

	started, err := start(repo, configuration, "main", client)
	if err != nil {
		t.Fatalf("start() error = %v", err)
	}
	if started.Name == "" || client.name != started.Name {
		t.Fatalf("start() name = %q, client name = %q", started.Name, client.name)
	}
	if len(client.windows) != 2 || client.windows[0].Name != "backend" || client.windows[1].Name != "frontend" {
		t.Fatalf("start() windows = %v, want sorted command windows", client.windows)
	}
	for _, window := range client.windows {
		if window.Directory != repo.MainCheckout {
			t.Fatalf("start() window directory = %q, want %q", window.Directory, repo.MainCheckout)
		}
	}
	backendEnvironment := map[string]string{
		"RW_PORT":          "8080",
		"RW_PORT_BACKEND":  "8080",
		"RW_PORT_FRONTEND": "3000",
	}
	frontendEnvironment := map[string]string{
		"RW_PORT":          "3000",
		"RW_PORT_BACKEND":  "8080",
		"RW_PORT_FRONTEND": "3000",
	}
	if !maps.Equal(client.windows[0].Environment, backendEnvironment) || !maps.Equal(client.windows[1].Environment, frontendEnvironment) {
		t.Fatalf("start() window environments = %v, want all calculated ports", client.windows)
	}
}

func TestStartCreatesShellWindowWithoutCommands(t *testing.T) {
	repo := initializeRepository(t)
	client := &fakeStarter{}

	_, err := start(repo, config.Config{}, "main", client)
	if err != nil {
		t.Fatalf("start() error = %v", err)
	}
	if len(client.windows) != 1 {
		t.Fatalf("start() windows = %v, want one shell window", client.windows)
	}
	window := client.windows[0]
	if window.Name != "shell" || len(window.Arguments) != 0 || window.Shell != "" || window.Directory != repo.MainCheckout || len(window.Environment) != 0 {
		t.Fatalf("start() shell window = %+v, want default shell in %q", window, repo.MainCheckout)
	}
}

func TestStartCreatesCommandWindowWithoutPort(t *testing.T) {
	repo := initializeRepository(t)
	client := &fakeStarter{}
	configuration := config.Config{Commands: map[string]config.Command{"tests": {Shell: "go test ./..."}}}

	_, err := start(repo, configuration, "main", client)
	if err != nil {
		t.Fatalf("start() error = %v", err)
	}
	if len(client.windows) != 1 || client.windows[0].Shell != "go test ./..." || len(client.windows[0].Environment) != 0 {
		t.Fatalf("start() windows = %v, want command without port", client.windows)
	}
}

func TestStartReportsExistingSession(t *testing.T) {
	repo := initializeRepository(t)
	client := &fakeStarter{alreadyRunning: true}
	configuration := config.Config{
		Commands:   map[string]config.Command{"web": {Shell: "just dev-web"}},
		Ports:      map[string]int{"web": 3000},
		PortStride: 100,
	}

	started, err := start(repo, configuration, "main", client)
	if err != nil {
		t.Fatalf("start() error = %v", err)
	}
	if !started.AlreadyRunning {
		t.Fatal("start() AlreadyRunning = false, want true")
	}
}

func TestStartExpandsStructuredEnvironmentAndPreservesArguments(t *testing.T) {
	repo := initializeRepository(t)
	client := &fakeStarter{}
	configuration := config.Config{
		Commands: map[string]config.Command{
			"web": {
				Run: []string{"just", "dev-web", "--host", "127.0.0.1"},
				Env: map[string]string{
					"PORT":    "{ports.web}",
					"API_URL": "http://localhost:{ports.api}",
				},
			},
			"api": {Shell: "just dev-server | tee server.log"},
		},
		Ports:      map[string]int{"web": 3000, "api": 8080},
		PortStride: 100,
	}

	_, err := start(repo, configuration, "main", client)
	if err != nil {
		t.Fatalf("start() error = %v", err)
	}
	web := client.windows[1]
	if !maps.Equal(web.Environment, map[string]string{
		"RW_PORT": "3000", "RW_PORT_API": "8080", "RW_PORT_WEB": "3000",
		"PORT": "3000", "API_URL": "http://localhost:8080",
	}) {
		t.Fatalf("start() web environment = %v", web.Environment)
	}
	if len(web.Arguments) != 4 || web.Arguments[0] != "just" || web.Arguments[3] != "127.0.0.1" || web.Shell != "" {
		t.Fatalf("start() web command = arguments %v, shell %q", web.Arguments, web.Shell)
	}
	if client.windows[0].Shell != "just dev-server | tee server.log" {
		t.Fatalf("start() api shell = %q", client.windows[0].Shell)
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
