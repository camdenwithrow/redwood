package redwood_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type testProject struct {
	repository  string
	binary      string
	environment []string
}

func TestCreateCommandInTemporaryRepository(t *testing.T) {
	project := newTestProject(t)

	output := project.run(t, "create", "feature/integration")

	if !strings.Contains(output, "Created worktree feature/integration") {
		t.Fatalf("rw create output = %q, want created branch", output)
	}
	wantPath := filepath.Join(filepath.Dir(project.repository), "project-feature-integration")
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("created worktree: %v", err)
	}
	worktrees := gitOutput(t, project.repository, "worktree", "list", "--porcelain")
	if !strings.Contains(worktrees, "worktree "+wantPath) || !strings.Contains(worktrees, "branch refs/heads/feature/integration") {
		t.Fatalf("git worktree list = %q, want created worktree", worktrees)
	}
	allocationPath := filepath.Join(project.repository, ".git", "redwood", "allocations.toml")
	if _, err := os.Stat(allocationPath); err != nil {
		t.Fatalf("allocation file: %v", err)
	}
}

func TestEveryCommandRunsFromMainCheckout(t *testing.T) {
	project := newTestProject(t)

	project.run(t, "create", "feature/all-commands")
	startOutput := project.run(t, "start", "feature/all-commands")
	listOutput := project.run(t, "list")
	project.run(t, "attach", "feature/all-commands")
	stopOutput := project.run(t, "stop", "feature/all-commands")

	if !strings.Contains(startOutput, "Started tmux session") {
		t.Fatalf("rw start output = %q, want started session", startOutput)
	}
	if !strings.Contains(listOutput, "feature/all-commands\t1\trunning\tapp=4100") {
		t.Fatalf("rw list output = %q, want running feature worktree", listOutput)
	}
	if !strings.Contains(stopOutput, "Stopped tmux session") {
		t.Fatalf("rw stop output = %q, want stopped session", stopOutput)
	}
}

func newTestProject(t *testing.T) testProject {
	t.Helper()

	projectRoot := filepath.Join(t.TempDir(), "project")
	runCommand(t, "", "git", "init", "-b", "main", projectRoot)
	writeFile(t, filepath.Join(projectRoot, "README.md"), "# Test project\n")
	writeFile(t, filepath.Join(projectRoot, "redwood.toml"), `base_branch = "main"
worktree_path = "../{repo}-{branch}"
port_stride = 100

[ports]
app = 4000

[commands]
app = "sleep 60"
`)
	runCommand(t, projectRoot, "git", "add", ".")
	runCommand(
		t,
		projectRoot,
		"git", "-c", "user.name=Redwood Tests", "-c", "user.email=redwood@example.com",
		"commit", "-m", "Initial commit",
	)

	moduleRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("get module root: %v", err)
	}
	binary := filepath.Join(t.TempDir(), "rw")
	runCommand(t, moduleRoot, "go", "build", "-o", binary, "./cmd/rw")
	tmuxDirectory := t.TempDir()
	tmuxBinary := filepath.Join(tmuxDirectory, "tmux")
	writeExecutable(t, tmuxBinary, `#!/bin/sh
command=$1
shift
name=
target=
while [ "$#" -gt 1 ]; do
	case "$1" in
		-s) name=$2 ;;
		-t) target=$2 ;;
	esac
	shift
done
if [ -z "$name" ]; then
	name=$target
fi
name=${name#=}
name=${name%:}
session="$REDWOOD_TEST_TMUX_STATE/$name"
case "$command" in
	has-session) test -f "$session" ;;
	new-session) touch "$session" ;;
	new-window) test -f "$session" ;;
	attach-session) test -f "$session" ;;
	kill-session) rm -f "$session" ;;
	*) exit 2 ;;
esac
`)
	stateDirectory := t.TempDir()
	environment := []string{
		"PATH=" + tmuxDirectory + string(os.PathListSeparator) + os.Getenv("PATH"),
		"REDWOOD_TEST_TMUX_STATE=" + stateDirectory,
	}

	canonicalRoot, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	return testProject{repository: canonicalRoot, binary: binary, environment: environment}
}

func (project testProject) run(t *testing.T, args ...string) string {
	t.Helper()
	command := exec.Command(project.binary, args...)
	command.Dir = project.repository
	command.Env = append(os.Environ(), project.environment...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("rw %s: %v: %s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func gitOutput(t *testing.T, directory string, args ...string) string {
	t.Helper()
	return runCommand(t, directory, "git", args...)
}

func runCommand(t *testing.T, directory, name string, args ...string) string {
	t.Helper()
	command := exec.Command(name, args...)
	if directory != "" {
		command.Dir = directory
	}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v: %s", name, strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeExecutable(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
