package worktree

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	gitexec "github.com/camdenwithrow/redwood/internal/git"
)

func copyEnvFiles(sourcePath, destinationPath string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(paths))
	for _, path := range paths {
		if err := validateEnvPath(path); err != nil {
			return err
		}
		clean := filepath.Clean(path)
		if seen[clean] {
			return fmt.Errorf("copy env %q: path was specified more than once", path)
		}
		seen[clean] = true
	}

	// Root keeps file access inside each checkout, including if paths change
	// between inspection and opening. Component checks also reject symlinks.
	source, err := os.OpenRoot(sourcePath)
	if err != nil {
		return fmt.Errorf("open env source checkout: %w", err)
	}
	defer source.Close()
	destination, err := os.OpenRoot(destinationPath)
	if err != nil {
		return fmt.Errorf("open env destination worktree: %w", err)
	}
	defer destination.Close()

	for _, path := range paths {
		if err := copyEnvFile(source, destination, filepath.Clean(path)); err != nil {
			return fmt.Errorf("copy env %q: %w", path, err)
		}
	}
	return nil
}

func validateEnvPath(path string) error {
	if !filepath.IsLocal(path) || strings.Contains(path, "\\") || filepath.Clean(path) == "." {
		return fmt.Errorf("copy env %q: expected a repository-relative file path", path)
	}
	for _, component := range strings.Split(filepath.ToSlash(path), "/") {
		if component == ".." || strings.EqualFold(component, ".git") {
			return fmt.Errorf("copy env %q: traversal and .git paths are not allowed", path)
		}
	}
	return nil
}

func copyEnvFile(source, destination *os.Root, path string) error {
	if err := envParentDirectories(source, path, false); err != nil {
		return fmt.Errorf("inspect source: %w", err)
	}
	info, err := source.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect source: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("source must be a regular file, not a symlink or directory")
	}
	input, err := source.Open(path)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer input.Close()
	openedInfo, err := input.Stat()
	if err != nil {
		return fmt.Errorf("inspect opened source: %w", err)
	}
	if !os.SameFile(info, openedInfo) || !openedInfo.Mode().IsRegular() {
		return fmt.Errorf("source changed while opening it")
	}

	if err := envParentDirectories(destination, path, true); err != nil {
		return fmt.Errorf("prepare destination: %w", err)
	}
	if _, err := destination.Lstat(path); err == nil {
		return fmt.Errorf("destination already exists; refusing to overwrite it")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect destination: %w", err)
	}
	// A ./ prefix prevents Git from interpreting a leading colon as pathspec
	// syntax and checking a different path from the file we will create.
	if err := gitexec.NewRunner(destination.Name()).Run("check-ignore", "--quiet", "--", "./"+filepath.ToSlash(path)); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
			return fmt.Errorf("destination is not Git-ignored; add an ignore rule before copying")
		}
		return fmt.Errorf("check destination ignore rules: %w", err)
	}

	output, err := destination.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		return fmt.Errorf("write destination: %w", err)
	}
	return nil
}

func envParentDirectories(root *os.Root, path string, create bool) error {
	parent := filepath.Dir(path)
	if parent == "." {
		return nil
	}
	current := ""
	for _, component := range strings.Split(parent, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := root.Lstat(current)
		if os.IsNotExist(err) && create {
			if err := root.Mkdir(current, 0o700); err != nil {
				return err
			}
			info, err = root.Lstat(current)
		}
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("parent %q must be a directory, not a symlink or file", current)
		}
	}
	return nil
}
