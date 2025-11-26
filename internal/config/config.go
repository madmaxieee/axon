package config

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/madmaxieee/axon/internal/client"
	"github.com/madmaxieee/axon/internal/proto"
	"github.com/madmaxieee/axon/internal/utils"
	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	OverrideModel *string
	Quiet         *bool
	Prompts       map[string]Prompt
	*ConfigFile
}

type ConfigFile struct {
	General   GeneralConfig
	Providers []*ProviderConfig
	Patterns  []*Pattern
}

type GeneralConfig struct {
	// in a form of provider/model
	PromptPath []string `toml:"prompt_path"`
	Model      *string
	// TODO: add other configs like temperature, max tokens, etc.
}

type ProviderConfig struct {
	Name      string
	BaseURL   *string `toml:"base_url"`
	APIKey    *string `toml:"api_key"`
	APIKeyEnv *string `toml:"api_key_env"`
	APIKeyCmd *string `toml:"api_key_cmd"`
}

type Prompt struct {
	Name   string
	System *string
	User   *string
	Path   *string
	loaded bool
}

type Pattern struct {
	// TODO: add description field
	Name  string
	Steps []Step
}

type Step struct {
	*CommandStep
	*AIStep
	Output *string // the name of the output variable to store the result of this step
}

type CommandStep struct {
	Command string // the command to run, will be ran with $SHELL -c
	PipeIn  *bool  `toml:"pipe_in"` // whether or not to pipe the previous step's output as input to this step
	Tty     bool   // whether to connect the running command to a TTY, can't capture output if true
}

type AIStep struct {
	Prompt string  // the prompt to use @<prompt_name> or direct content
	Model  *string // optional override model for this step
}

var defaultConfig = Config{
	OverrideModel: nil,
	Quiet:         utils.BoolPtr(false),
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
			PromptPath: []string{filepath.Join(GetConfigHome(), "prompts")},
			Model:      utils.StringPtr("openai/gpt-4o"),
		},
		Providers: []*ProviderConfig{
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
				APIKeyEnv: utils.StringPtr("GEMINI_API_KEY"),
			},
			{
				Name:      "anthropic",
				BaseURL:   utils.StringPtr("https://api.anthropic.com/v1"),
				APIKey:    nil,
				APIKeyEnv: utils.StringPtr("ANTHROPIC_API_KEY"),
			},
		},
		Patterns: []*Pattern{
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
	if after, ok := strings.CutPrefix(name, "@"); ok {
		pattern := MakeSinglePromptPattern(after)
		return &pattern
	}

	for _, pattern := range cfg.Patterns {
		if pattern.Name == name {
			return pattern
		}
	}

	return nil
}

func (cfg Config) GetQuiet() bool {
	return utils.DefaultBool(cfg.Quiet, false)
}

func (cfg *Config) GetAllPatternNames() []string {
	names := make([]string, 0, len(cfg.Patterns))
	for _, pattern := range cfg.Patterns {
		names = append(names, pattern.Name)
	}
	return names
}

