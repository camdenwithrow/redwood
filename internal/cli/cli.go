package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/camdenwithrow/redwood/internal/config"
	"github.com/camdenwithrow/redwood/internal/repository"
	"github.com/camdenwithrow/redwood/internal/session"
	worktreemanager "github.com/camdenwithrow/redwood/internal/worktree"
)

type command func(args []string, environment commandEnvironment) error

type repositoryFinder func() (repository.Repository, error)
type configLoader func(repositoryRoot string) (config.Config, error)
type baseBranchResolver func(repo repository.Repository, configured string) (string, error)
type worktreeCreator func(repo repository.Repository, configuration config.Config, branch string) (worktreemanager.Created, error)
type worktreeRemover func(repo repository.Repository, branch string) (repository.Worktree, error)
type sessionStarter func(repo repository.Repository, configuration config.Config, branch string) (session.Started, error)
type sessionPlanner func(repo repository.Repository, configuration config.Config, branch string) (session.Plan, error)
type sessionAttacher func(repo repository.Repository, branch string) error
type sessionStopper func(repo repository.Repository, branch string) (string, error)
type worktreeLister func(repo repository.Repository, configuration config.Config) ([]worktreemanager.Info, error)

type runtimeDependencies struct {
	findRepository    repositoryFinder
	loadConfig        configLoader
	resolveBaseBranch baseBranchResolver
	createWorktree    worktreeCreator
	removeWorktree    worktreeRemover
	startSession      sessionStarter
	planSession       sessionPlanner
	attachSession     sessionAttacher
	stopSession       sessionStopper
	listWorktrees     worktreeLister
}

type commandEnvironment struct {
	repository repository.Repository
	config     config.Config
	stdout     io.Writer
	create     worktreeCreator
	remove     worktreeRemover
	start      sessionStarter
	plan       sessionPlanner
	attach     sessionAttacher
	stop       sessionStopper
	list       worktreeLister
}

type commandSpec struct {
	name      string
	arguments string
	argCount  int
	run       command
}

const usage = `Usage:
  rw <command> [arguments]

Commands:
  create <branch>  Create a worktree and assign its slot
  remove <branch>  Remove a worktree while keeping its branch
  start <branch>   Start commands in a detached tmux session
  start --dry-run <branch>
                   Show the session plan without launching tmux
  env <branch>     Show calculated ports and injected variables
  config check     Validate and summarize redwood.toml
  attach <branch>  Attach to a worktree's tmux session
  stop <branch>    Stop a worktree's tmux session
  list             Show worktrees, ports, and running state

Run "rw help" to show this message.
`

var commandSpecs = []commandSpec{
	{name: "create", arguments: "<branch>", argCount: 1, run: createWorktree},
	{name: "remove", arguments: "<branch>", argCount: 1, run: removeWorktree},
	{name: "start", arguments: "[--dry-run] <branch>", argCount: -1, run: startSession},
	{name: "env", arguments: "<branch>", argCount: 1, run: inspectEnvironment},
	{name: "config", arguments: "check", argCount: -1, run: checkConfig},
	{name: "attach", arguments: "<branch>", argCount: 1, run: attachSession},
	{name: "stop", arguments: "<branch>", argCount: 1, run: stopSession},
	{name: "list", argCount: 0, run: listWorktrees},
}

func Run(args []string, stdout, stderr io.Writer) int {
	return run(args, stdout, stderr, runtimeDependencies{
		findRepository:    repository.Discover,
		loadConfig:        config.Load,
		resolveBaseBranch: repository.ResolveBaseBranch,
		createWorktree:    worktreemanager.Create,
		removeWorktree:    worktreemanager.Remove,
		startSession:      session.Start,
		planSession:       session.BuildPlan,
		attachSession:     session.Attach,
		stopSession:       session.Stop,
		listWorktrees:     worktreemanager.List,
	})
}

func run(
	args []string,
	stdout io.Writer,
	stderr io.Writer,
	deps runtimeDependencies,
) int {
	if len(args) == 0 {
		writeUsageError(stderr, "no command provided")
		return 2
	}

	if args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		if len(args) != 1 {
			writeUsageError(stderr, "%s does not accept arguments", args[0])
			return 2
		}

		fmt.Fprint(stdout, usage)
		return 0
	}

	spec, ok := findCommand(args[0])
	if !ok {
		writeUsageError(stderr, "unknown command %q", args[0])
		return 2
	}

	commandArgs := args[1:]
	if message := validateArguments(*spec, commandArgs); message != "" {
		fmt.Fprintf(stderr, "rw: %s\nUsage: rw %s", message, spec.name)
		if spec.arguments != "" {
			fmt.Fprintf(stderr, " %s", spec.arguments)
		}
		fmt.Fprintln(stderr)
		return 2
	}

	discoveredRepository, err := deps.findRepository()
	if err != nil {
		fmt.Fprintf(stderr, "rw: %v\n", err)
		return 1
	}

	loadedConfig, err := deps.loadConfig(discoveredRepository.MainCheckout)
	if err != nil {
		fmt.Fprintf(stderr, "rw: %v\n", err)
		return 1
	}

	baseBranch, err := deps.resolveBaseBranch(discoveredRepository, loadedConfig.BaseBranch)
	if err != nil {
		fmt.Fprintf(stderr, "rw: %v\n", err)
		return 1
	}
	loadedConfig.BaseBranch = baseBranch

	environment := commandEnvironment{
		repository: discoveredRepository,
		config:     loadedConfig,
		stdout:     stdout,
		create:     deps.createWorktree,
		remove:     deps.removeWorktree,
		start:      deps.startSession,
		plan:       deps.planSession,
		attach:     deps.attachSession,
		stop:       deps.stopSession,
		list:       deps.listWorktrees,
	}
	if err := spec.run(commandArgs, environment); err != nil {
		fmt.Fprintf(stderr, "rw: %v\n", err)
		return 1
	}

	return 0
}

