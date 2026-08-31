package worktree

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/camdenwithrow/redwood/internal/allocation"
	"github.com/camdenwithrow/redwood/internal/config"
	"github.com/camdenwithrow/redwood/internal/repository"
)

func TestCreateCopiesOnlyRequestedEnvFiles(t *testing.T) {
	repo, configuration := envTestRepository(t)
	paths := []string{".env", "apps/api/.env.local"}
	content := "# preserved verbatim\r\nTOKEN=fake-test-value\r\nPORT=3000\r\n"
	for _, path := range paths {
		writeEnvFixture(t, filepath.Join(repo.MainCheckout, path), content)
	}
	writeEnvFixture(t, filepath.Join(repo.MainCheckout, ".env.local"), "DO_NOT_COPY=true\n")

	created, err := Create(repo, configuration, "feature/env", paths...)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		target := filepath.Join(created.Worktree.Path, path)
		data, err := os.ReadFile(target)
		if err != nil || string(data) != content {
			t.Fatalf("copied file %q differs from source: %v", path, err)
		}
		info, err := os.Stat(target)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("copied file permissions = %o, want 600", info.Mode().Perm())
		}
		writeEnvFixture(t, target, "CHANGED=true\n")
		original, err := os.ReadFile(filepath.Join(repo.MainCheckout, path))
		if err != nil || string(original) != content {
			t.Fatalf("editing copy changed source %q: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(created.Worktree.Path, ".env.local")); !os.IsNotExist(err) {
		t.Fatalf("unrequested env file exists: %v", err)
	}
	for _, path := range []string{"apps", "apps/api"} {
		info, err := os.Stat(filepath.Join(created.Worktree.Path, path))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o700 {
			t.Fatalf("new directory permissions = %o, want 700", info.Mode().Perm())
		}
	}

	withoutCopy, err := Create(repo, configuration, "feature/no-env")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(withoutCopy.Worktree.Path, ".env")); !os.IsNotExist(err) {
		t.Fatalf("env copied without opting in: %v", err)
	}
}

func TestCreateCopiesEnvBeforePostCreateHooks(t *testing.T) {
	repo, configuration := envTestRepository(t)
	configuration.Hooks.PostCreate = []string{"cp .env hook-copy.txt"}
	created, err := Create(repo, configuration, "feature/env-hook", ".env")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(created.Worktree.Path, "hook-copy.txt"))
	if err != nil || string(data) != "TOKEN=fake\n" {
		t.Fatalf("hook could not use copied env file: %v", err)
	}
}

func TestCreateHookFailureRollsBackCopiedEnv(t *testing.T) {
	repo, configuration := envTestRepository(t)
	configuration.Hooks.PostCreate = []string{"test -s .env && exit 17"}
	_, err := Create(repo, configuration, "feature/env-hook-failure", ".env")
	if err == nil || !strings.Contains(err.Error(), "exit status 17") {
		t.Fatalf("Create() error = %v, want hook failure after copying env", err)
	}
	path, err := repository.ResolveWorktreePath(repo, configuration.WorktreePath, "feature/env-hook-failure")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("worktree and copied env remain after hook failure: %v", err)
	}
	if branchExists(t, repo.MainCheckout, "feature/env-hook-failure") {
		t.Fatal("new branch remains after hook failure")
	}
	state, err := allocation.NewStore(repo).Load()
	if err != nil || len(state.Slots) != 0 {
		t.Fatalf("allocation was not restored: %+v, %v", state, err)
	}
	data, err := os.ReadFile(filepath.Join(repo.MainCheckout, ".env"))
	if err != nil || string(data) != "TOKEN=fake\n" {
		t.Fatalf("source was changed after hook failure: %v", err)
	}
}

