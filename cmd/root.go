package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "rmit",
	Short: "Generate git commit messages with AI",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hi")
	},
}
