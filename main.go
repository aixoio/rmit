package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
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

// OpenRouter request structure
type OpenRouterRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// Message structure for OpenRouter API
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenRouter response structure
type OpenRouterResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// No longer needed as we have moved this to the default configuration values

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

// getGitDiff gets the current changes in the git repository
func getGitDiff() (string, error) {
	// Check if git is installed
	_, err := exec.LookPath("git")
	if err != nil {
		return "", fmt.Errorf("git is not installed or not in PATH")
	}

	// Check if current directory is a git repository
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("current directory is not a git repository")
	}

	// Get staged changes
	stagedCmd := exec.Command("git", "diff", "--staged")
	stagedOutput, err := stagedCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get staged changes: %w", err)
	}

	// Get unstaged changes if no staged changes
	if len(stagedOutput) == 0 {
		unstagedCmd := exec.Command("git", "diff")
		unstagedOutput, err := unstagedCmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get unstaged changes: %w", err)
		}

		if len(unstagedOutput) == 0 {
			return "", fmt.Errorf("no changes detected in the repository")
		}

		return string(unstagedOutput), nil
	}

	return string(stagedOutput), nil
}

// trackCodeChanges analyzes a message to identify and structure code changes
func trackCodeChanges(message string) (map[string]string, error) {
	changes := make(map[string]string)

	// Split message into lines
	lines := strings.Split(message, "\n")

	// Track current file being modified
	var currentFile string

	for _, line := range lines {
		// Detect file changes
		if strings.HasPrefix(line, "+++ b/") || strings.HasPrefix(line, "--- a/") {
			filePath := strings.TrimPrefix(line, "+++ b/")
			filePath = strings.TrimPrefix(filePath, "--- a/")
			currentFile = filePath
			continue
		}

		// Track additions and deletions
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			if currentFile != "" {
				changes[currentFile] += line + "\n"
			}
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			if currentFile != "" {
				changes[currentFile] += line + "\n"
			}
		}
	}

	return changes, nil
}

// getChangedFiles gets the names of files that have been changed
func getChangedFiles() ([]string, error) {
	// Check if git is installed
	_, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("git is not installed or not in PATH")
	}

	// Get staged files
	stagedCmd := exec.Command("git", "diff", "--staged", "--name-only")
	stagedOutput, err := stagedCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get staged files: %w", err)
	}

	// Get unstaged files if no staged files
	if len(stagedOutput) == 0 {
		unstagedCmd := exec.Command("git", "diff", "--name-only")
		unstagedOutput, err := unstagedCmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to get unstaged files: %w", err)
		}

		if len(unstagedOutput) == 0 {
			return nil, fmt.Errorf("no changed files detected in the repository")
		}

		return strings.Split(strings.TrimSpace(string(unstagedOutput)), "\n"), nil
	}

	return strings.Split(strings.TrimSpace(string(stagedOutput)), "\n"), nil
}

// getProjectInfo gets information about the project
func getProjectInfo() (string, error) {
	// Try to determine the project type based on files
	files, err := filepath.Glob("*")
	if err != nil {
		return "", fmt.Errorf("failed to list files: %w", err)
	}

	var projectInfo strings.Builder
	projectInfo.WriteString("Project files include: ")

	// Look for specific project indicators
	hasGoMod := false
	hasPackageJSON := false
	hasPomXML := false
	hasCMake := false
	hasPyProject := false

	for _, file := range files {
		switch file {
		case "go.mod":
			hasGoMod = true
		case "package.json":
			hasPackageJSON = true
		case "pom.xml":
			hasPomXML = true
		case "CMakeLists.txt":
			hasCMake = true
		case "pyproject.toml":
			hasPyProject = true
		}
	}

	if hasGoMod {
		projectInfo.WriteString("Go project. ")
	}
	if hasPackageJSON {
		projectInfo.WriteString("JavaScript/Node.js project. ")
	}
	if hasPomXML {
		projectInfo.WriteString("Java/Maven project. ")
	}
	if hasCMake {
		projectInfo.WriteString("C/C++ project with CMake. ")
	}
	if hasPyProject {
		projectInfo.WriteString("Python project. ")
	}

	return projectInfo.String(), nil
}

