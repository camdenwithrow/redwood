package session

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/camdenwithrow/redwood/internal/allocation"
	"github.com/camdenwithrow/redwood/internal/config"
	"github.com/camdenwithrow/redwood/internal/repository"
	"github.com/camdenwithrow/redwood/internal/tmux"
)

type detachedStarter interface {
	StartDetached(name string, windows []tmux.Window) (bool, error)
}

type Started struct {
	Name           string
	AlreadyRunning bool
}

func Start(repo repository.Repository, configuration config.Config, branch string) (Started, error) {
	return start(repo, configuration, branch, tmux.NewClient())
}

func start(repo repository.Repository, configuration config.Config, branch string, client detachedStarter) (Started, error) {
	worktrees, err := repository.ListWorktrees(repo)
	if err != nil {
		return Started{}, err
	}
	state, err := allocation.NewStore(repo).Reconcile(worktrees)
	if err != nil {
		return Started{}, fmt.Errorf("reconcile worktree slots: %w", err)
	}
	for _, worktree := range worktrees {
		if worktree.Branch != branch {
			continue
		}

		name := Name(repo, worktree)
		slot, exists := state.Slots[branch]
		if !exists {
			return Started{}, fmt.Errorf("branch %q has no allocated slot", branch)
		}
		ports, err := allocation.CalculatePorts(configuration, slot)
		if err != nil {
			return Started{}, fmt.Errorf("calculate ports for branch %q: %w", branch, err)
		}
		labels := make([]string, 0, len(configuration.Commands))
		for label := range configuration.Commands {
			labels = append(labels, label)
		}
		sort.Strings(labels)
		windows := make([]tmux.Window, 0, max(1, len(labels)))
		if len(labels) == 0 {
			windows = append(windows, tmux.Window{Name: "shell", Directory: worktree.Path})
		}
		portEnvironment := make(map[string]string, len(ports))
		for label, port := range ports {
			portEnvironment[config.PortEnvironmentVariable(label)] = strconv.Itoa(port)
		}
		for _, label := range labels {
			command := configuration.Commands[label]
			environment := cloneEnvironment(portEnvironment)
			if port, exists := ports[label]; exists {
				environment["RW_PORT"] = strconv.Itoa(port)
			}
			for name, value := range command.Env {
				environment[name] = config.ExpandPortPlaceholders(value, ports)
			}
			windows = append(windows, tmux.Window{
				Name:        label,
				Arguments:   append([]string(nil), command.Run...),
				Shell:       command.Shell,
				Directory:   worktree.Path,
				Environment: environment,
			})
		}
		alreadyRunning, err := client.StartDetached(name, windows)
		if err != nil {
			return Started{}, fmt.Errorf("start tmux session for branch %q: %w", branch, err)
		}
		return Started{Name: name, AlreadyRunning: alreadyRunning}, nil
	}

	return Started{}, fmt.Errorf("branch %q has no worktree", branch)
}

func cloneEnvironment(environment map[string]string) map[string]string {
	cloned := make(map[string]string, len(environment)+1)
	for name, value := range environment {
		cloned[name] = value
	}
	return cloned
}

func worktreeForBranch(repo repository.Repository, branch string) (repository.Worktree, error) {
	worktrees, err := repository.ListWorktrees(repo)
	if err != nil {
		return repository.Worktree{}, err
	}
	for _, worktree := range worktrees {
		if worktree.Branch == branch {
			return worktree, nil
		}
	}
	return repository.Worktree{}, fmt.Errorf("branch %q has no worktree", branch)
}
