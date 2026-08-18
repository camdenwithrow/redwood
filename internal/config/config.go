package config

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

const FileName = "redwood.toml"

type Config struct {
	BaseBranch   string            `toml:"base_branch"`
	WorktreePath string            `toml:"worktree_path"`
	PortStride   int               `toml:"port_stride"`
	Ports        map[string]int    `toml:"ports"`
	Commands     map[string]string `toml:"commands"`
	Hooks        Hooks             `toml:"hooks"`
}

type Hooks struct {
	PostCreate []string `toml:"post_create"`
}

func PortEnvironmentVariable(label string) string {
	var name strings.Builder
	name.WriteString("RW_PORT_")
	previousUnderscore := true
	for _, character := range strings.ToUpper(label) {
		if character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' {
			name.WriteRune(character)
			previousUnderscore = false
			continue
		}
		if !previousUnderscore {
			name.WriteByte('_')
			previousUnderscore = true
		}
	}
	return strings.TrimSuffix(name.String(), "_")
}

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
	if len(config.Ports) > 0 && config.PortStride <= 0 {
		return fmt.Errorf("port_stride must be greater than zero")
	}

	portNames := sortedKeys(config.Ports)
	portEnvironmentVariables := make(map[string]string, len(portNames))
	for _, name := range portNames {
		port := config.Ports[name]
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("ports contains an empty name")
		}
		if port < 1 || port > 65535 {
			return fmt.Errorf("ports.%s must be between 1 and 65535", name)
		}
		environmentVariable := PortEnvironmentVariable(name)
		if environmentVariable == "RW_PORT" {
			return fmt.Errorf("ports.%s must contain at least one ASCII letter or digit", name)
		}
		if existingName, exists := portEnvironmentVariables[environmentVariable]; exists {
			return fmt.Errorf(
				"ports.%s and ports.%s map to the same environment variable %s",
				existingName,
				name,
				environmentVariable,
			)
		}
		portEnvironmentVariables[environmentVariable] = name
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

	for index, command := range config.Hooks.PostCreate {
		if strings.TrimSpace(command) == "" {
			return fmt.Errorf("hooks.post_create[%d] must not be empty", index)
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
