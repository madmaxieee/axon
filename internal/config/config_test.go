package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/madmaxieee/axon/internal/utils"
)

func TestConfig_GetPatternByName(t *testing.T) {
	cfg := Config{
		ConfigFile: &ConfigFile{
			Patterns: []*Pattern{
				{Name: "test-pattern1"},
				{Name: "test-pattern2"},
			},
		},
	}

	// Test existing pattern
	p1 := cfg.GetPatternByName("test-pattern1")
	if p1 == nil || p1.Name != "test-pattern1" {
		t.Errorf("expected pattern test-pattern1, got %v", p1)
	}

	// Test non-existing pattern
	p3 := cfg.GetPatternByName("test-pattern3")
	if p3 != nil {
		t.Errorf("expected nil, got %v", p3)
	}

	// Test @ syntax (single prompt pattern)
	pAt := cfg.GetPatternByName("@my-prompt")
	if pAt == nil || pAt.Name != "@my-prompt" {
		t.Errorf("expected @my-prompt, got %v", pAt)
	}
	if len(pAt.Steps) != 1 || pAt.Steps[0].AIStep == nil || pAt.Steps[0].AIStep.Prompt != "@my-prompt" {
		t.Errorf("single prompt pattern structure incorrect")
	}
}

func TestConfig_GetAllPatternNames(t *testing.T) {
	cfg := Config{
		ConfigFile: &ConfigFile{
			Patterns: []*Pattern{
				{Name: "patternA"},
				{Name: "patternB"},
			},
		},
	}

	names := cfg.GetAllPatternNames()
	expected := []string{"patternA", "patternB"}
	if !reflect.DeepEqual(names, expected) {
		t.Errorf("expected %v, got %v", expected, names)
	}
}

func TestConfig_GetProviderByName(t *testing.T) {
	cfg := Config{
		ConfigFile: &ConfigFile{
			Providers: []*ProviderConfig{
				{Name: "provider1"},
				{Name: "provider2"},
			},
		},
	}

	p := cfg.GetProviderByName("provider2")
	if p == nil || p.Name != "provider2" {
		t.Errorf("expected provider2, got %v", p)
	}

	pNil := cfg.GetProviderByName("nonexistent")
	if pNil != nil {
		t.Errorf("expected nil, got %v", pNil)
	}
}

func TestConfig_GetQuiet(t *testing.T) {
	cfgDefault := Config{}
	if cfgDefault.GetQuiet() != false {
		t.Errorf("expected default quiet to be false")
	}

	cfgTrue := Config{Quiet: utils.BoolPtr(true)}
	if cfgTrue.GetQuiet() != true {
		t.Errorf("expected quiet to be true")
	}

	cfgFalse := Config{Quiet: utils.BoolPtr(false)}
	if cfgFalse.GetQuiet() != false {
		t.Errorf("expected quiet to be false")
	}
}

func TestConfig_Merge(t *testing.T) {
	baseCfg := Config{
		Quiet: utils.BoolPtr(false),
		ConfigFile: &ConfigFile{
			General: GeneralConfig{
				Model: utils.StringPtr("base-model"),
			},
			Providers: []*ProviderConfig{
				{Name: "provider1", BaseURL: utils.StringPtr("url1")},
			},
			Patterns: []*Pattern{
				{Name: "pattern1"},
			},
		},
	}

	overrideCfg := &Config{
		Quiet: utils.BoolPtr(true),
		ConfigFile: &ConfigFile{
			General: GeneralConfig{
				Model: utils.StringPtr("override-model"),
			},
			Providers: []*ProviderConfig{
				{Name: "provider1", BaseURL: utils.StringPtr("url1-override")},
				{Name: "provider2", BaseURL: utils.StringPtr("url2")},
			},
			Patterns: []*Pattern{
				{Name: "pattern2"},
			},
		},
	}

	err := baseCfg.Merge(overrideCfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *baseCfg.Quiet != true {
		t.Errorf("expected Quiet to be true, got %v", *baseCfg.Quiet)
	}
	if *baseCfg.General.Model != "override-model" {
		t.Errorf("expected General.Model to be override-model, got %v", *baseCfg.General.Model)
	}

	if len(baseCfg.Providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(baseCfg.Providers))
	}
	p1 := baseCfg.GetProviderByName("provider1")
	if *p1.BaseURL != "url1-override" {
		t.Errorf("expected provider1 baseurl to be url1-override, got %v", *p1.BaseURL)
	}
	p2 := baseCfg.GetProviderByName("provider2")
	if *p2.BaseURL != "url2" {
		t.Errorf("expected provider2 baseurl to be url2, got %v", *p2.BaseURL)
	}

	if len(baseCfg.Patterns) != 2 {
		t.Fatalf("expected 2 patterns, got %d", len(baseCfg.Patterns))
	}
}

func TestProviderConfig_GetAPIKey(t *testing.T) {
	// Test direct API Key
	pDirect := ProviderConfig{Name: "direct", APIKey: utils.StringPtr("direct-key")}
	key, err := pDirect.GetAPIKey()
	if err != nil || *key != "direct-key" {
		t.Errorf("expected direct-key, got key=%v, err=%v", key, err)
	}

	// Test Env Var
	os.Setenv("TEST_API_KEY_ENV", "env-key")
	defer os.Unsetenv("TEST_API_KEY_ENV")
	pEnv := ProviderConfig{Name: "env", APIKeyEnv: utils.StringPtr("TEST_API_KEY_ENV")}
	key, err = pEnv.GetAPIKey()
	if err != nil || *key != "env-key" {
		t.Errorf("expected env-key, got key=%v, err=%v", key, err)
	}

	// Test Command
	pCmd := ProviderConfig{Name: "cmd", APIKeyCmd: utils.StringPtr("echo cmd-key")}
	key, err = pCmd.GetAPIKey()
	if err != nil || *key != "cmd-key" {
		t.Errorf("expected cmd-key, got key=%v, err=%v", key, err)
	}

	// Test Missing
	pMissing := ProviderConfig{Name: "missing"}
	_, err = pMissing.GetAPIKey()
	if err == nil {
		t.Errorf("expected error for missing key")
	}
}

func TestEnsureConfig(t *testing.T) {
	// Create a temporary config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[general]
model = "my-custom-model"

[[providers]]
name = "custom-provider"
base_url = "https://custom.api.com"

[[patterns]]
name = "custom-pattern"
`
	err := os.WriteFile(configPath, []byte(tomlContent), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := EnsureConfig(&configPath)
	if err != nil {
		t.Fatalf("EnsureConfig failed: %v", err)
	}

	if *cfg.General.Model != "my-custom-model" {
		t.Errorf("expected model my-custom-model, got %v", *cfg.General.Model)
	}

	p := cfg.GetProviderByName("custom-provider")
	if p == nil || *p.BaseURL != "https://custom.api.com" {
		t.Errorf("provider not loaded correctly")
	}

	pat := cfg.GetPatternByName("custom-pattern")
	if pat == nil {
		t.Errorf("pattern not loaded correctly")
	}
}
