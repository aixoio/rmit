package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var GetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get configuration values",
	Long: `Get configuration values like API key, URL, and default model.

Examples:
  - ` + "`rmit get`" + ` – show all configuration values
  - ` + "`rmit get api_key`" + ` – show the current API key
  - ` + "`rmit get default_model`" + ` – show the configured model`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ks := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true).MarginRight(1)
		vs := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(false)
		vns := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true).Background(lipgloss.Color("1"))
		aks := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true).Background(lipgloss.Color("4"))
		es := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Bold(true)

		if len(args) == 0 {

			api_key := viper.GetString("api_key")
			if strings.TrimSpace(api_key) == "" {
				fmt.Println(ks.Render("api_key"), vns.Render("[NOT SET]"))
			} else {
				fmt.Println(ks.Render("api_key"), aks.Render("[SET]"))
			}

			fmt.Println(ks.Render("api_url"), vs.Render(viper.GetString("api_url")))
			fmt.Println(ks.Render("default_model"), vs.Render(viper.GetString("default_model")))
			fmt.Println(ks.Render("config_path"), vs.Render(viper.ConfigFileUsed()))

		} else {
			key := args[0]

			switch key {
			case "api_key":

				api_key := viper.GetString("api_key")
				if strings.TrimSpace(api_key) == "" {
					fmt.Println(ks.Render("api_key"), vns.Render("[NOT SET]"))
				} else {
					fmt.Println(ks.Render("api_key"), vs.Render(viper.GetString("api_key")))
				}

			case "api_url":
				fmt.Println(ks.Render("api_url"), vs.Render(viper.GetString("api_url")))
			case "default_model":
				fmt.Println(ks.Render("default_model"), vs.Render(viper.GetString("default_model")))
			default:
				log.Fatalf("%s %s. Valid keys are: api_key, api_url, default_model", es.Render("Unknown configuration key:"), key)
			}
		}

	},
}
