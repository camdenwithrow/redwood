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

	portNames := sortedKeys(config.Ports)
	for _, name := range portNames {
		port := config.Ports[name]
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("ports contains an empty name")
		}
		if port < 1 || port > 65535 {
			return fmt.Errorf("ports.%s must be between 1 and 65535", name)
		}
	}

	commandNames := sortedKeys(config.Commands)
	for _, name := range commandNames {
		command := config.Commands[name]
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("commands contains an empty name")
		}
		if strings.TrimSpace(command) == "" {
			return fmt.Errorf("commands.%s must not be empty", name)
		}
		if _, ok := config.Ports[name]; !ok {
			return fmt.Errorf("commands.%s requires a matching ports.%s entry", name, name)
		}
	}

	for _, name := range portNames {
		if _, ok := config.Commands[name]; !ok {
			return fmt.Errorf("ports.%s requires a matching commands.%s entry", name, name)
		}
	}

	for i, firstName := range portNames {
		for _, secondName := range portNames[i+1:] {
			if config.Ports[firstName]%config.PortStride == config.Ports[secondName]%config.PortStride {
				return fmt.Errorf(
					"ports.%s and ports.%s can collide across worktree slots; choose base ports with different remainders modulo port_stride",
					firstName,
					secondName,
				)
			}
		}
	}

	return nil
}

func sortedKeys[Value any](values map[string]Value) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
