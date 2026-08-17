package worktree

import (
	"fmt"

	"github.com/camdenwithrow/redwood/internal/allocation"
	"github.com/camdenwithrow/redwood/internal/config"
	"github.com/camdenwithrow/redwood/internal/repository"
	"github.com/camdenwithrow/redwood/internal/session"
	"github.com/camdenwithrow/redwood/internal/tmux"
)

type Info struct {
	Worktree repository.Worktree
	Slot     *int
	Ports    map[string]int
	Running  bool
}

type sessionChecker interface {
	HasSession(name string) (bool, error)
}

func List(repo repository.Repository, configuration config.Config) ([]Info, error) {
	worktrees, err := repository.ListWorktrees(repo)
	if err != nil {
		return nil, err
	}

	state, err := allocation.NewStore(repo).Reconcile(worktrees)
	if err != nil {
		return nil, fmt.Errorf("reconcile worktree slots: %w", err)
	}

	listed, err := combine(worktrees, state)
	if err != nil {
		return nil, err
	}
	listed, err = addPorts(configuration, listed)
	if err != nil {
		return nil, err
	}

	return addRunningState(repo, listed, tmux.NewClient())
}

func combine(worktrees []repository.Worktree, state allocation.State) ([]Info, error) {
	listed := make([]Info, 0, len(worktrees))
	for _, discovered := range worktrees {
		entry := Info{Worktree: discovered}
		if !discovered.Detached {
			slot, exists := state.Slots[discovered.Branch]
			if !exists {
				return nil, fmt.Errorf("branch %q has no allocated slot", discovered.Branch)
			}
			entry.Slot = &slot
		}
		listed = append(listed, entry)
	}

	return listed, nil
}

func addPorts(configuration config.Config, listed []Info) ([]Info, error) {
	for i := range listed {
		if listed[i].Slot == nil {
			continue
		}
		ports, err := allocation.CalculatePorts(configuration, *listed[i].Slot)
		if err != nil {
			return nil, fmt.Errorf("calculate ports for worktree %q: %w", listed[i].Worktree.Path, err)
		}
		listed[i].Ports = ports
	}

	return listed, nil
}

func addRunningState(repo repository.Repository, listed []Info, checker sessionChecker) ([]Info, error) {
	for i := range listed {
		name := session.Name(repo, listed[i].Worktree)
		running, err := checker.HasSession(name)
		if err != nil {
			return nil, fmt.Errorf("check tmux session for worktree %q: %w", listed[i].Worktree.Path, err)
		}
		listed[i].Running = running
	}

	return listed, nil
}
