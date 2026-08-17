package repository

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	root, err := gitOutput(start, "rev-parse", "--show-toplevel")
	if err != nil {
		return Repository{}, fmt.Errorf("locate repository root: %w", err)
	}

	gitDir, err := gitOutput(start, "rev-parse", "--path-format=absolute", "--git-common-dir")
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

func gitOutput(directory string, args ...string) (string, error) {
	commandArgs := append([]string{"-C", directory}, args...)
	output, err := exec.Command("git", commandArgs...).CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if detail == "" {
			detail = err.Error()
		}

		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), detail)
	}

	return strings.TrimSpace(string(output)), nil
}
