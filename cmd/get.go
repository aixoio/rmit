package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var GetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get configuration values",
	Long:  "Get configuration values like API key, URL, and default model",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			ks := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true).MarginRight(1)
			vs := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(false)
			vns := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true).Background(lipgloss.Color("1"))
			aks := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true).Background(lipgloss.Color("4"))

			api_key := viper.GetString("api_key")
			if strings.TrimSpace(api_key) == "" {
				fmt.Println(ks.Render("api_key"), vns.Render("[NOT SET]"))
			} else {
				fmt.Println(ks.Render("api_key"), aks.Render("[SET]"))
			}

			fmt.Println(ks.Render("api_url"), vs.Render(viper.GetString("api_url")))
			fmt.Println(ks.Render("default_model"), vs.Render(viper.GetString("default_model")))
			fmt.Println(ks.Render("config_path"), vs.Render(viper.ConfigFileUsed()))

		}
	},
}
