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
	"github.com/madmaxieee/axon/internal/temp"
	"github.com/madmaxieee/axon/internal/utils"
	"github.com/openai/openai-go/v3"
)

const (
	INPUT_VAR  = "INPUT"
	PROMPT_VAR = "PROMPT"
)

var PIPE_VAR = fmt.Sprintf("PIPE_%s", utils.Nonce())

func (p *Pattern) Run(ctx context.Context, cfg *Config, stdin *string, prompt *string) (string, error) {
	variables := make(map[string]string)
	if stdin != nil {
		variables[INPUT_VAR] = *stdin
		variables[PIPE_VAR] = *stdin
	} else {
		variables[INPUT_VAR] = ""
		variables[PIPE_VAR] = ""
	}
	if prompt != nil {
		variables[PROMPT_VAR] = *prompt
	} else {
		variables[PROMPT_VAR] = ""
	}

	tempManager := temp.NewManager("")
	defer tempManager.Cleanup()

	for _, step := range p.Steps {
		var output *string
		var err error
		if step.AIStep != nil {
			output, err = step.AIStep.Run(ctx, cfg, &variables)
			if err != nil {
				err = fmt.Errorf(`AI step with prompt "%s" failed: %w`, step.AIStep.Prompt, err)
			}
		} else if step.CommandStep != nil {
			output, err = step.CommandStep.Run(ctx, cfg, &variables)
			if err != nil {
				err = fmt.Errorf(`Command step "%s" failed: %w`, step.CommandStep.Command, err)
			}
		} else {
			return "", fmt.Errorf("step has neither AIStep nor CommandStep defined")
		}
		if err != nil {
			return "", err
		}

		if output != nil {
			if err := storeStepOutput(step, *output, variables, tempManager); err != nil {
				return "", fmt.Errorf("failed to store step output: %w", err)
			}
		}
	}

	return variables[PIPE_VAR], nil
}

func storeStepOutput(step Step, content string, variables map[string]string, tempManager *temp.Manager) error {
	if step.Output == nil {
		variables[PIPE_VAR] = content
		return nil
	}

	output := *step.Output
	var key string
	var appendMode bool

	if k, ok := strings.CutPrefix(output, ">>"); ok {
		key = k
		appendMode = true
	} else if k, ok := strings.CutPrefix(output, ">"); ok {
		key = k
		appendMode = false
	} else {
		variables[output] = content
		return nil
	}

	tempFile, err := tempManager.GetTempFile(key)
	if err != nil {
		return fmt.Errorf("failed to get temp file for output %s: %w", output, err)
	}

	if appendMode {
		err = tempFile.AppendText(content)
	} else {
		err = tempFile.WriteText(content)
	}

	if err != nil {
		op := "write"
		if appendMode {
			op = "append"
		}
		return fmt.Errorf("failed to %s to temp file for output %s: %w", op, output, err)
	}

	variables[key] = tempFile.Path()
	return nil
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
			if promptName, ok := strings.CutPrefix(step.AIStep.Prompt, "@"); ok {
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
			if step.CommandStep.Stdin != nil {
				explanation.WriteString(fmt.Sprintf("  Stdin: `%s`\n", *step.CommandStep.Stdin))
			}
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

func (step AIStep) Run(ctx context.Context, cfg *Config, variables *map[string]string) (*string, error) {
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
		tmpl, err := template.New("system").Option("missingkey=error").Parse(*prompt.System)
		if err != nil {
			return nil, fmt.Errorf("failed to parse system prompt: %w", err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, variables); err != nil {
			return nil, err
		}
		systemPrompt := buf.String()
		messages = append(messages, openai.SystemMessage(systemPrompt))
	}

	hasUserMessage := false
	if prompt.User != nil {
		tmpl, err := template.New("user").Option("missingkey=error").Parse(*prompt.User)
		if err != nil {
			return nil, fmt.Errorf("failed to parse user prompt: %w", err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, variables); err != nil {
			return nil, err
		}
		userPrompt := buf.String()
		messages = append(messages, openai.UserMessage(userPrompt))
		hasUserMessage = true
	} else {
		// provide context before user prompt
		if input, ok := (*variables)[INPUT_VAR]; ok && input != "" {
			messages = append(messages, openai.UserMessage((*variables)[INPUT_VAR]))
			hasUserMessage = true
		}
		if userPrompt, ok := (*variables)[PROMPT_VAR]; ok && userPrompt != "" {
			messages = append(messages, openai.UserMessage((*variables)[PROMPT_VAR]))
			hasUserMessage = true
		}
	}

	if len(messages) > 0 && !hasUserMessage {
		return nil, fmt.Errorf(`No user message found in the prompt. Try providing a message by typing after the pattern name or piping into the command. For example:

  echo "Tell me a joke" | axon %s

or

  axon %s -- Tell me a joke`, step.Prompt, step.Prompt)
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

func (step CommandStep) Run(ctx context.Context, cfg *Config, variables *map[string]string) (*string, error) {
	shell := utils.GetShell()

	shellQuotedArgs := make(map[string]string)
	for k, v := range *variables {
		shellQuotedArgs[k] = utils.ShellQuote(v)
	}

	trimmedCmd := strings.TrimSpace(step.Command)
	var commandTemplate string
	var pipeIn bool
	if strings.HasPrefix(trimmedCmd, "|") {
		commandTemplate = trimmedCmd[1:]
		pipeIn = true
	} else {
		commandTemplate = trimmedCmd
		pipeIn = false
	}
	commandTemplate = strings.TrimSpace(commandTemplate)

	if step.Stdin != nil && pipeIn {
		return nil, fmt.Errorf("stdin configuration and pipe command (|) are mutually exclusive")
	}

	tmpl, err := template.New("command").Option("missingkey=error").Parse(commandTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse command: %w", err)
	}
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

	if step.Stdin != nil {
		tmpl, err := template.New("stdin").Parse(*step.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to parse stdin: %w", err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, variables); err != nil {
			return nil, fmt.Errorf("failed to execute stdin template: %w", err)
		}
		cmd.Stdin = &buf
	} else if stdin, ok := (*variables)[PIPE_VAR]; ok && pipeIn {
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
	var modelStr string
	if cfg.OverrideModel != nil {
		modelStr = *cfg.OverrideModel
	} else {
		modelStr = utils.DefaultString(step.Model, *cfg.General.Model)
	}
	if aliasTarget, ok := cfg.General.ModelAliases[modelStr]; ok {
		modelStr = aliasTarget
	}
	return modelStr
}
