package config

import (
	"testing"

	"github.com/madmaxieee/axon/internal/proto"
	"github.com/madmaxieee/axon/internal/utils"
)

func TestGetOverrideConfig(t *testing.T) {
	flags := proto.Flags{
		Model: "override-model",
		Quiet: true,
	}

	cfg := GetOverrideConfig(flags)
	if cfg.OverrideModel == nil || *cfg.OverrideModel != "override-model" {
		t.Errorf("expected override-model, got %v", cfg.OverrideModel)
	}
	if cfg.Quiet == nil || *cfg.Quiet != true {
		t.Errorf("expected Quiet to be true, got %v", cfg.Quiet)
	}

	// Test with empty string for Model
	flagsEmpty := proto.Flags{
		Model: "   ",
		Quiet: false,
	}
	cfgEmpty := GetOverrideConfig(flagsEmpty)
	if cfgEmpty.OverrideModel != nil {
		t.Errorf("expected OverrideModel to be nil for empty string, got %v", *cfgEmpty.OverrideModel)
	}
	if cfgEmpty.Quiet == nil || *cfgEmpty.Quiet != false {
		t.Errorf("expected Quiet to be false, got %v", cfgEmpty.Quiet)
	}
}

func TestGetAllPromptNames(t *testing.T) {
	cfg := Config{
		Prompts: map[string]Prompt{
			"prompt1": {Name: "prompt1"},
			"prompt2": {Name: "prompt2"},
		},
		ConfigFile: &ConfigFile{},
	}

	names := cfg.GetAllPromptNames()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	hasP1 := false
	hasP2 := false
	for _, n := range names {
		if n == "prompt1" {
			hasP1 = true
		}
		if n == "prompt2" {
			hasP2 = true
		}
	}
	if !hasP1 || !hasP2 {
		t.Errorf("missing expected prompt names, got %v", names)
	}
}

func TestGetClientOptions(t *testing.T) {
	cfg := Config{
		ConfigFile: &ConfigFile{
			Providers: []*ProviderConfig{
				{
					Name:    "openai",
					BaseURL: utils.StringPtr("https://api.openai.com/v1"),
					APIKey:  utils.StringPtr("test-key"),
				},
			},
		},
	}

	// Successful parsing
	opts, err := cfg.GetClientOptions("openai/gpt-4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.ProviderName != "openai" {
		t.Errorf("expected openai, got %v", opts.ProviderName)
	}
	if opts.ModelName != "gpt-4" {
		t.Errorf("expected gpt-4, got %v", opts.ModelName)
	}
	if opts.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("expected baseurl, got %v", opts.BaseURL)
	}
	if opts.APIKey != "test-key" {
		t.Errorf("expected test-key, got %v", opts.APIKey)
	}

	// Invalid model string
	_, err = cfg.GetClientOptions("invalid-format")
	if err == nil {
		t.Errorf("expected error for invalid format")
	}

	// Missing provider
	_, err = cfg.GetClientOptions("unknown/gpt-4")
	if err == nil || err.Error() != "provider unknown not found" {
		t.Errorf("expected provider not found error, got %v", err)
	}

	// Missing API key
	cfgNoKey := Config{
		ConfigFile: &ConfigFile{
			Providers: []*ProviderConfig{
				{Name: "nokey"},
			},
		},
	}
	_, err = cfgNoKey.GetClientOptions("nokey/model")
	if err == nil {
		t.Errorf("expected error for missing API key")
	}
}
