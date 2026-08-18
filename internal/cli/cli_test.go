package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/camdenwithrow/redwood/internal/config"
	"github.com/camdenwithrow/redwood/internal/repository"
	"github.com/camdenwithrow/redwood/internal/session"
	"github.com/camdenwithrow/redwood/internal/tmux"
	worktreemanager "github.com/camdenwithrow/redwood/internal/worktree"
)

func TestRunHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"help"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("Run() exit code = %d, want 0", exitCode)
	}
	if stdout.String() != usage {
		t.Fatalf("Run() stdout = %q, want usage text", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("Run() stderr = %q, want empty", stderr.String())
	}
}

func TestRunUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing command", want: "rw: no command provided"},
		{name: "unknown command", args: []string{"launch"}, want: `rw: unknown command "launch"`},
		{name: "missing branch", args: []string{"create"}, want: "rw: create requires <branch>"},
		{name: "extra branch", args: []string{"start", "one", "two"}, want: "rw: start accepts exactly one <branch> argument"},
		{name: "missing remove branch", args: []string{"remove"}, want: "rw: remove requires <branch>"},
		{name: "list argument", args: []string{"list", "feature/a"}, want: "rw: list does not accept arguments"},
		{name: "missing env branch", args: []string{"env"}, want: "rw: env requires <branch>"},
		{name: "invalid config subcommand", args: []string{"config", "show"}, want: "rw: config requires the check subcommand"},
		{name: "missing dry-run branch", args: []string{"start", "--dry-run", "feature/a", "extra"}, want: "rw: start requires <branch> or --dry-run <branch>"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			exitCode := Run(test.args, &stdout, &stderr)

			if exitCode != 2 {
				t.Fatalf("Run() exit code = %d, want 2", exitCode)
			}
			if !strings.Contains(stderr.String(), test.want) {
				t.Fatalf("Run() stderr = %q, want it to contain %q", stderr.String(), test.want)
			}
			if stdout.Len() != 0 {
				t.Fatalf("Run() stdout = %q, want empty", stdout.String())
			}
		})
	}
}

