package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Configuration
type Config struct {
	APIKey       string `json:"api_key"`
	APIURL       string `json:"api_url"`
	DefaultModel string `json:"default_model"`
}

// Default configuration values
const (
	defaultAPIURL  = "https://openrouter.ai/api/v1/chat/completions"
	defaultModel   = "openai/gpt-3.5-turbo"
	configFileName = ".rmitconfig"
)

// getConfigPath returns the path to the configuration file
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, configFileName)

	return configPath, nil
}

// ensureConfigDir ensures the configuration directory exists (not needed for home directory)
func ensureConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return homeDir, nil
}

// loadConfig loads configuration from file or initializes defaults
func loadConfig() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	// Initialize default config
	config := &Config{
		APIURL:       defaultAPIURL,
		DefaultModel: defaultModel,
	}

	// Try to read API key from environment first
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey != "" {
		config.APIKey = apiKey
	}

	// Try to load config file
	data, err := os.ReadFile(configPath)
	if err == nil {
		// File exists, try to unmarshal
		var configMap map[string]string
		if err := json.Unmarshal(data, &configMap); err != nil {
			log.Printf("Warning: failed to parse config file (will use defaults): %v", err)
		} else {
			// Apply values from file
			if apiKey, ok := configMap["api_key"]; ok && apiKey != "" {
				config.APIKey = apiKey
			}
			if apiURL, ok := configMap["api_url"]; ok && apiURL != "" {
				config.APIURL = apiURL
			}
			if model, ok := configMap["default_model"]; ok && model != "" {
				config.DefaultModel = model
			}
		}
	} else if !os.IsNotExist(err) {
		// Error is not "file not found"
		log.Printf("Warning: failed to read config file (will use defaults): %v", err)
	}

	// Validate and apply defaults
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// saveConfig saves the configuration to disk
func saveConfig(config *Config) error {
	// Ensure config directory exists
	_, err := ensureConfigDir()
	if err != nil {
		return err
	}

	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	// Validate config before saving
	if config.APIURL == "" {
		config.APIURL = defaultAPIURL
	}
	if config.DefaultModel == "" {
		config.DefaultModel = defaultModel
	}

	// Create a clean map for marshaling
	configMap := map[string]string{
		"api_key":       config.APIKey,
		"api_url":       config.APIURL,
		"default_model": config.DefaultModel,
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(configMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// validateConfig checks if the configuration is valid
func validateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("configuration is nil")
	}

	// Set defaults for missing values
	if config.APIURL == "" {
		config.APIURL = defaultAPIURL
	}
	if config.DefaultModel == "" {
		config.DefaultModel = defaultModel
	}

	return nil
}
