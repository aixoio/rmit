package main

import (
	"context"
	"os"

	"github.com/aixoio/rmit/cmd"
	"github.com/charmbracelet/fang"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigName("rmit-v2-config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME")

	viper.SetDefault("api_key", "")
	viper.SetDefault("api_url", "https://openrouter.ai/api/v1")
	viper.SetDefault("default_model", "openai/gpt-4.1-nano")

	err := viper.ReadInConfig()
	if err != nil {
		if err := viper.SafeWriteConfig(); err != nil {
			panic(err)
		}
		err := viper.ReadInConfig()
		if err != nil {
			panic(err)
		}
	}

	if err := viper.WriteConfig(); err != nil {
		panic(err)
	}

	cmd.InitCommands()

	if err := fang.Execute(context.Background(), cmd.RootCmd); err != nil {
		os.Exit(1)
	}
}
