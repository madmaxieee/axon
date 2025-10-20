package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/madmaxieee/axon/internal/utils"
	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	*ConfigFile
	Prompts map[string]Prompt
}

type ConfigFile struct {
	General   GeneralConfig
	Providers []ProviderConfig
	Patterns  []Pattern
}

type GeneralConfig struct {
	// in a form of provider/model
	PromptPath []string
	Model      *string
	// TODO: add other configs like temperature, max tokens, etc.
}

type ProviderConfig struct {
	Name      string
	BaseURL   *string `toml:"base_url"`
	APIKey    *string `toml:"api_key"`
	APIKeyEnv *string `toml:"api_key_env"`
}

type Prompt struct {
	Name   string
	System *string
	User   *string
	Path   *string
	loaded bool
}

type Pattern struct {
	Name  string
	Steps []Step
}

type Step struct {
	*CommandStep
	*AIStep
	NeedsInput *bool
	Output     *string
}

type CommandStep struct {
	Command string
}

type AIStep struct {
	Prompt string
}

var defaultConfig = Config{
	Prompts: map[string]Prompt{
		"default": {
			Name: "default",
			Path: nil,
			System: utils.StringPtr(`
# IDENTITY and PURPOSE

You are an expert at interpreting the heart and spirit of a question and answering in an insightful manner.

# STEPS

- Deeply understand what's being asked.

- Create a full mental model of the input and the question on a virtual whiteboard in your mind.

# OUTPUT INSTRUCTIONS

- Do not output warnings or notes—just the requested sections.
`),
		},
	},
	ConfigFile: &ConfigFile{
		General: GeneralConfig{
			PromptPath: []string{filepath.Join(xdg.ConfigHome, "axon", "prompts")},
			Model:      utils.StringPtr("openai/gpt-4o"),
		},
		Providers: []ProviderConfig{
			{
				Name:      "openai",
				BaseURL:   utils.StringPtr("https://api.openai.com/v1"),
				APIKey:    nil,
				APIKeyEnv: utils.StringPtr("OPENAI_API_KEY"),
			},
			{
				Name:      "google",
				BaseURL:   utils.StringPtr("https://generativelanguage.googleapis.com/v1beta"),
				APIKey:    nil,
				APIKeyEnv: utils.StringPtr("GOOGLE_API_KEY"),
			},
			{
				Name:      "anthropic",
				BaseURL:   utils.StringPtr("https://api.anthropic.com/v1"),
				APIKey:    nil,
				APIKeyEnv: utils.StringPtr("ANTHROPIC_API_KEY"),
			},
		},
		Patterns: []Pattern{
			{
				Name: "default",
				Steps: []Step{
					{
						AIStep: &AIStep{Prompt: "default"},
					},
				},
			},
		},
	},
}

func (cfg *Config) GetPatternByName(name string) *Pattern {
	for _, pattern := range cfg.Patterns {
		if pattern.Name == name {
			return &pattern
		}
	}
	return nil
}

func (prompt *Prompt) LoadContent() (bool, error) {
	if prompt.loaded {
		return true, nil
	}

	if prompt.Path == nil {
		return false, errors.New("prompt has no content or path")
	}

	stats, err := os.Stat(*prompt.Path)
	if err != nil {
		return false, err
	}

	if stats.IsDir() {
		if prompt.System == nil {
			systemPath := filepath.Join(*prompt.Path, "system.md")
			data, err := os.ReadFile(systemPath)
			if err != nil {
				return false, err
			}
			content := string(data)
			prompt.System = &content
		}
		if prompt.User == nil {
			userPath := filepath.Join(*prompt.Path, "user.md")
			data, err := os.ReadFile(userPath)
			if err != nil {
				return false, err
			}
			content := string(data)
			prompt.User = &content
		}
		if prompt.System == nil && prompt.User == nil {
			return false, nil
		}
	} else if stats.Mode().IsRegular() {
		data, err := os.ReadFile(*prompt.Path)
		if err != nil {
			return false, err
		}
		content := string(data)
		prompt.System = &content
	}

	prompt.loaded = true
	return true, nil
}

