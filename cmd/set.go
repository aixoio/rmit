package cmd

import (
	"fmt"
	"log"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var setCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set configuration values",
	Long:  "Set configuration values like API key, URL, and default model",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		// Define styles (consistent with get.go)
		ks := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true).MarginRight(1)
		ns := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
		es := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Bold(true)

		key := args[0]
		value := args[1]

		// Validate configuration key
		validKeys := map[string]bool{"api_key": true, "api_url": true, "default_model": true}
		if !validKeys[key] {
			log.Fatalf("%s %s. Valid keys are: api_key, api_url, default_model",
				es.Render("Invalid configuration key:"), key)
		}

		viper.Set(key, value)
		if err := viper.WriteConfig(); err != nil {
			log.Fatalf("%s Failed to write config: %v", es.Render("Error:"), err)
		}

		fmt.Printf("%s %s ➜ %s\n",
			ks.Render(key),
			ns.Render("successfully set to"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Render(value),
		)
	},
}