func TestCreateEnvFailureRollsBack(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		setup func(*testing.T, string)
		want  string
	}{
		{name: "missing source after first copy", paths: []string{".env", ".env.local"}, want: "inspect source"},
		{name: "absolute path", paths: []string{"/tmp/.env"}, want: "repository-relative"},
		{name: "escape", paths: []string{"../.env"}, want: "repository-relative"},
		{name: "internal traversal", paths: []string{"apps/../.env"}, want: "traversal"},
		{name: "git metadata", paths: []string{".git/config"}, want: ".git paths"},
		{name: "nested git metadata", paths: []string{"apps/.GIT/config"}, want: ".git paths"},
		{name: "empty path", paths: []string{""}, want: "repository-relative"},
		{name: "current directory", paths: []string{"."}, want: "repository-relative"},
		{name: "duplicate path", paths: []string{".env", "./.env"}, want: "more than once"},
		{name: "unignored destination", paths: []string{"local.env"}, setup: func(t *testing.T, root string) {
			writeEnvFixture(t, filepath.Join(root, "local.env"), "TOKEN=fake\n")
		}, want: "not Git-ignored"},
		{name: "git pathspec syntax is a literal filename", paths: []string{":(top).env"}, setup: func(t *testing.T, root string) {
			writeEnvFixture(t, filepath.Join(root, ":(top).env"), "TOKEN=fake\n")
		}, want: "not Git-ignored"},
		{name: "source directory", paths: []string{"directory/.env"}, setup: func(t *testing.T, root string) {
			if err := os.MkdirAll(filepath.Join(root, "directory/.env"), 0o700); err != nil {
				t.Fatal(err)
			}
		}, want: "regular file"},
		{name: "source symlink", paths: []string{".env.local"}, setup: func(t *testing.T, root string) {
			if err := os.Symlink(".env", filepath.Join(root, ".env.local")); err != nil {
				t.Fatal(err)
			}
		}, want: "regular file"},
		{name: "source parent symlink", paths: []string{"linked/.env"}, setup: func(t *testing.T, root string) {
			if err := os.Symlink(root, filepath.Join(root, "linked")); err != nil {
				t.Fatal(err)
			}
		}, want: "parent"},
		{name: "existing tracked destination", paths: []string{".env"}, setup: func(t *testing.T, root string) {
			runGit(t, root, "add", "--force", ".env")
			commitEnvFixture(t, root)
		}, want: "destination already exists"},
		{name: "destination file symlink", paths: []string{".env.local"}, setup: func(t *testing.T, root string) {
			if err := os.Symlink("missing-target", filepath.Join(root, ".env.local")); err != nil {
				t.Fatal(err)
			}
			runGit(t, root, "add", "--force", ".env.local")
			commitEnvFixture(t, root)
			if err := os.Remove(filepath.Join(root, ".env.local")); err != nil {
				t.Fatal(err)
			}
			writeEnvFixture(t, filepath.Join(root, ".env.local"), "TOKEN=fake\n")
		}, want: "destination already exists"},
		{name: "destination parent symlink", paths: []string{"linked/.env"}, setup: func(t *testing.T, root string) {
			outside := t.TempDir()
			if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
				t.Fatal(err)
			}
			runGit(t, root, "add", "linked")
			commitEnvFixture(t, root)
			if err := os.Remove(filepath.Join(root, "linked")); err != nil {
				t.Fatal(err)
			}
			writeEnvFixture(t, filepath.Join(root, "linked/.env"), "TOKEN=fake\n")
			t.Cleanup(func() {
				if _, err := os.Stat(filepath.Join(outside, ".env")); !os.IsNotExist(err) {
					t.Errorf("copy wrote through destination symlink: %v", err)
				}
			})
		}, want: "parent"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, configuration := envTestRepository(t)
			if test.setup != nil {
				test.setup(t, repo.MainCheckout)
			}
			store := allocation.NewStore(repo)
			previous := allocation.State{Version: 1, Slots: map[string]int{"main": 0}}
			if err := store.Save(previous); err != nil {
				t.Fatal(err)
			}
			_, err := Create(repo, configuration, "feature/failure", test.paths...)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Create() error = %v, want %q", err, test.want)
			}
			path, err := repository.ResolveWorktreePath(repo, configuration.WorktreePath, "feature/failure")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("failed worktree still exists: %v", err)
			}
			if branchExists(t, repo.MainCheckout, "feature/failure") {
				t.Fatal("new branch still exists after copy failure")
			}
			state, err := store.Load()
			if err != nil || !reflect.DeepEqual(state, previous) {
				t.Fatalf("allocation state = %+v, %v; want %+v", state, err, previous)
			}
			data, err := os.ReadFile(filepath.Join(repo.MainCheckout, ".env"))
			if err != nil || string(data) != "TOKEN=fake\n" {
				t.Fatalf("source was changed after copy failure: %v", err)
			}
		})
	}
}

func TestCreateChecksDestinationBranchIgnoreRulesAndKeepsExistingBranch(t *testing.T) {
	repo, configuration := envTestRepository(t)
	runGit(t, repo.MainCheckout, "checkout", "-b", "feature/different-ignore")
	writeEnvFixture(t, filepath.Join(repo.MainCheckout, ".gitignore"), "# no env rules\n")
	runGit(t, repo.MainCheckout, "add", ".gitignore")
	commitEnvFixture(t, repo.MainCheckout)
	runGit(t, repo.MainCheckout, "checkout", "main")

	_, err := Create(repo, configuration, "feature/different-ignore", ".env")
	if err == nil || !strings.Contains(err.Error(), "not Git-ignored") {
		t.Fatalf("Create() error = %v, want destination ignore rejection", err)
	}
	if !branchExists(t, repo.MainCheckout, "feature/different-ignore") {
		t.Fatal("copy failure deleted a pre-existing branch")
	}
}

func envTestRepository(t *testing.T) (repository.Repository, config.Config) {
	t.Helper()
	root := initializeRepository(t)
	writeEnvFixture(t, filepath.Join(root, ".gitignore"), ".env\n.env.local\n")
	runGit(t, root, "add", ".gitignore")
	commitEnvFixture(t, root)
	writeEnvFixture(t, filepath.Join(root, ".env"), "TOKEN=fake\n")
	repo, err := repository.DiscoverFrom(root)
	if err != nil {
		t.Fatal(err)
	}
	return repo, config.Config{
		BaseBranch: "main", WorktreePath: filepath.Join(t.TempDir(), "{repo}-{branch}"),
	}
}

func writeEnvFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func commitEnvFixture(t *testing.T, root string) {
	t.Helper()
	runGit(t, root, "-c", "user.name=Redwood Tests", "-c", "user.email=redwood@example.com", "commit", "-m", "Env fixture")
}
