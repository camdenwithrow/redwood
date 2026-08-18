package worktree

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/camdenwithrow/redwood/internal/allocation"
	"github.com/camdenwithrow/redwood/internal/config"
	"github.com/camdenwithrow/redwood/internal/repository"
)

type Created struct {
	Worktree repository.Worktree
	Slot     int
	Ports    map[string]int
}

func Create(repo repository.Repository, configuration config.Config, branch string) (Created, error) {
	store := allocation.NewStore(repo)
	previousState, err := store.Load()
	if err != nil {
		return Created{}, err
	}

	path, err := repository.ResolveWorktreePath(repo, configuration.WorktreePath, branch)
	if err != nil {
		return Created{}, err
	}

	createdWorktree, err := repository.CreateWorktree(repo, branch, path, configuration.BaseBranch)
	if err != nil {
		return Created{}, err
	}
	rollback := func(cause error) error {
		worktreeErr := repository.RollbackWorktree(
			repo,
			createdWorktree.Worktree.Path,
			branch,
			createdWorktree.BranchCreated,
		)
		allocationErr := store.Save(previousState)
		return errors.Join(cause, worktreeErr, allocationErr)
	}

	worktrees, err := repository.ListWorktrees(repo)
	if err != nil {
		return Created{}, rollback(err)
	}
	state, err := store.Reconcile(worktrees)
	if err != nil {
		return Created{}, rollback(fmt.Errorf("allocate worktree slot: %w", err))
	}
	slot, exists := state.Slots[branch]
	if !exists {
		return Created{}, rollback(fmt.Errorf("allocation state has no slot for branch %q", branch))
	}
	ports, err := allocation.CalculatePorts(configuration, slot)
	if err != nil {
		return Created{}, rollback(fmt.Errorf("calculate worktree ports: %w", err))
	}
	for index, hook := range configuration.Hooks.PostCreate {
		if err := runPostCreateHook(createdWorktree.Worktree.Path, hook); err != nil {
			return Created{}, rollback(fmt.Errorf("run post-create hook %d %q: %w", index+1, hook, err))
		}
	}

	return Created{Worktree: createdWorktree.Worktree, Slot: slot, Ports: ports}, nil
}

func runPostCreateHook(directory, hook string) error {
	command := exec.Command("/bin/sh", "-c", hook)
	command.Dir = directory
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}
