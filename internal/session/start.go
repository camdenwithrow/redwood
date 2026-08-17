package session

import (
	"fmt"
	"sort"

	"github.com/camdenwithrow/redwood/internal/config"
	"github.com/camdenwithrow/redwood/internal/repository"
	"github.com/camdenwithrow/redwood/internal/tmux"
)

type detachedStarter interface {
	StartDetached(name string, windows []tmux.Window) error
}

func Start(repo repository.Repository, configuration config.Config, branch string) (string, error) {
	return start(repo, configuration, branch, tmux.NewClient())
}

func start(repo repository.Repository, configuration config.Config, branch string, client detachedStarter) (string, error) {
	worktrees, err := repository.ListWorktrees(repo)
	if err != nil {
		return "", err
	}
	for _, worktree := range worktrees {
		if worktree.Branch != branch {
			continue
		}

		name := Name(repo, worktree)
		labels := make([]string, 0, len(configuration.Commands))
		for label := range configuration.Commands {
			labels = append(labels, label)
		}
		sort.Strings(labels)
		windows := make([]tmux.Window, 0, len(labels))
		for _, label := range labels {
			windows = append(windows, tmux.Window{Name: label, Command: configuration.Commands[label]})
		}
		if err := client.StartDetached(name, windows); err != nil {
			return "", fmt.Errorf("start tmux session for branch %q: %w", branch, err)
		}
		return name, nil
	}

	return "", fmt.Errorf("branch %q has no worktree", branch)
}
