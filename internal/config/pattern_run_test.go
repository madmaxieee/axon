package config

import (
	"context"
	"strings"
	"testing"

	"github.com/madmaxieee/axon/internal/proto"
	"github.com/madmaxieee/axon/internal/utils"
)

func TestCommandStep_Run(t *testing.T) {
	cfg := &Config{
		Quiet: utils.BoolPtr(true),
	}

	ctx := context.Background()

	// Simple command
	step1 := CommandStep{
		Command: "echo 'hello world'",
	}
	args := proto.TemplateArgs{}
	out1, err := step1.Run(ctx, cfg, &args, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(*out1) != "hello world" {
		t.Errorf("expected 'hello world', got %q", *out1)
	}

	// Command with template args
	step2 := CommandStep{
		Command: "echo {{.PROMPT}}",
	}
	args2 := proto.TemplateArgs{
		"PROMPT": "templated value",
	}
	out2, err := step2.Run(ctx, cfg, &args2, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(*out2) != "templated value" {
		t.Errorf("expected 'templated value', got %q", *out2)
	}

	// Command with pipe in
	step3 := CommandStep{
		Command: "cat",
	}
	args3 := proto.TemplateArgs{
		PIPE_VAR: "piped input",
	}
	out3, err := step3.Run(ctx, cfg, &args3, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(*out3) != "piped input" {
		t.Errorf("expected 'piped input', got %q", *out3)
	}
}

func TestPattern_Run_CommandOnly(t *testing.T) {
	cfg := &Config{
		Quiet: utils.BoolPtr(true),
	}

	// Pattern with multiple command steps
	pattern := Pattern{
		Name: "test-pattern",
		Steps: []Step{
			{
				CommandStep: &CommandStep{
					Command: "echo 'step 1 output'",
				},
				Output: utils.StringPtr("STEP1_OUT"),
			},
			{
				CommandStep: &CommandStep{
					Command: "echo 'step 2 output'",
				},
				// implicitly goes to PIPE_VAR
			},
			{
				CommandStep: &CommandStep{
					Command: "echo \"prev pipe: $(cat), step 1 out: {{.STEP1_OUT}}\"",
					PipeIn:  utils.BoolPtr(true),
				},
			},
		},
	}

	ctx := context.Background()
	stdin := "initial stdin"
	prompt := "initial prompt"
	finalOut, err := pattern.Run(ctx, cfg, &stdin, &prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	finalOutTrimmed := strings.TrimSpace(finalOut)
	if !strings.Contains(finalOutTrimmed, "prev pipe: step 2 output") || !strings.Contains(finalOutTrimmed, "step 1 out: 'step 1 output") {
		t.Errorf("output not expected: %q", finalOutTrimmed)
	}
}
