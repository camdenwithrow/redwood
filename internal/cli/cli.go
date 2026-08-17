package cli

import "fmt"

type command func(args []string) error

var commands = map[string]command{
	"attach": notImplemented("attach"),
	"create": notImplemented("create"),
	"list":   notImplemented("list"),
	"start":  notImplemented("start"),
	"stop":   notImplemented("stop"),
}

// Run dispatches args to the requested Redwood command.
func Run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command provided")
	}

	run, ok := commands[args[0]]
	if !ok {
		return fmt.Errorf("unknown command %q", args[0])
	}

	return run(args[1:])
}

func notImplemented(name string) command {
	return func(_ []string) error {
		return fmt.Errorf("rw %s is not implemented yet", name)
	}
}
