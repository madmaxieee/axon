package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/madmaxieee/axon/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// TODO: add show pattern flag to print the details of the pattern being used
type Flags struct {
	ConfigFilePath string
	Pattern        string
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

	// TODO: handle errors more gracefully
	Run: func(cmd *cobra.Command, args []string) {
		if flags.ShowLast {
			lastOutputPath, err := GetLastOutputPath()
			if err != nil {
				panic(err)
			}
			data, err := os.ReadFile(lastOutputPath)
			if err != nil {
				panic(err)
			}
			_, err = os.Stdout.Write(data)
			return
		}

		stdin, err := ReadStdinIfPiped()
		if err != nil {
			panic(err)
		}

		promptString := strings.Join(args, " ")
		prompt := removeWhitespace(promptString)

		if stdin == nil && prompt == nil && flags.Pattern == "default" {
			println("No input provided. Use --help for usage information.")
			return
		}

		pattern := cfg.GetPatternByName(flags.Pattern)
		if pattern == nil {
			panic("pattern not found: " + flags.Pattern)
		}
		output, err := pattern.Run(cmd.Context(), cfg, stdin, prompt)
		if err != nil {
			panic(err)
		}

		// only print to stdout if it is not a tty
		if !term.IsTerminal(int(os.Stdout.Fd())) {
			_, err = io.WriteString(os.Stdout, output)
			if err != nil {
				panic(err)
			}
		}

		lastOutputPath, err := GetLastOutputPath()
		if err != nil {
			panic(err)
		}
		err = os.WriteFile(lastOutputPath, []byte(output), 0644)
		if err != nil {
			panic(err)
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

	if strings.HasPrefix(flags.ConfigFilePath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		flags.ConfigFilePath = filepath.Join(homeDir, flags.ConfigFilePath[1:])
	}

	var err error
	cfg, err = config.EnsureConfig(&flags.ConfigFilePath)
	if err != nil {
		panic(err)
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
		return removeWhitespace(s), nil
	}

	return nil, nil
}

func GetLastOutputPath() (string, error) {
	return xdg.CacheFile("axon/last_output.txt")
}

func removeWhitespace(str string) *string {
	if strings.TrimSpace(str) == "" {
		return nil
	}
	return &str
}