func TestRunAttachDispatchesSelectedBranch(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var attachedBranch string
	deps := successfulDependencies()
	deps.attachSession = func(_ repository.Repository, branch string) error {
		attachedBranch = branch
		return nil
	}

	exitCode := run(
		[]string{"attach", "feature/a"},
		&stdout,
		&stderr,
		deps,
	)

	if exitCode != 0 {
		t.Fatalf("Run() exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	if attachedBranch != "feature/a" {
		t.Fatalf("run() attached branch = %q, want feature/a", attachedBranch)
	}
}

func TestRunCreatePrintsWorktreeDetails(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	deps := successfulDependencies()
	deps.createWorktree = func(repository.Repository, config.Config, string) (worktreemanager.Created, error) {
		return worktreemanager.Created{
			Worktree: repository.Worktree{Path: "/repo-feature-a", Branch: "feature/a"},
			Slot:     2,
			Ports:    map[string]int{"frontend": 3200, "backend": 8280},
		}, nil
	}

	exitCode := run([]string{"create", "feature/a"}, &stdout, &stderr, deps)

	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	want := "Created worktree feature/a\nPath: /repo-feature-a\nSlot: 2\nPorts:\n  backend: 8280\n  frontend: 3200\n"
	if stdout.String() != want {
		t.Fatalf("run() stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunCreateOmitsPortsWhenNoneAreConfigured(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	deps := successfulDependencies()
	deps.createWorktree = func(repository.Repository, config.Config, string) (worktreemanager.Created, error) {
		return worktreemanager.Created{
			Worktree: repository.Worktree{Path: "/repo-feature-a", Branch: "feature/a"},
			Slot:     2,
		}, nil
	}

	exitCode := run([]string{"create", "feature/a"}, &stdout, &stderr, deps)

	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	want := "Created worktree feature/a\nPath: /repo-feature-a\nSlot: 2\n"
	if stdout.String() != want {
		t.Fatalf("run() stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunRemovePrintsWorktreeDetails(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	deps := successfulDependencies()
	deps.removeWorktree = func(repository.Repository, string) (repository.Worktree, error) {
		return repository.Worktree{Path: "/repo-feature-a", Branch: "feature/a"}, nil
	}

	exitCode := run([]string{"remove", "feature/a"}, &stdout, &stderr, deps)

	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	want := "Removed worktree feature/a\nPath: /repo-feature-a\n"
	if stdout.String() != want {
		t.Fatalf("run() stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunStartPrintsSessionName(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{"start", "feature/a"}, &stdout, &stderr, successfulDependencies())

	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	if got, want := stdout.String(), "Started tmux session rw-redwood-main-123456789abc\n"; got != want {
		t.Fatalf("run() stdout = %q, want %q", got, want)
	}
}

func TestRunStartReportsExistingSession(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	deps := successfulDependencies()
	deps.startSession = func(repository.Repository, config.Config, string) (session.Started, error) {
		return session.Started{Name: "existing-session", AlreadyRunning: true}, nil
	}

	exitCode := run([]string{"start", "feature/a"}, &stdout, &stderr, deps)

	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	if got, want := stdout.String(), "Tmux session already running: existing-session\n"; got != want {
		t.Fatalf("run() stdout = %q, want %q", got, want)
	}
}

func TestRunConfigCheckSummarizesValidatedConfiguration(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	deps := successfulDependencies()
	deps.loadConfig = func(string) (config.Config, error) {
		return config.Config{
			BaseBranch: "main",
			Commands:   map[string]string{"api": "just api", "web": "just web"},
			Ports:      map[string]int{"api": 8080},
		}, nil
	}

	exitCode := run([]string{"config", "check"}, &stdout, &stderr, deps)

	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	want := "Configuration valid\nPath: /repo/redwood.toml\nBase branch: main\nCommands: 2\nPorts: 1\n"
	if stdout.String() != want {
		t.Fatalf("run() stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunEnvShowsCalculatedPortsAndInjectedVariables(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	deps := successfulDependencies()
	started := false
	deps.startSession = func(repository.Repository, config.Config, string) (session.Started, error) {
		started = true
		return session.Started{}, nil
	}
	deps.planSession = func(repository.Repository, config.Config, string) (session.Plan, error) {
		return inspectionPlan(), nil
	}

	exitCode := run([]string{"env", "feature/a"}, &stdout, &stderr, deps)

	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	if started {
		t.Fatal("run() started a session during environment inspection")
	}
	want := "Branch: feature/a\nPath: /repo-feature-a\nSlot: 2\nSession: rw-repo-feature-a-abc\n" +
		"Ports:\n  api=8280\n  web=3200\nWindows:\n  api:\n    Environment:\n" +
		"      RW_PORT=8280\n      RW_PORT_API=8280\n      RW_PORT_WEB=3200\n"
	if stdout.String() != want {
		t.Fatalf("run() stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunStartDryRunShowsExpandedCommandAndTmuxArguments(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	deps := successfulDependencies()
	started := false
	deps.startSession = func(repository.Repository, config.Config, string) (session.Started, error) {
		started = true
		return session.Started{}, nil
	}
	deps.planSession = func(repository.Repository, config.Config, string) (session.Plan, error) {
		return inspectionPlan(), nil
	}

	exitCode := run([]string{"start", "--dry-run", "feature/a"}, &stdout, &stderr, deps)

	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	if started {
		t.Fatal("run() started a session during dry-run")
	}
	for _, want := range []string{
		"Expanded command: just api --port 8280 --web 3200 --token $TOKEN\n",
		`Tmux arguments: ["new-session","-d","-s","rw-repo-feature-a-abc","-n","api","-c","/repo-feature-a","-e","RW_PORT=8280","-e","RW_PORT_API=8280","-e","RW_PORT_WEB=3200","just api --port $RW_PORT --web ${RW_PORT_WEB} --token $TOKEN"]`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("run() stdout = %q, want it to contain %q", stdout.String(), want)
		}
	}
}

func TestRunStopPrintsSessionName(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{"stop", "feature/a"}, &stdout, &stderr, successfulDependencies())

	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	if got, want := stdout.String(), "Stopped tmux session rw-redwood-main-123456789abc\n"; got != want {
		t.Fatalf("run() stdout = %q, want %q", got, want)
	}
}

func TestRunListPrintsWorktreeDetails(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	mainSlot := 0
	featureSlot := 2
	deps := successfulDependencies()
	deps.listWorktrees = func(repository.Repository, config.Config) ([]worktreemanager.Info, error) {
		return []worktreemanager.Info{
			{
				Worktree: repository.Worktree{Branch: "feature/a", Path: "/repo feature a"},
				Slot:     &featureSlot,
				Ports:    map[string]int{"web": 3200, "api": 8280},
				Running:  true,
			},
			{
				Worktree: repository.Worktree{Branch: "main", Path: "/repo"},
				Slot:     &mainSlot,
				Ports:    map[string]int{"web": 3000, "api": 8080},
			},
			{Worktree: repository.Worktree{Path: "/repo-detached", Detached: true}},
		}, nil
	}

	exitCode := run([]string{"list"}, &stdout, &stderr, deps)

	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	want := "BRANCH\tSLOT\tRW_SESSION\tPORTS\tPATH\n" +
		"main\t0\tnone\tapi=8080,web=3000\t/repo\n" +
		"feature/a\t2\trunning\tapi=8280,web=3200\t/repo feature a\n" +
		"(detached)\t-\tnone\t-\t/repo-detached\n"
	if stdout.String() != want {
		t.Fatalf("run() stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunReportsRepositoryDiscoveryError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	finder := func() (repository.Repository, error) {
		return repository.Repository{}, errors.New("run rw from the main checkout")
	}

	deps := successfulDependencies()
	deps.findRepository = finder
	exitCode := run([]string{"list"}, &stdout, &stderr, deps)

	if exitCode != 1 {
		t.Fatalf("run() exit code = %d, want 1", exitCode)
	}
	if got := stderr.String(); got != "rw: run rw from the main checkout\n" {
		t.Fatalf("run() stderr = %q, want repository error", got)
	}
}

func TestRunReportsConfigError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	loader := func(string) (config.Config, error) {
		return config.Config{}, errors.New("load redwood.toml: port_stride must be greater than zero")
	}

	deps := successfulDependencies()
	deps.loadConfig = loader
	exitCode := run([]string{"list"}, &stdout, &stderr, deps)

	if exitCode != 1 {
		t.Fatalf("run() exit code = %d, want 1", exitCode)
	}
	if got := stderr.String(); got != "rw: load redwood.toml: port_stride must be greater than zero\n" {
		t.Fatalf("run() stderr = %q, want configuration error", got)
	}
}

func TestRunReportsBaseBranchResolutionError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	deps := successfulDependencies()
	deps.resolveBaseBranch = func(repository.Repository, string) (string, error) {
		return "", errors.New(`both "main" and "master" exist; set base_branch explicitly`)
	}

	exitCode := run([]string{"list"}, &stdout, &stderr, deps)

	if exitCode != 1 {
		t.Fatalf("run() exit code = %d, want 1", exitCode)
	}
	if got := stderr.String(); got != "rw: both \"main\" and \"master\" exist; set base_branch explicitly\n" {
		t.Fatalf("run() stderr = %q, want base branch error", got)
	}
}

func successfulRepositoryFinder() (repository.Repository, error) {
	return repository.Repository{
		Name:         "repo",
		MainCheckout: "/repo",
		GitDir:       "/repo/.git",
	}, nil
}

func successfulConfigLoader(string) (config.Config, error) {
	return config.Config{}, nil
}

func successfulDependencies() runtimeDependencies {
	return runtimeDependencies{
		findRepository: successfulRepositoryFinder,
		loadConfig:     successfulConfigLoader,
		resolveBaseBranch: func(repository.Repository, string) (string, error) {
			return "main", nil
		},
		createWorktree: func(repository.Repository, config.Config, string) (worktreemanager.Created, error) {
			return worktreemanager.Created{}, nil
		},
		removeWorktree: func(repository.Repository, string) (repository.Worktree, error) {
			return repository.Worktree{}, nil
		},
		startSession: func(repository.Repository, config.Config, string) (session.Started, error) {
			return session.Started{Name: "rw-redwood-main-123456789abc"}, nil
		},
		planSession: func(repository.Repository, config.Config, string) (session.Plan, error) {
			return inspectionPlan(), nil
		},
		attachSession: func(repository.Repository, string) error { return nil },
		stopSession: func(repository.Repository, string) (string, error) {
			return "rw-redwood-main-123456789abc", nil
		},
		listWorktrees: func(repository.Repository, config.Config) ([]worktreemanager.Info, error) {
			return nil, nil
		},
	}
}

func inspectionPlan() session.Plan {
	return session.Plan{
		Name:   "rw-repo-feature-a-abc",
		Branch: "feature/a",
		Path:   "/repo-feature-a",
		Slot:   2,
		Ports:  map[string]int{"web": 3200, "api": 8280},
		Windows: []tmux.Window{{
			Name:      "api",
			Command:   "just api --port $RW_PORT --web ${RW_PORT_WEB} --token $TOKEN",
			Directory: "/repo-feature-a",
			Environment: map[string]string{
				"RW_PORT": "8280", "RW_PORT_API": "8280", "RW_PORT_WEB": "3200",
			},
		}},
		TmuxArgs: [][]string{{
			"new-session", "-d", "-s", "rw-repo-feature-a-abc", "-n", "api", "-c", "/repo-feature-a",
			"-e", "RW_PORT=8280", "-e", "RW_PORT_API=8280", "-e", "RW_PORT_WEB=3200",
			"just api --port $RW_PORT --web ${RW_PORT_WEB} --token $TOKEN",
		}},
	}
}
