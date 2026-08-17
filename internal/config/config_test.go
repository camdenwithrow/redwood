package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validConfig = `
worktree_path = "../{repo}-{branch}"
port_stride = 100

[ports]
web = 3000
api = 8080

[commands]
web = "just dev-web"
api = "just dev-server"
`

func TestLoadValidConfig(t *testing.T) {
	repositoryRoot := writeConfig(t, validConfig)

	loaded, err := Load(repositoryRoot)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.BaseBranch != "" {
		t.Fatalf("Load() BaseBranch = %q, want empty value for auto-detection", loaded.BaseBranch)
	}
	if loaded.WorktreePath != "../{repo}-{branch}" {
		t.Fatalf("Load() WorktreePath = %q", loaded.WorktreePath)
	}
	if loaded.PortStride != 100 {
		t.Fatalf("Load() PortStride = %d, want 100", loaded.PortStride)
	}
	if loaded.Ports["web"] != 3000 {
		t.Fatalf("Load() web port = %d, want 3000", loaded.Ports["web"])
	}
	if loaded.Commands["api"] != "just dev-server" {
		t.Fatalf("Load() api command = %q", loaded.Commands["api"])
	}
}

func TestLoadRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "missing file", want: "no such file"},
		{name: "unknown field", content: strings.Replace(validConfig, "worktree_path", "unknown = true\nworktree_path", 1), want: "unknown field(s): unknown"},
		{name: "empty base branch", content: strings.Replace(validConfig, "worktree_path", "base_branch = \"\"\nworktree_path", 1), want: "base_branch must not be empty"},
		{name: "missing worktree path", content: strings.Replace(validConfig, "worktree_path = \"../{repo}-{branch}\"\n", "", 1), want: "worktree_path is required"},
		{name: "missing branch placeholder", content: strings.Replace(validConfig, "../{repo}-{branch}", "../{repo}", 1), want: "worktree_path must contain {branch}"},
		{name: "invalid stride", content: strings.Replace(validConfig, "port_stride = 100", "port_stride = 0", 1), want: "port_stride must be greater than zero"},
		{name: "missing ports", content: strings.Replace(validConfig, "web = 3000\napi = 8080", "", 1), want: "ports must contain at least one entry"},
		{name: "invalid port", content: strings.Replace(validConfig, "web = 3000", "web = 70000", 1), want: "ports.web must be between 1 and 65535"},
		{name: "missing commands", content: strings.Replace(validConfig, "web = \"just dev-web\"\napi = \"just dev-server\"", "", 1), want: "commands must contain at least one entry"},
		{name: "empty command", content: strings.Replace(validConfig, "web = \"just dev-web\"", "web = \" \"", 1), want: "commands.web must not be empty"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repositoryRoot := t.TempDir()
			if test.content != "" {
				writeConfigAt(t, repositoryRoot, test.content)
			}

			_, err := Load(repositoryRoot)
			if err == nil {
				t.Fatal("Load() error = nil, want validation error")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Load() error = %q, want it to contain %q", err, test.want)
			}
		})
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	repositoryRoot := t.TempDir()
	writeConfigAt(t, repositoryRoot, content)
	return repositoryRoot
}

func writeConfigAt(t *testing.T, repositoryRoot, content string) {
	t.Helper()

	configPath := filepath.Join(repositoryRoot, FileName)
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}
}
