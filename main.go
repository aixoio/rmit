package main

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:   "rmit",
		Short: "Generate git commit messages with AI",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Hi")
		},
	}

	if err := fang.Execute(context.Background(), cmd); err != nil {
		os.Exit(1)
	}
}
