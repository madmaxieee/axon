package config

import (
	"os"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Prompts  []Prompt
	Patterns []Pattern
}

type Prompt struct {
	Name     string
	Content  *string
	Template *string
}

type Pattern struct {
	Name  string
	Steps []Step
}

type Step struct {
	*CommandStep
	*AIStep
	Stdin  *bool
	Output *string
}

type CommandStep struct {
	Command string
}

type AIStep struct {
	Prompt string
}

func (cfg *Config) GetPatternByName(name string) *Pattern {
	for _, pattern := range cfg.Patterns {
		if pattern.Name == name {
			return &pattern
		}
	}
	return nil
}

func (cfg *Config) GetPromptByName(name string) *Prompt {
	for _, prompt := range cfg.Prompts {
		if prompt.Name == name {
			return &prompt
		}
	}
	return nil
}

func NewConfigFromTOML(data []byte) (*Config, error) {
	var config Config
	err := toml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func EnsureConfig(configFilePath *string) (*Config, error) {
	data, err := os.ReadFile(*configFilePath)
	if err != nil {
		return nil, err
	}
	return NewConfigFromTOML(data)
}