// readUserInput reads a single character from the user
func readUserInput() (string, error) {
	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		return "", err
	}
	return strings.ToLower(strings.TrimSpace(input)), nil
}

// generateCommitMessage uses OpenRouter to generate a commit message based on git diff and project information
func generateCommitMessage(config *Config, diff string, model string) (string, error) {
	if model == "" {
		model = config.DefaultModel
	}

	// Get changed files for more context
	changedFiles, err := getChangedFiles()
	if err != nil {
		// Non-fatal error, we can continue without this info
		log.Printf("Warning: couldn't get changed files: %v", err)
	}

	// Get project information for more context
	projectInfo, err := getProjectInfo()
	if err != nil {
		// Non-fatal error, we can continue without this info
		log.Printf("Warning: couldn't get project info: %v", err)
	}

	// Build file list string
	var fileListStr string
	if len(changedFiles) > 0 {
		fileListStr = fmt.Sprintf("Changed files: %s\n\n", strings.Join(changedFiles, ", "))
	}

	// Prepare the prompt with more context
	prompt := "Generate a concise and descriptive git commit message based on the following changes. " +
		"Follow the conventional commit format (e.g., feat:, fix:, docs:, style:, refactor:, test:, chore:). " +
		"Only respond with the commit message, nothing else.\n\n"

	if projectInfo != "" {
		prompt += "Project information: " + projectInfo + "\n\n"
	}

	prompt += fileListStr + "Changes:\n" + diff

	// Create request body
	requestBody := OpenRouterRequest{
		Model: model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request body: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", config.APIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("HTTP-Referer", "https://github.com/aixoio/rmit")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: %s (status code: %d)", string(body), resp.StatusCode)
	}

	// Parse response
	var openRouterResp OpenRouterResponse
	if err := json.Unmarshal(body, &openRouterResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(openRouterResp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI model")
	}

	return strings.TrimSpace(openRouterResp.Choices[0].Message.Content), nil
}

// makeCommit creates a git commit with the provided message
func makeCommit(message string) error {
	// Stage all changes
	addCmd := exec.Command("git", "add", ".")
	addCmd.Stdout = os.Stdout
	addCmd.Stderr = os.Stderr
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Create commit
	commitCmd := exec.Command("git", "commit", "-m", message)
	commitCmd.Stdout = os.Stdout
	commitCmd.Stderr = os.Stderr
	return commitCmd.Run()
}

// validateAPIKey checks if the API key is valid
func validateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}
	return nil
}

// validateAPIURL checks if the API URL is valid
func validateAPIURL(url string) error {
	if url == "" {
		return fmt.Errorf("API URL cannot be empty")
	}
	return nil
}

