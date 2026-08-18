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

[hooks]
post_create = ["go mod download", "cp .env.example .env"]
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
	if got := loaded.Hooks.PostCreate; len(got) != 2 || got[0] != "go mod download" || got[1] != "cp .env.example .env" {
		t.Fatalf("Load() post-create hooks = %v", got)
	}
}

func TestLoadMinimalConfigWithoutPortsOrCommands(t *testing.T) {
	repositoryRoot := writeConfig(t, `worktree_path = "../{repo}-{branch}"`)

	loaded, err := Load(repositoryRoot)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.PortStride != 0 || len(loaded.Ports) != 0 || len(loaded.Commands) != 0 {
		t.Fatalf("Load() port settings = stride %d, ports %v, commands %v; want empty", loaded.PortStride, loaded.Ports, loaded.Commands)
	}
}

func TestLoadAllowsCommandWithoutPort(t *testing.T) {
	repositoryRoot := writeConfig(t, `
worktree_path = "../{repo}-{branch}"

[commands]
tests = "go test ./..."
`)

	loaded, err := Load(repositoryRoot)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Commands["tests"] != "go test ./..." {
		t.Fatalf("Load() tests command = %q", loaded.Commands["tests"])
	}
}

func TestPortEnvironmentVariable(t *testing.T) {
	for label, want := range map[string]string{
		"frontend": "RW_PORT_FRONTEND",
		"user-api": "RW_PORT_USER_API",
		"Docs UI":  "RW_PORT_DOCS_UI",
	} {
		if got := PortEnvironmentVariable(label); got != want {
			t.Errorf("PortEnvironmentVariable(%q) = %q, want %q", label, got, want)
		}
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
		{name: "invalid port", content: strings.Replace(validConfig, "web = 3000", "web = 70000", 1), want: "ports.web must be between 1 and 65535"},
		{name: "empty command", content: strings.Replace(validConfig, "web = \"just dev-web\"", "web = \" \"", 1), want: "commands.web must not be empty"},
		{name: "empty post-create hook", content: strings.Replace(validConfig, "go mod download", " ", 1), want: "hooks.post_create[0] must not be empty"},
		{name: "port label without environment name", content: strings.Replace(validConfig, "web = 3000", "\"---\" = 3000", 1), want: "must contain at least one ASCII letter or digit"},
		{name: "colliding port environment variables", content: strings.Replace(validConfig, "web = 3000", "web = 3000\n\"WEB\" = 3101", 1), want: "map to the same environment variable RW_PORT_WEB"},
		{name: "port without command", content: strings.Replace(validConfig, "api = \"just dev-server\"\n", "", 1), want: "ports.api requires a matching commands.api entry"},
		{name: "ports can collide between slots", content: strings.Replace(validConfig, "api = 8080", "api = 3100", 1), want: "ports.api and ports.web can collide across worktree slots"},
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