func (cfg *Config) GetAllPromptNames() []string {
	cfg.scanPromptPath()
	names := make([]string, 0, len(cfg.Prompts))
	for name := range cfg.Prompts {
		names = append(names, name)
	}
	return names
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
			prompt.System = utils.RemoveWhitespace(content)
		}
		if prompt.User == nil {
			userPath := filepath.Join(*prompt.Path, "user.md")
			data, err := os.ReadFile(userPath)
			if err != nil {
				return false, err
			}
			content := string(data)
			prompt.User = utils.RemoveWhitespace(content)
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

func (cfg *Config) scanPromptPath() {
	for _, root := range cfg.General.PromptPath {
		if !filepath.IsAbs(root) {
			root = filepath.Join(GetConfigHome(), root)
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			fullPath := filepath.Join(root, entry.Name())

			// resolve symlink
			{
				info, err := entry.Info()
				if err != nil {
					continue
				}
				if info.Mode()&os.ModeSymlink != 0 {
					fullPath, err = filepath.EvalSymlinks(fullPath)
					if err != nil {
						continue
					}
				}
			}

			stat, err := os.Stat(fullPath)
			if err != nil {
				continue
			}

			if stat.IsDir() {
				if _, exists := cfg.Prompts[stat.Name()]; exists {
					continue
				}
				cfg.Prompts[stat.Name()] = Prompt{
					Name:   stat.Name(),
					Path:   utils.StringPtr(fullPath),
					loaded: false,
					System: nil,
					User:   nil,
				}
			} else if strings.HasSuffix(entry.Name(), ".md") {
				promptName := strings.TrimSuffix(entry.Name(), ".md")
				if _, exists := cfg.Prompts[promptName]; exists {
					continue
				}
				cfg.Prompts[promptName] = Prompt{
					Name:   promptName,
					Path:   utils.StringPtr(fullPath),
					loaded: false,
					System: nil,
					User:   nil,
				}
			}
		}
	}
}

func (cfg *Config) GetPromptByName(name string) (*Prompt, error) {
	if prompt, ok := cfg.Prompts[name]; ok {
		prompt.LoadContent()
		return &prompt, nil
	}

	cfg.scanPromptPath()

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
			return provider
		}
	}
	return nil
}

func (cfg *Config) GetClientOptions(modelKey string) (*client.ClientOptions, error) {
	providerName, modelName, err := client.ParseModelString(modelKey)
	if err != nil {
		return nil, err
	}

	provider := cfg.GetProviderByName(providerName)
	if provider == nil {
		return nil, errors.New("provider " + providerName + " not found")
	}

	apiKey, err := provider.GetAPIKey()
	if err != nil {
		return nil, err
	}

	baseURL := ""
	if provider.BaseURL != nil {
		baseURL = *provider.BaseURL
	}

	return &client.ClientOptions{
		ProviderName: providerName,
		ModelName:    modelName,
		BaseURL:      baseURL,
		APIKey:       *apiKey,
	}, nil
}

func (cfg *Config) Merge(other *Config) error {
	if other == nil {
		return nil
	}

	if other.OverrideModel != nil {
		cfg.OverrideModel = other.OverrideModel
	}

	if other.Quiet != nil {
		cfg.Quiet = other.Quiet
	}

	if other.Prompts != nil {
		if cfg.Prompts == nil {
			cfg.Prompts = make(map[string]Prompt)
		}
		maps.Copy(cfg.Prompts, other.Prompts)
	}

	if other.ConfigFile == nil {
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
			err := existingProvider.Merge(overrideProvider)
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
			*existingPattern = *overridePattern
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
	if other.APIKeyCmd != nil {
		prov.APIKeyCmd = other.APIKeyCmd
	}
	return nil
}

func (prov *ProviderConfig) GetAPIKey() (*string, error) {
	if prov.APIKey != nil {
		return prov.APIKey, nil
	}
	if prov.APIKeyEnv != nil {
		if value, exists := os.LookupEnv(*prov.APIKeyEnv); exists {
			prov.APIKey = utils.RemoveWhitespace(value)
			if prov.APIKey != nil {
				return prov.APIKey, nil
			}
		}
	}
	if prov.APIKeyCmd != nil {
		shell := utils.GetShell()
		cmd := exec.Command(shell, "-c", *prov.APIKeyCmd)
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("command failed: %w", err)
		}
		outputString := strings.TrimSpace(stdout.String())
		prov.APIKey = utils.RemoveWhitespace(outputString)
		if prov.APIKey != nil {
			return prov.APIKey, nil
		}
	}
	return nil, errors.New("no API key, environment variable or command specified for provider " + prov.Name)
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

func GetOverrideConfig(flags proto.Flags) *Config {
	overrideCfg := &Config{
		OverrideModel: utils.RemoveWhitespace(flags.Model),
		Quiet:         utils.BoolPtr(flags.Quiet),
	}
	return overrideCfg
}