func main() {
	var (
		autoCommit bool
		model      string
	)

	// Create root command
	rootCmd := &cobra.Command{
		Use:   "rmit",
		Short: "Generate git commit messages with AI",
		Long:  "rmit uses OpenRouter to generate descriptive git commit messages based on your changes",
		Run: func(cmd *cobra.Command, args []string) {
			// Load configuration
			config, err := loadConfig()
			if err != nil {
				log.Fatalf("Error loading configuration: %v", err)
			}

			// Get git diff
			diff, err := getGitDiff()
			if err != nil {
				log.Fatalf("Error getting git diff: %v", err)
			}

			// Generate commit message
			message, err := generateCommitMessage(config, diff, model)
			if err != nil {
				log.Fatalf("Error generating commit message: %v", err)
			}

			// Output commit message
			fmt.Println("Generated commit message:")
			fmt.Println(message)

			// Handle commit based on auto-commit flag or user confirmation
			if autoCommit {
				// Auto-commit mode - commit without confirmation
				if err := makeCommit(message); err != nil {
					log.Fatalf("Error creating commit: %v", err)
				}
				fmt.Println("Commit created successfully")
			} else {
				// Ask for confirmation with additional options
				for {
					fmt.Print("Create commit with this message? (y/n/g/r): ")

					response, err := readUserInput()
					if err != nil {
						log.Fatalf("Error reading user input: %v", err)
					}

					if response == "y" || response == "yes" {
						if err := makeCommit(message); err != nil {
							log.Fatalf("Error creating commit: %v", err)
						}
						fmt.Println("Commit created successfully")
						break
					} else if response == "n" || response == "no" {
						fmt.Println("Commit canceled")
						break
					} else if response == "g" {
						fmt.Println("Generating a more detailed commit message...")
						// Add more context to the prompt for a more detailed message
						message, err = generateCommitMessage(config, diff+"\n\nPlease provide a more detailed commit message with additional context and explanations.", model)
						if err != nil {
							log.Fatalf("Error generating detailed commit message: %v", err)
						}
						fmt.Println("Generated detailed commit message:")
						fmt.Println(message)
					} else if response == "r" {
						fmt.Println("Retrying with a new generation...")
						message, err = generateCommitMessage(config, diff, model)
						if err != nil {
							log.Fatalf("Error regenerating commit message: %v", err)
						}
						fmt.Println("Regenerated commit message:")
						fmt.Println(message)
					} else {
						fmt.Println("Invalid option. Please choose y (yes), n (no), g (generate detailed), or r (retry).")
					}
				}
			}
		},
	}

	// Create set command
	setCmd := &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set configuration values",
		Long:  "Set configuration values like API key, URL, and default model",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			key := args[0]
			value := args[1]

			// Load current config
			config, err := loadConfig()
			if err != nil {
				config = &Config{
					APIURL:       defaultAPIURL,
					DefaultModel: defaultModel,
				}
			}

			// Update based on key
			switch key {
			case "api_key":
				if err := validateAPIKey(value); err != nil {
					log.Fatalf("Invalid API key: %v", err)
				}
				config.APIKey = value
			case "api_url":
				if err := validateAPIURL(value); err != nil {
					log.Fatalf("Invalid API URL: %v", err)
				}
				config.APIURL = value
			case "default_model":
				config.DefaultModel = value
			default:
				log.Fatalf("Unknown configuration key: %s. Valid keys are: api_key, api_url, default_model", key)
			}

			// Save config
			if err := saveConfig(config); err != nil {
				log.Fatalf("Error saving configuration: %v", err)
			}

			fmt.Printf("Configuration updated: %s = %s\n", key, value)
		},
	}

	// Create get command
	getCmd := &cobra.Command{
		Use:   "get [key]",
		Short: "Get configuration values",
		Long:  "Get configuration values like API key, URL, and default model",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Load config
			config, err := loadConfig()
			if err != nil {
				log.Fatalf("Error loading configuration: %v", err)
			}

			// If no key specified, show all (except sensitive data like API key)
			if len(args) == 0 {
				fmt.Println("Current configuration:")
				if config.APIKey != "" {
					fmt.Println("api_key: [SET]")
				} else {
					fmt.Println("api_key: [NOT SET]")
				}
				fmt.Printf("api_url: %s\n", config.APIURL)
				fmt.Printf("default_model: %s\n", config.DefaultModel)

				// Show config file location
				configPath, _ := getConfigPath()
				fmt.Printf("\nConfiguration stored at: %s\n", configPath)
				return
			}

			// Get specific key
			key := args[0]
			switch key {
			case "api_key":
				if config.APIKey != "" {
					fmt.Println("[SET]")
				} else {
					fmt.Println("[NOT SET]")
				}
			case "api_url":
				fmt.Println(config.APIURL)
			case "default_model":
				fmt.Println(config.DefaultModel)
			default:
				log.Fatalf("Unknown configuration key: %s. Valid keys are: api_key, api_url, default_model", key)
			}
		},
	}

	// Add commands to root
	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(getCmd)

	// Add flags
	rootCmd.Flags().BoolVarP(&autoCommit, "commit", "c", false, "Automatically create commit with generated message")
	rootCmd.Flags().StringVarP(&model, "model", "m", defaultModel, "OpenRouter model to use for generation")

	// Execute command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
