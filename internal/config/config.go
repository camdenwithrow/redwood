package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

const FileName = "redwood.toml"

type Config struct {
	BaseBranch   string             `toml:"base_branch"`
	WorktreePath string             `toml:"worktree_path"`
	PortStride   int                `toml:"port_stride"`
	Ports        map[string]int     `toml:"ports"`
	Commands     map[string]Command `toml:"commands"`
}

type Command struct {
	Run   []string          `toml:"run"`
	Shell string            `toml:"shell"`
	Env   map[string]string `toml:"env"`
}

func (command *Command) UnmarshalTOML(value any) error {
	switch typed := value.(type) {
	case string:
		command.Shell = typed
		return nil
	case map[string]any:
		return decodeCommandTable(command, typed)
	default:
		return fmt.Errorf("must be a shell string or table")
	}
}

func decodeCommandTable(command *Command, value map[string]any) error {
	for field, raw := range value {
		switch field {
		case "run":
			arguments, ok := raw.([]any)
			if !ok {
				return fmt.Errorf("run must be an array of strings")
			}
			command.Run = make([]string, len(arguments))
			for index, argument := range arguments {
				text, ok := argument.(string)
				if !ok {
					return fmt.Errorf("run[%d] must be a string", index)
				}
				command.Run[index] = text
			}
		case "shell":
			text, ok := raw.(string)
			if !ok {
				return fmt.Errorf("shell must be a string")
			}
			command.Shell = text
		case "env":
			environment, ok := raw.(map[string]any)
			if !ok {
				return fmt.Errorf("env must be a table of strings")
			}
			command.Env = make(map[string]string, len(environment))
			for name, rawValue := range environment {
				text, ok := rawValue.(string)
				if !ok {
					return fmt.Errorf("env.%s must be a string", name)
				}
				command.Env[name] = text
			}
		default:
			return fmt.Errorf("unknown field %s", field)
		}
	}
	return nil
}

var (
	environmentNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	portPlaceholderPattern = regexp.MustCompile(`\{ports\.([^{}]+)\}`)
)

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

func ExpandPortPlaceholders(value string, ports map[string]int) string {
	return portPlaceholderPattern.ReplaceAllStringFunc(value, func(placeholder string) string {
		label := strings.TrimSuffix(strings.TrimPrefix(placeholder, "{ports."), "}")
		return fmt.Sprint(ports[label])
	})
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
		if len(command.Run) > 0 && strings.TrimSpace(command.Shell) != "" {
			return fmt.Errorf("commands.%s must define only one of run or shell", name)
		}
		if len(command.Run) == 0 && strings.TrimSpace(command.Shell) == "" {
			return fmt.Errorf("commands.%s must define run or shell", name)
		}
		if len(command.Run) > 0 && strings.TrimSpace(command.Run[0]) == "" {
			return fmt.Errorf("commands.%s.run[0] must not be empty", name)
		}
		for _, environmentName := range sortedKeys(command.Env) {
			if !environmentNamePattern.MatchString(environmentName) {
				return fmt.Errorf("commands.%s.env contains invalid environment variable name %q", name, environmentName)
			}
			for _, match := range portPlaceholderPattern.FindAllStringSubmatch(command.Env[environmentName], -1) {
				if _, exists := config.Ports[match[1]]; !exists {
					return fmt.Errorf("commands.%s.env.%s references unknown port %q", name, environmentName, match[1])
				}
			}
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
