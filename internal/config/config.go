package config

import (
	"errors"
	"os"

	"github.com/madmaxieee/axon/internal/utils"
	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	*ConfigFile
}

type ConfigFile struct {
	General   GeneralConfig
	Providers []ProviderConfig
	Prompts   []Prompt
	Patterns  []Pattern
}

type GeneralConfig struct {
	// in a form of provider/model
	Model *string
	// TODO: add other configs like temperature, max tokens, etc.
}

type ProviderConfig struct {
	Name      string
	BaseURL   *string `toml:"base_url"`
	APIKey    *string `toml:"api_key"`
	APIKeyEnv *string `toml:"api_key_env"`
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

var defaultConfig = Config{
	ConfigFile: &ConfigFile{
		Providers: []ProviderConfig{
			{
				Name:      "openai",
				BaseURL:   utils.StringPtr("https://api.openai.com/v1"),
				APIKey:    nil,
				APIKeyEnv: utils.StringPtr("OPENAI_API_KEY"),
			},
			{
				Name:      "google",
				BaseURL:   utils.StringPtr("https://generativelanguage.googleapis.com/v1beta2"),
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
		Prompts: []Prompt{
			{
				Name: "default",
				Template: utils.StringPtr(`
# IDENTITY and PURPOSE

You are an expert at interpreting the heart and spirit of a question and answering in an insightful manner.

# STEPS

- Deeply understand what's being asked.

- Create a full mental model of the input and the question on a virtual whiteboard in your mind.

- If the question is suitable to answer in bullet lists, answer the question with 3-5 concise bullet points of 10 words each.

- If the user request other formats (like code, essay, etc), answer in that format.

# OUTPUT INSTRUCTIONS

- Do not output warnings or notes—just the requested sections.

# INPUT:

INPUT:

{{ .PROMPT }}

{{ .STDIN }}`),
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

func (cfg *Config) GetPromptByName(name string) *Prompt {
	for _, prompt := range cfg.Prompts {
		if prompt.Name == name {
			return &prompt
		}
	}
	return nil
}

func (cfg *Config) GetProviderByName(name string) *ProviderConfig {
	for _, provider := range cfg.Providers {
		if provider.Name == name {
			return &provider
		}
	}
	return nil
}

func (cfg *Config) Override(override *Config) error {
	if override == nil {
		return nil
	}

	if override.General.Model != nil {
		cfg.General.Model = override.General.Model
	}

	for _, overrideProvider := range override.Providers {
		existingProvider := cfg.GetProviderByName(overrideProvider.Name)
		if existingProvider == nil {
			cfg.Providers = append(cfg.Providers, overrideProvider)
		} else {
			err := existingProvider.Override(&overrideProvider)
			if err != nil {
				return err
			}
		}
	}

	for _, overridePrompt := range override.Prompts {
		existingPrompt := cfg.GetPromptByName(overridePrompt.Name)
		if existingPrompt == nil {
			cfg.Prompts = append(cfg.Prompts, overridePrompt)
		} else {
			*existingPrompt = overridePrompt
		}
	}

	for _, overridePattern := range override.Patterns {
		existingPattern := cfg.GetPatternByName(overridePattern.Name)
		if existingPattern == nil {
			cfg.Patterns = append(cfg.Patterns, overridePattern)
		} else {
			*existingPattern = overridePattern
		}
	}

	return nil
}

func (prov *ProviderConfig) Override(override *ProviderConfig) error {
	if override == nil {
		return nil
	}
	if prov.Name != override.Name {
		return errors.New("cannot merge provider configs with different names")
	}
	if override.BaseURL != nil {
		prov.BaseURL = override.BaseURL
	}
	if override.APIKey != nil {
		prov.APIKey = override.APIKey
	}
	if override.APIKeyEnv != nil {
		prov.APIKeyEnv = override.APIKeyEnv
	}
	return nil
}

func (prov *ProviderConfig) GetAPIKey() *string {
	if prov.APIKey != nil {
		return prov.APIKey
	}
	if prov.APIKeyEnv != nil {
		if value, exists := os.LookupEnv(*prov.APIKeyEnv); exists {
			return &value
		}
	}
	return nil
}

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
	err = cfg.Override(&Config{ConfigFile: &configFile})
	return &cfg, err
}
