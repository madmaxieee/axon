/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
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

type Flags struct {
	ConfigFilePath string
	Pattern        string
	ShowLast       bool
}

var flags Flags

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "axon",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,

	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.EnsureConfig(&flags.ConfigFilePath)
		if err != nil {
			panic(err)
		}

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
		output, err := pattern.Run(cmd.Context(), cfg, stdin, prompt)
		if err != nil {
			panic(err)
		}

		if output == "" {
			println("<empty response>")
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
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
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
		filepath.Join(xdg.ConfigHome, "axon", "config.toml"),
		"config file (default is $XDG_CONFIG_HOME/axon/config.toml)",
	)
	rootCmd.Flags().StringVarP(&flags.Pattern, "pattern", "p", "default", "pattern to use")
	rootCmd.Flags().BoolVarP(&flags.ShowLast, "show-last", "S", false, "show last output")
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
