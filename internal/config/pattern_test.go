package config

import (
	"context"
	"strings"
	"testing"

	"github.com/madmaxieee/axon/internal/utils"
)

func TestMakeSinglePromptPattern(t *testing.T) {
	pattern := MakeSinglePromptPattern("@test-prompt")
	if pattern.Name != "@test-prompt" {
		t.Errorf("expected @test-prompt, got %s", pattern.Name)
	}
	if len(pattern.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(pattern.Steps))
	}
	if pattern.Steps[0].AIStep == nil {
		t.Fatalf("expected AIStep to not be nil")
	}
	if pattern.Steps[0].AIStep.Prompt != "@test-prompt" {
		t.Errorf("expected @test-prompt, got %s", pattern.Steps[0].AIStep.Prompt)
	}

	// Test without @ prefix (should add it)
	patternNoAt := MakeSinglePromptPattern("another-prompt")
	if patternNoAt.Name != "@another-prompt" {
		t.Errorf("expected @another-prompt, got %s", patternNoAt.Name)
	}
	if patternNoAt.Steps[0].AIStep.Prompt != "@another-prompt" {
		t.Errorf("expected @another-prompt, got %s", patternNoAt.Steps[0].AIStep.Prompt)
	}
}

func TestSelectModelForStep(t *testing.T) {
	cfg := &Config{
		ConfigFile: &ConfigFile{
			General: GeneralConfig{
				Model: utils.StringPtr("default-model"),
			},
		},
	}

	// Step without override model
	stepNoModel := AIStep{Prompt: "test"}
	model1 := selectModelForStep(cfg, stepNoModel)
	if model1 != "default-model" {
		t.Errorf("expected default-model, got %s", model1)
	}

	// Step with override model
	stepWithModel := AIStep{Prompt: "test", Model: utils.StringPtr("step-model")}
	model2 := selectModelForStep(cfg, stepWithModel)
	if model2 != "step-model" {
		t.Errorf("expected step-model, got %s", model2)
	}

	// Global override model
	cfgOverride := &Config{
		OverrideModel: utils.StringPtr("global-override"),
		ConfigFile: &ConfigFile{
			General: GeneralConfig{
				Model: utils.StringPtr("default-model"),
			},
		},
	}
	model3 := selectModelForStep(cfgOverride, stepWithModel)
	if model3 != "global-override" {
		t.Errorf("expected global-override, got %s", model3)
	}
}

func TestExplain(t *testing.T) {
	cfg := &Config{
		Prompts: map[string]Prompt{
			"test-prompt": {
				Name: "test-prompt",
				Path: utils.StringPtr("/path/to/prompt"),
			},
		},
		ConfigFile: &ConfigFile{
			General: GeneralConfig{
				Model: utils.StringPtr("test-model"),
			},
		},
	}

	pattern := Pattern{
		Name: "test-explain",
		Steps: []Step{
			{
				AIStep: &AIStep{
					Prompt: "@test-prompt",
					Model:  utils.StringPtr("custom-model"),
				},
			},
			{
				CommandStep: &CommandStep{
					Command: "echo hello",
				},
				Output: utils.StringPtr("result"),
			},
		},
	}

	ctx := context.Background()
	explanation, err := pattern.Explain(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(explanation, "Pattern: test-explain") {
		t.Errorf("explanation missing pattern name")
	}
	if !strings.Contains(explanation, "Type: AI Step") {
		t.Errorf("explanation missing AI Step type")
	}
	if !strings.Contains(explanation, "Model: custom-model") {
		t.Errorf("explanation missing custom-model")
	}
	if !strings.Contains(explanation, "Type: Command Step") {
		t.Errorf("explanation missing Command Step type")
	}
	if !strings.Contains(explanation, "Command: `echo hello`") {
		t.Errorf("explanation missing command string")
	}
	if !strings.Contains(explanation, "==> $result") {
		t.Errorf("explanation missing output variable")
	}
}
