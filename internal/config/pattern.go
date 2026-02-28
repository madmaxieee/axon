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

	"github.com/madmaxieee/axon/internal"
	"github.com/madmaxieee/axon/internal/client"
	"github.com/madmaxieee/axon/internal/proto"
	"github.com/madmaxieee/axon/internal/utils"
	"github.com/openai/openai-go/v3"
)

const (
	INPUT_VAR  = "INPUT"
	PROMPT_VAR = "PROMPT"
)

var PIPE_VAR = fmt.Sprintf("PIPE_%s", utils.Nonce())

func (p *Pattern) Run(ctx context.Context, cfg *Config, stdin *string, prompt *string) (string, error) {
	templateArgs := make(proto.TemplateArgs)
	if stdin != nil {
		templateArgs[INPUT_VAR] = *stdin
		templateArgs[PIPE_VAR] = *stdin
	} else {
		templateArgs[INPUT_VAR] = ""
		templateArgs[PIPE_VAR] = ""
	}
	if prompt != nil {
		templateArgs[PROMPT_VAR] = *prompt
	} else {
		templateArgs[PROMPT_VAR] = ""
	}

	for _, step := range p.Steps {
		var output *string
		var err error
		if step.AIStep != nil {
			output, err = step.AIStep.Run(ctx, cfg, &templateArgs)
		} else if step.CommandStep != nil {
			output, err = step.CommandStep.Run(ctx, cfg, &templateArgs, utils.DefaultBool(step.PipeIn, false))
		} else {
			return "", fmt.Errorf("step has neither AIStep nor CommandStep defined")
		}
		if err != nil {
			return "", err
		}

		if output != nil {
			if step.Output != nil {
				templateArgs[*step.Output] = *output
			} else {
				templateArgs[PIPE_VAR] = *output
			}
		}
	}

	return templateArgs[PIPE_VAR], nil
}

func (pattern Pattern) Explain(ctx context.Context, cfg *Config) (string, error) {
	var explanation strings.Builder
	explanation.WriteString(fmt.Sprintf("Pattern: %s\n\n", pattern.Name))
	for i, step := range pattern.Steps {
		explanation.WriteString(fmt.Sprintf("Step %d:\n", i+1))
		if step.AIStep != nil {
			explanation.WriteString("  Type: AI Step\n")
			explanation.WriteString(fmt.Sprintf("  Model: %s\n", selectModelForStep(cfg, *step.AIStep)))
			explanation.WriteString(fmt.Sprintf("  Prompt: %s\n", step.AIStep.Prompt))
			if strings.HasPrefix(step.AIStep.Prompt, "@") {
				promptName := step.AIStep.Prompt[1:]
				prompt, err := cfg.GetPromptByName(promptName)
				if err != nil {
					return "", err
				}
				if prompt == nil {
					explanation.WriteString("  (Prompt not found)\n")
				}
				explanation.WriteString(fmt.Sprintf("  Stored in: %s\n", *prompt.Path))
			}
		} else if step.CommandStep != nil {
			explanation.WriteString("  Type: Command Step\n")
			explanation.WriteString(fmt.Sprintf("  Command: `%s`\n", step.CommandStep.Command))
		} else {
			explanation.WriteString("  Type: Unknown Step\n")
		}
		if step.Output != nil {
			explanation.WriteString(fmt.Sprintf("  ==> $%s\n", *step.Output))
		}
		explanation.WriteString("\n")
	}
	return explanation.String(), nil
}

func (step AIStep) Run(ctx context.Context, cfg *Config, templateArgs *proto.TemplateArgs) (*string, error) {
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

	hasUserMessage := false
	if prompt.User != nil {
		tmpl := template.Must(template.New("user").Parse(*prompt.User))
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, templateArgs); err != nil {
			return nil, err
		}
		userPrompt := buf.String()
		messages = append(messages, openai.UserMessage(userPrompt))
		hasUserMessage = true
	} else {
		if userPrompt, ok := (*templateArgs)[PROMPT_VAR]; ok && userPrompt != "" {
			messages = append(messages, openai.UserMessage((*templateArgs)[PROMPT_VAR]))
			hasUserMessage = true
		}
		if input, ok := (*templateArgs)[INPUT_VAR]; ok && input != "" {
			messages = append(messages, openai.UserMessage((*templateArgs)[INPUT_VAR]))
			hasUserMessage = true
		}
	}

	if len(messages) > 0 && !hasUserMessage {
		return nil, fmt.Errorf(`No user message found in the prompt. Try providing a message by typing after the pattern name or piping into the command. For example:

  echo "Tell me a joke" | axon -p %s

or

  axon -p %s -- Tell me a joke`, step.Prompt, step.Prompt)
	}

	modelStr := selectModelForStep(cfg, step)

	clientOptions, err := cfg.GetClientOptions(modelStr)
	if err != nil {
		return nil, err
	}

	client := client.GetClient(*clientOptions)

	if !cfg.GetQuiet() {
		spinner := internal.NewSpinner()
		spinner.Start("Thinking...")
		defer spinner.Stop()
	}

	stream := client.Request(ctx, proto.Request{Messages: messages})

	completion, err := stream.Collect(
		func(chunk openai.ChatCompletionChunk) {
			// print(chunk.Choices[0].Delta.Content)
		},
	)
	if err != nil {
		return nil, err
	}

	return &completion.Choices[0].Message.Content, nil
}

func (step CommandStep) Run(ctx context.Context, cfg *Config, templateArgs *proto.TemplateArgs, pipeIn bool) (*string, error) {
	shell := utils.GetShell()

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

	var stdoutBuf bytes.Buffer

	if step.Tty {
		if ttyFile, err := os.OpenFile("/dev/tty", os.O_RDWR, 0); err == nil {
			defer ttyFile.Close()
			cmd.Stdout = ttyFile
		} else {
			cmd.Stdout = &stdoutBuf
		}
	} else {
		cmd.Stdout = &stdoutBuf
	}

	if cfg.GetQuiet() {
		cmd.Stderr = io.Discard
	} else {
		cmd.Stderr = os.Stderr
	}

	if stdin, ok := (*templateArgs)[PIPE_VAR]; ok && pipeIn {
		cmd.Stdin = bytes.NewBufferString(stdin)
	}

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("command failed: %w", err)
	}

	outputString := stdoutBuf.String()

	return &outputString, nil
}

// promptName should be with out the "@" prefix
func MakeSinglePromptPattern(promptName string) Pattern {
	promptName = strings.TrimPrefix(promptName, "@")
	promptSpecifier := "@" + promptName
	return Pattern{
		Name: promptSpecifier,
		Steps: []Step{
			{
				AIStep: &AIStep{Prompt: promptSpecifier},
			},
		},
	}
}

func selectModelForStep(cfg *Config, step AIStep) string {
	if cfg.OverrideModel != nil {
		return *cfg.OverrideModel
	}
	modelStr := utils.DefaultString(step.Model, *cfg.General.Model)
	return modelStr
}
