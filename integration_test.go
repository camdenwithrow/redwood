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

func TestPortlessProjectCreatesInteractiveSession(t *testing.T) {
	project := newTestProject(t)
	writeFile(t, filepath.Join(project.repository, "redwood.toml"), `base_branch = "main"
worktree_path = "../{repo}-{branch}"
`)

	createOutput := project.run(t, "create", "feature/portless")
	startOutput := project.run(t, "start", "feature/portless")
	listOutput := project.run(t, "list")

	if strings.Contains(createOutput, "Ports:") {
		t.Fatalf("rw create output = %q, want no ports section", createOutput)
	}
	if !strings.Contains(startOutput, "Started tmux session") {
		t.Fatalf("rw start output = %q, want started session", startOutput)
	}
	if !strings.Contains(listOutput, "feature/portless\t1\trunning\t-") {
		t.Fatalf("rw list output = %q, want running portless worktree", listOutput)
	}

	project.run(t, "stop", "feature/portless")
}

func TestRemoveCommandDeletesWorktreeButKeepsBranch(t *testing.T) {
	project := newTestProject(t)
	project.run(t, "create", "feature/remove")
	worktreePath := filepath.Join(filepath.Dir(project.repository), "project-feature-remove")

	output := project.run(t, "remove", "feature/remove")

	if !strings.Contains(output, "Removed worktree feature/remove") {
		t.Fatalf("rw remove output = %q, want removed branch", output)
	}
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Fatalf("removed worktree path error = %v, want not found", err)
	}
	if output := gitOutput(t, project.repository, "branch", "--list", "feature/remove"); output != "feature/remove" {
		t.Fatalf("git branch --list output = %q, want retained branch", output)
	}
	if listOutput := project.run(t, "list"); strings.Contains(listOutput, "feature/remove") {
		t.Fatalf("rw list output = %q, want removed worktree absent", listOutput)
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

func TestEveryCommandRunsFromLinkedWorktree(t *testing.T) {
	project := newTestProject(t)
	project.run(t, "create", "feature/source")
	sourcePath := filepath.Join(filepath.Dir(project.repository), "project-feature-source")
	targetPath := filepath.Join(filepath.Dir(project.repository), "project-feature-target")

	createOutput := project.runFrom(t, sourcePath, "create", "feature/target")
	startOutput := project.runFrom(t, sourcePath, "start", "feature/target")
	listOutput := project.runFrom(t, sourcePath, "list")
	project.runFrom(t, sourcePath, "attach", "feature/target")
	stopOutput := project.runFrom(t, sourcePath, "stop", "feature/target")
	removeOutput := project.runFrom(t, sourcePath, "remove", "feature/target")

	if !strings.Contains(createOutput, "Created worktree feature/target") {
		t.Fatalf("rw create output = %q, want created target", createOutput)
	}
	if !strings.Contains(startOutput, "Started tmux session") {
		t.Fatalf("rw start output = %q, want started session", startOutput)
	}
	if !strings.Contains(listOutput, "feature/target\t2\trunning\tapp=4200") {
		t.Fatalf("rw list output = %q, want running target", listOutput)
	}
	if !strings.Contains(stopOutput, "Stopped tmux session") {
		t.Fatalf("rw stop output = %q, want stopped session", stopOutput)
	}
	if !strings.Contains(removeOutput, "Removed worktree feature/target") {
		t.Fatalf("rw remove output = %q, want removed target", removeOutput)
	}
	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Fatalf("removed target path error = %v, want not found", err)
	}
}

func TestTwoWorktreesUseDifferentPortsAndRunConcurrently(t *testing.T) {
	project := newTestProject(t)

	project.run(t, "create", "feature/a")
	project.run(t, "create", "feature/b")
	project.run(t, "start", "feature/a")
	project.run(t, "start", "feature/b")
	listOutput := project.run(t, "list")

	if !strings.Contains(listOutput, "feature/a\t1\trunning\tapp=4100") {
		t.Fatalf("rw list output = %q, want feature/a running on port 4100", listOutput)
	}
	if !strings.Contains(listOutput, "feature/b\t2\trunning\tapp=4200") {
		t.Fatalf("rw list output = %q, want feature/b running on port 4200", listOutput)
	}

	project.run(t, "stop", "feature/a")
	project.run(t, "stop", "feature/b")
}

func TestProcessesRemainRunningAfterDetaching(t *testing.T) {
	project, tmuxPath, socket := newTestProject(t).withRealTmux(t)
	t.Cleanup(func() {
		command := exec.Command(tmuxPath, "-S", socket, "kill-server")
		_ = command.Run()
	})

	project.run(t, "create", "feature/detached")
	project.run(t, "start", "feature/detached")
	listOutput := project.run(t, "list")
	panes := runCommand(t, "", tmuxPath, "-S", socket, "list-panes", "-a", "-F", "#{pane_dead}|#{pane_start_command}")

	if !strings.Contains(listOutput, "feature/detached\t1\trunning\tapp=4100") {
		t.Fatalf("rw list output = %q, want detached session running", listOutput)
	}
	if !strings.HasPrefix(panes, "0|") || !strings.Contains(panes, "sleep 60") {
		t.Fatalf("tmux panes = %q, want live sleep process", panes)
	}

	project.run(t, "stop", "feature/detached")
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
		"TMUX=",
	}

	canonicalRoot, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	return testProject{repository: canonicalRoot, binary: binary, environment: environment}
}

func (project testProject) run(t *testing.T, args ...string) string {
	t.Helper()
	return project.runFrom(t, project.repository, args...)
}

func (project testProject) runFrom(t *testing.T, directory string, args ...string) string {
	t.Helper()
	command := exec.Command(project.binary, args...)
	command.Dir = directory
	command.Env = append(os.Environ(), project.environment...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("rw %s: %v: %s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func (project testProject) withRealTmux(t *testing.T) (testProject, string, string) {
	t.Helper()

	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		t.Skip("tmux is required for detached process integration coverage")
	}
	socketFile, err := os.CreateTemp("/tmp", "redwood-tmux-")
	if err != nil {
		t.Fatalf("create tmux socket path: %v", err)
	}
	socket := socketFile.Name()
	if err := socketFile.Close(); err != nil {
		t.Fatalf("close tmux socket placeholder: %v", err)
	}
	if err := os.Remove(socket); err != nil {
		t.Fatalf("prepare tmux socket path: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(socket) })
	wrapperDirectory := t.TempDir()
	wrapper := filepath.Join(wrapperDirectory, "tmux")
	writeExecutable(
		t,
		wrapper,
		"#!/bin/sh\nexec "+shellQuote(tmuxPath)+" -S "+shellQuote(socket)+" \"$@\"\n",
	)
	project.environment = []string{
		"PATH=" + wrapperDirectory + string(os.PathListSeparator) + os.Getenv("PATH"),
	}
	return project, tmuxPath, socket
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

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
