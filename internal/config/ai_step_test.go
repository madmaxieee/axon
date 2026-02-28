package config

import (
	"context"
	"strings"
	"testing"

	"github.com/madmaxieee/axon/internal/proto"
	"github.com/madmaxieee/axon/internal/utils"
)

func TestAIStep_Run_Errors(t *testing.T) {
	cfg := &Config{
		ConfigFile: &ConfigFile{
			General: GeneralConfig{
				Model: utils.StringPtr("openai/gpt-4"),
			},
			Providers: []*ProviderConfig{
				{
					Name:   "openai",
					APIKey: utils.StringPtr("fake-key"),
				},
			},
		},
	}

	ctx := context.Background()

	// Error 1: Prompt not found
	step1 := AIStep{
		Prompt: "@missing-prompt",
	}
	_, err := step1.Run(ctx, cfg, &proto.TemplateArgs{})
	if err == nil || !strings.Contains(err.Error(), "prompt missing-prompt not found") {
		t.Errorf("expected prompt not found error, got %v", err)
	}

	// Error 2: No user message
	sysContent := "just system"
	cfg2 := &Config{
		Prompts: map[string]Prompt{
			"sys-only": {
				Name:   "sys-only",
				System: &sysContent,
				loaded: true,
			},
		},
		ConfigFile: cfg.ConfigFile,
	}

	step2 := AIStep{
		Prompt: "@sys-only",
	}
	_, err = step2.Run(ctx, cfg2, &proto.TemplateArgs{})
	if err == nil || !strings.Contains(err.Error(), "No user message found") {
		t.Errorf("expected no user message error, got %v", err)
	}

	// Error 3: Bad template syntax in prompt
	badSystem := "system {{ .MISSING } " // syntax error
	cfg3 := &Config{
		Prompts: map[string]Prompt{
			"bad-syntax": {
				Name:   "bad-syntax",
				System: &badSystem,
				loaded: true,
			},
		},
		ConfigFile: cfg.ConfigFile,
	}

	step3 := AIStep{
		Prompt: "@bad-syntax",
	}
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic for bad template syntax, got none")
			}
		}()
		step3.Run(ctx, cfg3, &proto.TemplateArgs{})
	}()

	// Error 4: Client option error
	step4 := AIStep{
		Prompt: "direct content",
		Model:  utils.StringPtr("unknown-provider/gpt-4"),
	}
	args4 := proto.TemplateArgs{
		"PROMPT": "has user message",
	}
	_, err = step4.Run(ctx, cfg, &args4)
	if err == nil || !strings.Contains(err.Error(), "provider unknown-provider not found") {
		t.Errorf("expected client options error, got %v", err)
	}
}
