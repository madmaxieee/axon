package cmd

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/madmaxieee/axon/internal/cache"
	"github.com/madmaxieee/axon/internal/config"
	"github.com/madmaxieee/axon/internal/proto"
	"github.com/madmaxieee/axon/internal/utils"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var flags proto.Flags
var cfg *config.Config

var rootCmd = &cobra.Command{
	Use: "axon",
	// TODO: come up with a better description
	Short: "A CLI tool to automate tasks with LLMs",
	Long: `Axon is a command-line tool that leverages the power of LLMs to automate tasks.
It's designed to be a versatile and scriptable tool that can be easily integrated into your workflows.`,

	Run: func(cmd *cobra.Command, args []string) {
		var lastRunData *cache.RunData
		var err error

		if flags.ShowLast {
			lastOutput, err := cache.GetLastOutput()
			if err != nil {
				utils.HandleError(err)
			}
			_, err = os.Stdout.Write([]byte(lastOutput))
			if err != nil {
				utils.HandleError(err)
			}
			return
		}

		if flags.Replay {
			lastRunData, err = cache.GetLastRunData()
			if err != nil {
				utils.HandleError(err)
			}
			err = cfg.Merge(config.GetOverrideConfig(lastRunData.Flags))
			if err != nil {
				utils.HandleError(err)
			}
		} else {
			err = cfg.Merge(config.GetOverrideConfig(flags))
			if err != nil {
				utils.HandleError(err)
			}
		}

		var pattern *config.Pattern
		var stdin *string
		var userExtraPrompt *string

		if flags.Replay {
			stdin = utils.RemoveWhitespace(lastRunData.Input)
			userExtraPrompt = utils.RemoveWhitespace(lastRunData.Prompt)
			pattern = &lastRunData.Pattern
		} else {
			stdin, err = ReadStdinIfPiped()
			if err != nil {
				utils.HandleError(err)
			}
			userExtraPrompt = utils.RemoveWhitespace(strings.Join(args, " "))
			pattern = cfg.GetPatternByName(flags.Pattern)
			_ = cache.SaveRunData(&cache.RunData{
				Pattern: *pattern,
				Flags:   flags,
				Input:   utils.DerefString(stdin),
				Prompt:  utils.DerefString(userExtraPrompt),
			})
		}

		if pattern == nil {
			utils.HandleError(errors.New("pattern not found: " + flags.Pattern))
		}

		if stdin == nil && userExtraPrompt == nil && flags.Pattern == "default" {
			println("No input provided. Use --help for usage information.")
			return
		}

		if flags.Explain {
			explanation, err := pattern.Explain(cmd.Context(), cfg)
			if err != nil {
				utils.HandleError(err)
			}
			_, err = io.WriteString(os.Stdout, explanation)
			if err != nil {
				utils.HandleError(err)
			}
			return
		}

		output, err := pattern.Run(cmd.Context(), cfg, stdin, userExtraPrompt)
		if err != nil {
			utils.HandleError(err)
		}

		if !cfg.GetQuiet() {
			println()
		}

		_, err = io.WriteString(os.Stdout, output)
		if err != nil {
			utils.HandleError(err)
		}

		_ = cache.SaveOutput(output)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(
		&flags.ConfigFilePath,
		"config",
		"c",
		filepath.Join(config.GetConfigHome(), "axon.toml"),
		"path to config file",
	)
	rootCmd.Flags().StringVarP(&flags.Pattern, "pattern", "p", "default", "pattern to use")
	rootCmd.Flags().BoolVarP(&flags.ShowLast, "show-last", "S", false, "show last output")
	rootCmd.Flags().BoolVarP(&flags.Replay, "replay", "R", false, "replay the last run with the same inputs and pattern")
	rootCmd.Flags().BoolVarP(&flags.Explain, "explain", "e", false, "explain the chosen pattern and exit")
	rootCmd.Flags().StringVarP(&flags.Model, "model", "m", "", "override the model for all AI steps")
	rootCmd.Flags().BoolVarP(&flags.Quiet, "quiet", "q", false, "suppress non-essential output")

	if strings.HasPrefix(flags.ConfigFilePath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			utils.HandleError(err)
		}
		flags.ConfigFilePath = filepath.Join(homeDir, flags.ConfigFilePath[1:])
	}

	// we have to populate cfg here to use it in the completion function
	var err error
	cfg, err = config.EnsureConfig(&flags.ConfigFilePath)
	if err != nil {
		utils.HandleError(err)
	}

	_ = rootCmd.RegisterFlagCompletionFunc("pattern", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if cfg != nil {
			var results []string
			if strings.HasPrefix(toComplete, "@") {
				for _, promptName := range cfg.GetAllPromptNames() {
					results = append(results, "@"+promptName)
				}
			} else {
				results = cfg.GetAllPatternNames()
			}
			return results, cobra.ShellCompDirectiveNoFileComp
		}
		return []string{}, cobra.ShellCompDirectiveError
	})
}

func ReadStdinIfPiped() (*string, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) && (stat.Mode()&os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, err
		}
		s := string(data)
		return utils.RemoveWhitespace(s), nil
	}

	return nil, nil
}
