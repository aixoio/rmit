package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "rmit",
	Short: "Generate git commit messages with AI",
	Long:  "rmit uses OpenRouter to generate descriptive git commit messages based on your changes",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hi")
	},
}
