package config

import (
	"testing"

	"github.com/madmaxieee/axon/internal/utils"
)

func TestMerge_Nils(t *testing.T) {
	var cfg Config
	err := cfg.Merge(nil)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	cfg2 := Config{}
	err = cfg2.Merge(&Config{
		OverrideModel: utils.StringPtr("override"),
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if *cfg2.OverrideModel != "override" {
		t.Errorf("expected override, got %v", *cfg2.OverrideModel)
	}

	var gen GeneralConfig
	err = gen.Merge(nil)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	var prov ProviderConfig
	err = prov.Merge(nil)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestProviderConfig_Merge_DifferentNames(t *testing.T) {
	prov1 := ProviderConfig{Name: "p1"}
	prov2 := ProviderConfig{Name: "p2"}

	err := prov1.Merge(&prov2)
	if err == nil {
		t.Errorf("expected error merging different names")
	}
}

func TestProviderConfig_Merge_Fields(t *testing.T) {
	prov1 := ProviderConfig{Name: "p1"}
	prov2 := ProviderConfig{
		Name:      "p1",
		APIKeyEnv: utils.StringPtr("env"),
		APIKeyCmd: utils.StringPtr("cmd"),
	}

	err := prov1.Merge(&prov2)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if *prov1.APIKeyEnv != "env" {
		t.Errorf("expected env, got %v", *prov1.APIKeyEnv)
	}
	if *prov1.APIKeyCmd != "cmd" {
		t.Errorf("expected cmd, got %v", *prov1.APIKeyCmd)
	}
}

func TestConfig_Merge_Prompts(t *testing.T) {
	cfg1 := Config{}
	cfg2 := Config{
		Prompts: map[string]Prompt{
			"p1": {Name: "p1"},
		},
	}
	err := cfg1.Merge(&cfg2)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(cfg1.Prompts) != 1 {
		t.Errorf("expected 1 prompt, got %d", len(cfg1.Prompts))
	}
}
