package config

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"text/template"

	"github.com/madmaxieee/axon/internal/client"
	"github.com/madmaxieee/axon/internal/proto"
	"github.com/openai/openai-go/v3"
)

func (p *Pattern) Run(ctx context.Context, cfg *Config, stdin *string, prompt *string) (string, error) {
	templateArgs := make(map[string]string)
	if stdin != nil {
		templateArgs["STDIN"] = *stdin
	} else {
		templateArgs["STDIN"] = ""
	}
	if prompt != nil {
		templateArgs["PROMPT"] = *prompt
	} else {
		templateArgs["PROMPT"] = ""
	}

	// TODO: support configurable client options
	// TODO: like model, provider, temperature, etc.
	client := client.NewClient()

	for _, step := range p.Steps {
		useStdin := true
		if step.Stdin != nil {
			useStdin = *step.Stdin
		}

		var output *string
		var err error
		if step.AIStep != nil {
			output, err = step.AIStep.Run(ctx, cfg, client, &templateArgs, useStdin)
		} else if step.CommandStep != nil {
			output, err = step.CommandStep.Run(&templateArgs, useStdin)
		} else {
			return "", fmt.Errorf("step has neither AIStep nor CommandStep defined")
		}
		if err != nil {
			return "", err
		}
		if step.Output != nil {
			templateArgs[*step.Output] = *output
		} else {
			templateArgs["STDIN"] = *output
		}
	}

	return templateArgs["STDIN"], nil
}

func (step *AIStep) Run(ctx context.Context, cfg *Config, client *client.Client, templateArgs *map[string]string, useStdin bool) (*string, error) {

	prompt := cfg.GetPromptByName(step.Prompt)

	if prompt == nil {
		return nil, fmt.Errorf("prompt %s not found", step.Prompt)
	}

	// TODO: certralize validation logic
	if prompt.Content == nil && prompt.Template == nil {
		return nil, fmt.Errorf("prompt %s has no content or template", step.Prompt)
	}
	if prompt.Content != nil && prompt.Template != nil {
		return nil, fmt.Errorf("prompt %s has both content and template defined", step.Prompt)
	}

	var templateStr string
	if prompt.Template != nil {
		templateStr = *prompt.Template
	} else if prompt.Content != nil {
		templateStr = *prompt.Content
		templateStr += `
# INPUT:

{{ .PROMPT }}

{{ .STDIN }}
`
	}

	tmpl := template.Must(template.New("prompt").Parse(templateStr))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateArgs); err != nil {
		return nil, err
	}
	promptStr := buf.String()

	stream := client.Request(ctx, proto.Request{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(promptStr),
		},
	})

	completion, err := stream.Collect(
		func(chunk openai.ChatCompletionChunk) {
			print(chunk.Choices[0].Delta.Content)
		},
	)
	if err != nil {
		return nil, err
	}

	return &completion.Choices[0].Message.Content, nil
}

func (step *CommandStep) Run(templateArgs *map[string]string, useStdin bool) (*string, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	cmd := exec.Command(shell, "-c", step.Command)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = io.Discard
	if stdin, ok := (*templateArgs)["STDIN"]; ok && useStdin {
		cmd.Stdin = bytes.NewBufferString(stdin)
	}

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("command failed: %w", err)
	}

	outputString := stdout.String()

	return &outputString, nil
}
