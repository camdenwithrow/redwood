package config

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

const FileName = "redwood.toml"

// Config contains the repository-specific settings Redwood needs for its MVP.
type Config struct {
	BaseBranch   string            `toml:"base_branch"`
	WorktreePath string            `toml:"worktree_path"`
	PortStride   int               `toml:"port_stride"`
	Ports        map[string]int    `toml:"ports"`
	Commands     map[string]string `toml:"commands"`
}

// Load reads and validates redwood.toml from repositoryRoot.
func Load(repositoryRoot string) (Config, error) {
	configPath := filepath.Join(repositoryRoot, FileName)
	loaded := Config{}

	metadata, err := toml.DecodeFile(configPath, &loaded)
	if err != nil {
		return Config{}, fmt.Errorf("load %s: %w", configPath, err)
	}

	if undecoded := metadata.Undecoded(); len(undecoded) > 0 {
		keys := make([]string, 0, len(undecoded))
		for _, key := range undecoded {
			keys = append(keys, key.String())
		}
		sort.Strings(keys)

		return Config{}, fmt.Errorf("load %s: unknown field(s): %s", configPath, strings.Join(keys, ", "))
	}
	if metadata.IsDefined("base_branch") && strings.TrimSpace(loaded.BaseBranch) == "" {
		return Config{}, fmt.Errorf("load %s: base_branch must not be empty", configPath)
	}

	if err := loaded.validate(); err != nil {
		return Config{}, fmt.Errorf("load %s: %w", configPath, err)
	}

	return loaded, nil
}

func (config Config) validate() error {
	if strings.TrimSpace(config.WorktreePath) == "" {
		return fmt.Errorf("worktree_path is required")
	}
	if !strings.Contains(config.WorktreePath, "{branch}") {
		return fmt.Errorf("worktree_path must contain {branch}")
	}
	if config.PortStride <= 0 {
		return fmt.Errorf("port_stride must be greater than zero")
	}
	if len(config.Ports) == 0 {
		return fmt.Errorf("ports must contain at least one entry")
	}
	if len(config.Commands) == 0 {
		return fmt.Errorf("commands must contain at least one entry")
	}

	for name, port := range config.Ports {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("ports contains an empty name")
		}
		if port < 1 || port > 65535 {
			return fmt.Errorf("ports.%s must be between 1 and 65535", name)
		}
	}

	for name, command := range config.Commands {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("commands contains an empty name")
		}
		if strings.TrimSpace(command) == "" {
			return fmt.Errorf("commands.%s must not be empty", name)
		}
	}

	return nil
}
