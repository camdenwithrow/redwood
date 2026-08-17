package session

import (
	"fmt"
	"sort"

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
		windows := make([]tmux.Window, 0, len(labels))
		for _, label := range labels {
			windows = append(windows, tmux.Window{
				Name:      label,
				Command:   configuration.Commands[label],
				Directory: worktree.Path,
				Port:      ports[label],
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
