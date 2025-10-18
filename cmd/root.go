/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"io"
	"os"
	"strings"

	"github.com/madmaxieee/axon/internal/config"
	"github.com/spf13/cobra"
)

type Flags struct {
	ConfigFilePath string
	Pattern        string
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

		stdin, err := ReadStdinIfPiped()
		if err != nil {
			panic(err)
		}

		promptString := strings.Join(args, " ")
		prompt := &promptString
		if strings.TrimSpace(promptString) == "" {
			prompt = nil
		}

		pattern := cfg.GetPatternByName(flags.Pattern)
		output, err := pattern.Run(cmd.Context(), cfg, stdin, prompt)
		if err != nil {
			panic(err)
		}

		_, err = io.WriteString(os.Stdout, output)
		if err != nil {
			panic(err)
		}
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
	// TODO: populate default config path
	rootCmd.PersistentFlags().StringVarP(&flags.ConfigFilePath, "config", "c", "", "config file (default is $XDG_CONFIG_HOME/axon/config.toml)")
	rootCmd.Flags().StringVarP(&flags.Pattern, "pattern", "p", "", "pattern to use")
}

func ReadStdinIfPiped() (*string, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}

	// Check if stdin is being piped (non-terminal)
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, err
		}
		s := string(data)
		return &s, nil
	}

	// Nothing piped in
	return nil, nil
}
