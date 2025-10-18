/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/madmaxieee/axon/internal/client"
	"github.com/madmaxieee/axon/internal/proto"
	"github.com/openai/openai-go/v3"
	"github.com/spf13/cobra"
)

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
		sysprompt := "Output the requested bible verse and no more explanation."

		question := "Genisisl 1:1-5"

		client := client.NewClient()

		stream := client.Request(cmd.Context(), proto.Request{
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(sysprompt),
				openai.UserMessage(question),
			},
		})

		completion, err := stream.Collect(
			func(chunk openai.ChatCompletionChunk) {
				print(chunk.Choices[0].Delta.Content)
			},
		)
		if err != nil {
			panic(err)
		}

		_, err = fmt.Print(completion.Choices[0].Message.Content)
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
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.axon.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
