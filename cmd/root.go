package cmd

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/madmaxieee/axon/internal/config"
	"github.com/madmaxieee/axon/internal/utils"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type Flags struct {
	ConfigFilePath string
	Pattern        string
	Explain        bool
	ShowLast       bool
}

var flags Flags
var cfg *config.Config

var rootCmd = &cobra.Command{
	Use: "axon",
	// TODO: come up with a better description
	Short: "A CLI tool to automate tasks with LLMs",
	Long: `Axon is a command-line tool that leverages the power of LLMs to automate tasks.
It's designed to be a versatile and scriptable tool that can be easily integrated into your workflows.`,

	Run: func(cmd *cobra.Command, args []string) {
		if flags.ShowLast {
			lastOutputPath, err := GetLastOutputPath()
			if err != nil {
				utils.HandleError(err)
			}
			data, err := os.ReadFile(lastOutputPath)
			if err != nil {
				utils.HandleError(err)
			}
			_, err = os.Stdout.Write(data)
			return
		}

		stdin, err := ReadStdinIfPiped()
		if err != nil {
			utils.HandleError(err)
		}

		promptString := strings.Join(args, " ")
		prompt := utils.RemoveWhitespace(promptString)

		if stdin == nil && prompt == nil && flags.Pattern == "default" {
			println("No input provided. Use --help for usage information.")
			return
		}

		pattern := cfg.GetPatternByName(flags.Pattern)
		if pattern == nil {
			utils.HandleError(errors.New("pattern not found: " + flags.Pattern))
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

		output, err := pattern.Run(cmd.Context(), cfg, stdin, prompt)
		if err != nil {
			utils.HandleError(err)
		}

		// only print to stdout if it is not a tty
		if !term.IsTerminal(int(os.Stdout.Fd())) {
			_, err = io.WriteString(os.Stdout, output)
			if err != nil {
				utils.HandleError(err)
			}
		}

		lastOutputPath, err := GetLastOutputPath()
		if err != nil {
			utils.HandleError(err)
		}
		err = os.WriteFile(lastOutputPath, []byte(output), 0644)
		if err != nil {
			utils.HandleError(err)
		}
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
		filepath.Join(xdg.ConfigHome, "axon", "axon.toml"),
		"path to config file",
	)
	rootCmd.Flags().StringVarP(&flags.Pattern, "pattern", "p", "default", "pattern to use")
	rootCmd.Flags().BoolVarP(&flags.ShowLast, "show-last", "S", false, "show last output")
	rootCmd.Flags().BoolVarP(&flags.Explain, "explain", "e", false, "explain the chosen pattern and exit")

	if strings.HasPrefix(flags.ConfigFilePath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			utils.HandleError(err)
		}
		flags.ConfigFilePath = filepath.Join(homeDir, flags.ConfigFilePath[1:])
	}

	var err error
	cfg, err = config.EnsureConfig(&flags.ConfigFilePath)
	if err != nil {
		utils.HandleError(err)
	}

	_ = rootCmd.RegisterFlagCompletionFunc("pattern", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if cfg != nil {
			return cfg.GetAllPatternNames(), cobra.ShellCompDirectiveDefault
		}
		return []string{}, cobra.ShellCompDirectiveDefault
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

func GetLastOutputPath() (string, error) {
	return xdg.CacheFile("axon/last_output.txt")
}