func (cfg *Config) GetPromptByName(name string) (*Prompt, error) {
	if prompt, ok := cfg.Prompts[name]; ok {
		prompt.LoadContent()
		return &prompt, nil
	}

	for _, root := range cfg.General.PromptPath {
		filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if path != root {
				if d.Name() == name {
					cfg.Prompts[name] = Prompt{
						Name:   name,
						Path:   utils.StringPtr(path),
						loaded: false,
						System: nil,
						User:   nil,
					}
				}
			}
			return nil
		})
	}

	if prompt, ok := cfg.Prompts[name]; ok {
		prompt.LoadContent()
		return &prompt, nil
	} else {
		return nil, errors.New("prompt " + name + " not found")
	}
}

func (cfg *Config) GetProviderByName(name string) *ProviderConfig {
	for _, provider := range cfg.Providers {
		if provider.Name == name {
			return &provider
		}
	}
	return nil
}

func parseModelString(modelStr string) (string, string, error) {
	parts := strings.SplitN(modelStr, "/", 2)
	if len(parts) != 2 {
		return "", "", errors.New("invalid model string format, expected provider/model, e.g. openai/gpt-4o")
	}
	// provider, model
	return parts[0], parts[1], nil
}

func (cfg *Config) GetProviderName() (string, error) {
	provider, _, err := parseModelString(*cfg.General.Model)
	if err != nil {
		return "", err
	}
	return provider, nil
}

func (cfg *Config) GetModelName() (string, error) {
	_, model, err := parseModelString(*cfg.General.Model)
	if err != nil {
		return "", err
	}
	return model, nil
}

func (cfg *Config) Merge(other *Config) error {
	if other == nil {
		return nil
	}

	err := cfg.General.Merge(&other.General)
	if err != nil {
		return err
	}

	for _, overrideProvider := range other.Providers {
		existingProvider := cfg.GetProviderByName(overrideProvider.Name)
		if existingProvider == nil {
			cfg.Providers = append(cfg.Providers, overrideProvider)
		} else {
			err := existingProvider.Merge(&overrideProvider)
			if err != nil {
				return err
			}
		}
	}

	for _, overridePattern := range other.Patterns {
		existingPattern := cfg.GetPatternByName(overridePattern.Name)
		if existingPattern == nil {
			cfg.Patterns = append(cfg.Patterns, overridePattern)
		} else {
			*existingPattern = overridePattern
		}
	}

	return nil
}

func (prov *ProviderConfig) Merge(other *ProviderConfig) error {
	if other == nil {
		return nil
	}
	if prov.Name != other.Name {
		return errors.New("cannot merge provider configs with different names")
	}
	if other.BaseURL != nil {
		prov.BaseURL = other.BaseURL
	}
	if other.APIKey != nil {
		prov.APIKey = other.APIKey
	}
	if other.APIKeyEnv != nil {
		prov.APIKeyEnv = other.APIKeyEnv
	}
	return nil
}

func (prov *ProviderConfig) GetAPIKey() (*string, error) {
	if prov.APIKey != nil {
		return prov.APIKey, nil
	}
	if prov.APIKeyEnv != nil {
		if value, exists := os.LookupEnv(*prov.APIKeyEnv); exists {
			return &value, nil
		} else {
			return nil, errors.New("environment variable " + *prov.APIKeyEnv + " not set")
		}
	}
	return nil, errors.New("no API key or environment variable specified for provider " + prov.Name)
}

func (cfg *GeneralConfig) Merge(other *GeneralConfig) error {
	if other == nil {
		return nil
	}
	if other.Model != nil {
		cfg.Model = other.Model
	}
	if other.PromptPath != nil {
		cfg.PromptPath = append(cfg.PromptPath, other.PromptPath...)
	}
	return nil
}

// TODO: read and combine all config files from $XDG_CONFIG_HOME/axon/
func EnsureConfig(configFilePath *string) (*Config, error) {
	data, err := os.ReadFile(*configFilePath)

	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	var configFile ConfigFile
	err = toml.Unmarshal(data, &configFile)
	if err != nil {
		return nil, err
	}

	cfg := defaultConfig
	err = cfg.Merge(&Config{ConfigFile: &configFile})
	return &cfg, err
}
