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

// Repository describes the main checkout and its shared Git directory.
type Repository struct {
	Root   string
	GitDir string
}

// Discover locates and validates the main checkout containing the current
// working directory.
func Discover() (Repository, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return Repository{}, fmt.Errorf("get current working directory: %w", err)
	}

	return DiscoverFrom(workingDirectory)
}

// DiscoverFrom locates a repository from start and verifies that start belongs
// to its main checkout rather than a linked worktree.
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

	return Repository{Root: root, GitDir: gitDir}, nil
}

// ResolveBaseBranch verifies a configured local base branch or auto-detects
// main or master when configured is empty.
func ResolveBaseBranch(repo Repository, configured string) (string, error) {
	if configured != "" {
		exists, err := localBranchExists(repo.Root, configured)
		if err != nil {
			return "", err
		}
		if !exists {
			return "", fmt.Errorf("configured base branch %q does not exist locally", configured)
		}

		return configured, nil
	}

	hasMain, err := localBranchExists(repo.Root, mainBranch)
	if err != nil {
		return "", err
	}
	hasMaster, err := localBranchExists(repo.Root, masterBranch)
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