func attachSession(args []string, environment commandEnvironment) error {
	return environment.attach(environment.repository, args[0])
}

func stopSession(args []string, environment commandEnvironment) error {
	name, err := environment.stop(environment.repository, args[0])
	if err != nil {
		return err
	}
	fmt.Fprintf(environment.stdout, "Stopped tmux session %s\n", name)
	return nil
}

func startSession(args []string, environment commandEnvironment) error {
	branch := args[0]
	if len(args) == 2 {
		branch = args[1]
		plan, err := environment.plan(environment.repository, environment.config, branch)
		if err != nil {
			return err
		}
		return writeSessionPlan(environment.stdout, plan, true)
	}
	started, err := environment.start(environment.repository, environment.config, branch)
	if err != nil {
		return err
	}
	if started.AlreadyRunning {
		fmt.Fprintf(environment.stdout, "Tmux session already running: %s\n", started.Name)
		return nil
	}
	fmt.Fprintf(environment.stdout, "Started tmux session %s\n", started.Name)
	return nil
}

func checkConfig(_ []string, environment commandEnvironment) error {
	fmt.Fprintln(environment.stdout, "Configuration valid")
	fmt.Fprintf(environment.stdout, "Path: %s\n", filepath.Join(environment.repository.MainCheckout, config.FileName))
	fmt.Fprintf(environment.stdout, "Base branch: %s\n", environment.config.BaseBranch)
	fmt.Fprintf(environment.stdout, "Commands: %d\n", len(environment.config.Commands))
	fmt.Fprintf(environment.stdout, "Ports: %d\n", len(environment.config.Ports))
	return nil
}

func inspectEnvironment(args []string, environment commandEnvironment) error {
	plan, err := environment.plan(environment.repository, environment.config, args[0])
	if err != nil {
		return err
	}
	return writeSessionPlan(environment.stdout, plan, false)
}

func writeSessionPlan(output io.Writer, plan session.Plan, includeTmux bool) error {
	fmt.Fprintf(output, "Branch: %s\nPath: %s\nSlot: %d\nSession: %s\n", plan.Branch, plan.Path, plan.Slot, plan.Name)
	fmt.Fprintln(output, "Ports:")
	portNames := sortedStringKeys(plan.Ports)
	if len(portNames) == 0 {
		fmt.Fprintln(output, "  (none)")
	}
	for _, name := range portNames {
		fmt.Fprintf(output, "  %s=%d\n", name, plan.Ports[name])
	}
	fmt.Fprintln(output, "Windows:")
	for index, window := range plan.Windows {
		fmt.Fprintf(output, "  %s:\n", window.Name)
		fmt.Fprintln(output, "    Environment:")
		environmentNames := sortedStringKeys(window.Environment)
		if len(environmentNames) == 0 {
			fmt.Fprintln(output, "      (none)")
		}
		for _, name := range environmentNames {
			fmt.Fprintf(output, "      %s=%s\n", name, window.Environment[name])
		}
		if !includeTmux {
			continue
		}
		expanded := expandInjectedVariables(window.Command, window.Environment)
		if expanded == "" {
			expanded = "(interactive shell)"
		}
		fmt.Fprintf(output, "    Expanded command: %s\n", expanded)
		arguments, err := json.Marshal(plan.TmuxArgs[index])
		if err != nil {
			return fmt.Errorf("encode tmux arguments: %w", err)
		}
		fmt.Fprintf(output, "    Tmux arguments: %s\n", arguments)
	}
	return nil
}

func sortedStringKeys[Value any](values map[string]Value) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func expandInjectedVariables(command string, environment map[string]string) string {
	var result strings.Builder
	for index := 0; index < len(command); {
		if command[index] != '$' || index+1 >= len(command) {
			result.WriteByte(command[index])
			index++
			continue
		}
		start := index
		index++
		if command[index] == '$' {
			result.WriteString("$$")
			index++
			continue
		}
		braced := command[index] == '{'
		if braced {
			index++
		}
		nameStart := index
		for index < len(command) && (command[index] == '_' || command[index] >= 'A' && command[index] <= 'Z' || command[index] >= 'a' && command[index] <= 'z' || index > nameStart && command[index] >= '0' && command[index] <= '9') {
			index++
		}
		if index == nameStart || braced && (index >= len(command) || command[index] != '}') {
			result.WriteString(command[start:index])
			continue
		}
		name := command[nameStart:index]
		if braced {
			index++
		}
		if value, exists := environment[name]; exists {
			result.WriteString(value)
		} else {
			result.WriteString(command[start:index])
		}
	}
	return result.String()
}

