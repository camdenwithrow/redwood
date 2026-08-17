package cli

import (
	"fmt"
	"io"
)

type command func(args []string) error

type commandSpec struct {
	name      string
	arguments string
	argCount  int
	run       command
}

const usage = `Usage:
  rw <command> [arguments]

Commands:
  create <branch>  Create a worktree and assign its ports
  start <branch>   Start commands in a detached tmux session
  attach <branch>  Attach to a worktree's tmux session
  stop <branch>    Stop a worktree's tmux session
  list             Show worktrees, ports, and running state

Run "rw help" to show this message.
`

var commandSpecs = []commandSpec{
	{name: "create", arguments: "<branch>", argCount: 1, run: notImplemented("create")},
	{name: "start", arguments: "<branch>", argCount: 1, run: notImplemented("start")},
	{name: "attach", arguments: "<branch>", argCount: 1, run: notImplemented("attach")},
	{name: "stop", arguments: "<branch>", argCount: 1, run: notImplemented("stop")},
	{name: "list", argCount: 0, run: notImplemented("list")},
}

// Run dispatches args to the requested Redwood command and returns a process
// exit code.
func Run(args []string, stdout, stderr io.Writer) int {
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
	if len(commandArgs) != spec.argCount {
		fmt.Fprintf(stderr, "rw: %s\nUsage: rw %s", argumentError(*spec, len(commandArgs)), spec.name)
		if spec.arguments != "" {
			fmt.Fprintf(stderr, " %s", spec.arguments)
		}
		fmt.Fprintln(stderr)
		return 2
	}

	if err := spec.run(commandArgs); err != nil {
		fmt.Fprintf(stderr, "rw: %v\n", err)
		return 1
	}

	return 0
}

func notImplemented(name string) command {
	return func(_ []string) error {
		return fmt.Errorf("%s is not implemented yet", name)
	}
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

func writeUsageError(stderr io.Writer, format string, args ...any) {
	fmt.Fprintf(stderr, "rw: "+format+"\n\n", args...)
	fmt.Fprint(stderr, usage)
}
