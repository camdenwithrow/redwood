package repository

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	gitexec "github.com/camdenwithrow/redwood/internal/git"
)

const (
	mainBranch   = "main"
	masterBranch = "master"
)

type Repository struct {
	Name         string
	MainCheckout string
	GitDir       string
}

func Discover() (Repository, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return Repository{}, fmt.Errorf("get current working directory: %w", err)
	}

	return DiscoverFrom(workingDirectory)
}

func DiscoverFrom(start string) (Repository, error) {
	runner := gitexec.NewRunner(start)
	root, err := runner.Output("rev-parse", "--show-toplevel")
	if err != nil {
		return Repository{}, fmt.Errorf("locate repository root: %w", err)
	}

	gitDir, err := runner.Output("rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return Repository{}, fmt.Errorf("locate shared Git directory: %w", err)
	}

	root = filepath.Clean(root)
	gitDir = filepath.Clean(gitDir)
	expectedGitDir := filepath.Join(root, ".git")
	if gitDir != expectedGitDir {
		return Repository{}, fmt.Errorf(
			"current checkout %q is not the main checkout; run rw from %q",
			root,
			filepath.Dir(gitDir),
		)
	}

	return Repository{
		Name:         filepath.Base(root),
		MainCheckout: root,
		GitDir:       gitDir,
	}, nil
}

func ResolveBaseBranch(repo Repository, configured string) (string, error) {
	if configured != "" {
		exists, err := localBranchExists(repo.MainCheckout, configured)
		if err != nil {
			return "", err
		}
		if !exists {
			return "", fmt.Errorf("configured base branch %q does not exist locally", configured)
		}

		return configured, nil
	}

	hasMain, err := localBranchExists(repo.MainCheckout, mainBranch)
	if err != nil {
		return "", err
	}
	hasMaster, err := localBranchExists(repo.MainCheckout, masterBranch)
	if err != nil {
		return "", err
	}

	switch {
	case hasMain && hasMaster:
		return "", fmt.Errorf("base_branch is omitted but both %q and %q exist; set base_branch explicitly", mainBranch, masterBranch)
	case hasMain:
		return mainBranch, nil
	case hasMaster:
		return masterBranch, nil
	default:
		return "", fmt.Errorf("base_branch is omitted but neither %q nor %q exists; set base_branch explicitly", mainBranch, masterBranch)
	}
}

func localBranchExists(repositoryRoot, branch string) (bool, error) {
	ref := "refs/heads/" + branch
	err := gitexec.NewRunner(repositoryRoot).Run("show-ref", "--verify", "--quiet", ref)
	if err == nil {
		return true, nil
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
		return false, nil
	}

	return false, fmt.Errorf("check local branch %q: %w", branch, err)
}