func createWorktree(args []string, environment commandEnvironment) error {
	created, err := environment.create(environment.repository, environment.config, args[0])
	if err != nil {
		return err
	}

	fmt.Fprintf(environment.stdout, "Created worktree %s\n", created.Worktree.Branch)
	fmt.Fprintf(environment.stdout, "Path: %s\n", created.Worktree.Path)
	fmt.Fprintf(environment.stdout, "Slot: %d\n", created.Slot)
	if len(created.Ports) == 0 {
		return nil
	}
	fmt.Fprintln(environment.stdout, "Ports:")
	labels := make([]string, 0, len(created.Ports))
	for label := range created.Ports {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	for _, label := range labels {
		fmt.Fprintf(environment.stdout, "  %s: %d\n", label, created.Ports[label])
	}

	return nil
}

func removeWorktree(args []string, environment commandEnvironment) error {
	removed, err := environment.remove(environment.repository, args[0])
	if err != nil {
		return err
	}
	fmt.Fprintf(environment.stdout, "Removed worktree %s\n", removed.Branch)
	fmt.Fprintf(environment.stdout, "Path: %s\n", removed.Path)
	return nil
}

func listWorktrees(_ []string, environment commandEnvironment) error {
	listed, err := environment.list(environment.repository, environment.config)
	if err != nil {
		return err
	}
	return writeWorktreeList(environment.stdout, listed)
}

func writeWorktreeList(output io.Writer, listed []worktreemanager.Info) error {
	sorted := append([]worktreemanager.Info(nil), listed...)
	sort.SliceStable(sorted, func(i, j int) bool {
		left := sorted[i]
		right := sorted[j]
		if left.Slot == nil || right.Slot == nil {
			if left.Slot != nil {
				return true
			}
			if right.Slot != nil {
				return false
			}
			return left.Worktree.Path < right.Worktree.Path
		}
		return *left.Slot < *right.Slot
	})

	writer := csv.NewWriter(output)
	writer.Comma = '\t'
	if err := writer.Write([]string{"BRANCH", "SLOT", "RW_SESSION", "PORTS", "PATH"}); err != nil {
		return fmt.Errorf("write worktree list: %w", err)
	}
	for _, info := range sorted {
		if err := writer.Write(worktreeFields(info)); err != nil {
			return fmt.Errorf("write worktree list: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("write worktree list: %w", err)
	}

	return nil
}

func worktreeFields(info worktreemanager.Info) []string {
	branch := info.Worktree.Branch
	if branch == "" {
		branch = "(detached)"
	}
	slot := "-"
	if info.Slot != nil {
		slot = strconv.Itoa(*info.Slot)
	}
	running := "none"
	if info.Running {
		running = "running"
	}
	labels := make([]string, 0, len(info.Ports))
	for label := range info.Ports {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	ports := make([]string, 0, len(labels))
	for _, label := range labels {
		ports = append(ports, label+"="+strconv.Itoa(info.Ports[label]))
	}
	portList := strings.Join(ports, ",")
	if portList == "" {
		portList = "-"
	}

	return []string{branch, slot, running, portList, info.Worktree.Path}
}

func findCommand(name string) (*commandSpec, bool) {
	for i := range commandSpecs {
		if commandSpecs[i].name == name {
			return &commandSpecs[i], true
		}
	}

	return nil, false
}

func argumentError(spec commandSpec, actual int) string {
	if spec.argCount == 0 {
		return fmt.Sprintf("%s does not accept arguments", spec.name)
	}
	if actual == 0 {
		return fmt.Sprintf("%s requires %s", spec.name, spec.arguments)
	}

	return fmt.Sprintf("%s accepts exactly one %s argument", spec.name, spec.arguments)
}

func validateArguments(spec commandSpec, args []string) string {
	switch spec.name {
	case "start":
		if len(args) == 1 {
			return ""
		}
		if len(args) == 2 && args[0] == "--dry-run" {
			return ""
		}
		if len(args) == 2 {
			return "start accepts exactly one <branch> argument"
		}
		return "start requires <branch> or --dry-run <branch>"
	case "config":
		if len(args) == 1 && args[0] == "check" {
			return ""
		}
		return "config requires the check subcommand"
	default:
		if len(args) == spec.argCount {
			return ""
		}
		return argumentError(spec, len(args))
	}
}

func writeUsageError(stderr io.Writer, format string, args ...any) {
	fmt.Fprintf(stderr, "rw: "+format+"\n\n", args...)
	fmt.Fprint(stderr, usage)
}
