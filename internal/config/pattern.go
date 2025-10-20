package config

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/madmaxieee/axon/internal/client"
	"github.com/madmaxieee/axon/internal/proto"
	"github.com/madmaxieee/axon/internal/utils"
	"github.com/openai/openai-go/v3"
)

func (p *Pattern) Run(ctx context.Context, cfg *Config, stdin *string, prompt *string) (string, error) {
	templateArgs := make(proto.TemplateArgs)
	if stdin != nil {
		templateArgs["INPUT"] = *stdin
	} else {
		templateArgs["INPUT"] = ""
	}
	if prompt != nil {
		templateArgs["PROMPT"] = *prompt
	} else {
		templateArgs["PROMPT"] = ""
	}

	providerName, err := cfg.GetProviderName()
	if err != nil {
		return "", err
	}

	providerCfg := cfg.GetProviderByName(providerName)
	if providerCfg == nil {
		return "", fmt.Errorf("provider %s not found", providerName)
	}

	baseURL := providerCfg.BaseURL
	if baseURL == nil {
		return "", fmt.Errorf("base URL for provider %s not found", providerName)
	}

	modelName, err := cfg.GetModelName()
	if err != nil {
		return "", err
	}

	apiKey, err := providerCfg.GetAPIKey()
	if apiKey == nil {
		if err != nil {
			println(err.Error())
		}
		return "", fmt.Errorf("API key for provider %s not found", providerName)
	}

	client := client.NewClient(client.ClientOptions{
		ProviderName: providerName,
		ModelName:    modelName,
		BaseURL:      *providerCfg.BaseURL,
		APIKey:       *apiKey,
	})

	for _, step := range p.Steps {
		needsInput := true
		if step.NeedsInput != nil {
			needsInput = *step.NeedsInput
		}

		var output *string
		var err error
		if step.AIStep != nil {
			output, err = step.AIStep.Run(ctx, cfg, client, &templateArgs)
		} else if step.CommandStep != nil {
			output, err = step.CommandStep.Run(&templateArgs, needsInput)
		} else {
			return "", fmt.Errorf("step has neither AIStep nor CommandStep defined")
		}
		if err != nil {
			return "", err
		}
		if step.Output != nil {
			templateArgs[*step.Output] = *output
		} else {
			templateArgs["INPUT"] = *output
		}
	}

	return templateArgs["INPUT"], nil
}

func (step *AIStep) Run(ctx context.Context, cfg *Config, client *client.Client, templateArgs *proto.TemplateArgs) (*string, error) {
	var prompt *Prompt

	if strings.HasPrefix(step.Prompt, "@") {
		promptName := step.Prompt[1:]
		var err error
		prompt, err = cfg.GetPromptByName(promptName)
		if err != nil {
			return nil, err
		}
		if prompt == nil {
			return nil, fmt.Errorf("prompt %s not found", step.Prompt)
		}
	} else {
		prompt = &Prompt{
			System: &step.Prompt,
			User:   nil,
			loaded: true,
		}
	}

	messages := []openai.ChatCompletionMessageParamUnion{}

	if prompt.System != nil {
		tmpl := template.Must(template.New("system").Parse(*prompt.System))
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, templateArgs); err != nil {
			return nil, err
		}
		systemPrompt := buf.String()
		messages = append(messages, openai.SystemMessage(systemPrompt))
	}

	if prompt.User != nil {
		tmpl := template.Must(template.New("user").Parse(*prompt.User))
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, templateArgs); err != nil {
			return nil, err
		}
		userPrompt := buf.String()
		messages = append(messages, openai.UserMessage(userPrompt))
	} else {
		if userPrompt, ok := (*templateArgs)["PROMPT"]; ok && userPrompt != "" {
			messages = append(messages, openai.UserMessage((*templateArgs)["PROMPT"]))
		}
		if input, ok := (*templateArgs)["INPUT"]; ok && input != "" {
			messages = append(messages, openai.UserMessage((*templateArgs)["INPUT"]))
		}
	}

	stream := client.Request(ctx, proto.Request{Messages: messages})

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

func (step *CommandStep) Run(templateArgs *proto.TemplateArgs, needInput bool) (*string, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	shellQuotedArgs := make(proto.TemplateArgs)
	for k, v := range *templateArgs {
		shellQuotedArgs[k] = utils.ShellQuote(v)
	}

	tmpl := template.Must(template.New("command").Parse(step.Command))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, shellQuotedArgs); err != nil {
		return nil, err
	}
	command := buf.String()

	cmd := exec.Command(shell, "-c", command)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = io.Discard
	if stdin, ok := (*templateArgs)["INPUT"]; ok && needInput {
		cmd.Stdin = bytes.NewBufferString(stdin)
	}

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("command failed: %w", err)
	}

	outputString := stdout.String()

	return &outputString, nil
}
