package main

import (
	"bufio"
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

	"github.com/fatih/color"
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
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return "y", nil
	}
	return strings.ToLower(input), nil
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

	// Initialize colors
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	magenta := color.New(color.FgMagenta).SprintFunc()

	// Print header
	fmt.Printf("%s\n", blue("██████╗ ███╗   ███╗██╗████████╗"))
	fmt.Printf("%s\n", blue("██╔══██╗████╗ ████║██║╚══██╔══╝"))
	fmt.Printf("%s\n", blue("██████╔╝██╔████╔██║██║   ██║   "))
	fmt.Printf("%s\n", blue("██╔══██╗██║╚██╔╝██║██║   ██║   "))
	fmt.Printf("%s\n", blue("██║  ██║██║ ╚═╝ ██║██║   ██║   "))
	fmt.Printf("%s\n", blue("╚═╝  ╚═╝╚═╝     ╚═╝╚═╝   ╚═╝   "))
	fmt.Println()

	// Print version info
	fmt.Printf("%s %s\n", cyan("RMIT"), green("v1.0.0"))
	fmt.Printf("%s\n", yellow("AI-powered commit message generator"))
	fmt.Println(magenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
	fmt.Println()

	// Create root command
	rootCmd := &cobra.Command{
		Use:   "rmit",
		Short: "Generate git commit messages with AI",
		Long:  "rmit uses OpenRouter to generate descriptive git commit messages based on your changes",
		Run: func(cmd *cobra.Command, args []string) {
			// Load configuration
			config, err := loadConfig()
			if err != nil {
				log.Fatalf("%s %v", red("Error loading configuration:"), err)
			}

			// Get git diff
			diff, err := getGitDiff()
			if err != nil {
				log.Fatalf("%s %v", red("Error getting git diff:"), err)
			}

			// Print which model is being used
			modelToUse := model
			if model == "" {
				modelToUse = config.DefaultModel
			}

			fmt.Printf("\n%s\n", magenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
			fmt.Printf("%s %s\n", green("🤖 USING MODEL:"), cyan(modelToUse))
			fmt.Printf("%s\n", magenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))

			// Generate commit message
			fmt.Printf("\n%s\n", yellow("Generating commit message..."))
			message, err := generateCommitMessage(config, diff, model)
			if err != nil {
				log.Fatalf("%s %v", red("Error generating commit message:"), err)
			}

			// Output commit message with prominent formatting
			fmt.Printf("\n%s\n", magenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
			fmt.Printf("%s\n", blue("✨ GENERATED COMMIT MESSAGE:"))
			fmt.Printf("%s\n", magenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
			fmt.Printf("\n%s\n\n", cyan(message))
			fmt.Printf("%s\n", magenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))

			// Handle commit based on auto-commit flag or user confirmation
			if autoCommit {
				// Auto-commit mode - commit without confirmation
				if err := makeCommit(message); err != nil {
					log.Fatalf("%s %v", red("Error creating commit:"), err)
				}
				fmt.Printf("%s\n", green("✅ Commit created successfully"))
			} else {
				// Ask for confirmation with additional options
				fmt.Printf("\n%s\n", yellow("⚙️  OPTIONS:"))
				fmt.Printf("%s\n", magenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
				fmt.Printf("  %s - Create commit with this message\n", green("y/yes"))
				fmt.Printf("  %s - Cancel commit\n", red("n/no"))
				fmt.Printf("  %s - Generate more detailed message\n", blue("g"))
				fmt.Printf("  %s - Retry with new generation\n", blue("r"))
				fmt.Printf("  %s - Summarize message\n", blue("s"))
				fmt.Printf("  %s - Provide feedback for the message\n", blue("p"))
				fmt.Printf("%s\n", magenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))

				for {
					fmt.Print(yellow("Create commit with this message? [y/n/g/r/s/p]: "))

					response, err := readUserInput()
					if err != nil {
						log.Fatalf("%s %v", red("Error reading user input:"), err)
					}

					if response == "y" || response == "yes" {
						if err := makeCommit(message); err != nil {
							log.Fatalf("%s %v", red("Error creating commit:"), err)
						}
						fmt.Printf("%s\n", green("✅ Commit created successfully"))
						break
					} else if response == "n" || response == "no" {
						fmt.Printf("%s\n", yellow("⚠️ Commit canceled"))
						break
					} else if response == "g" {
						fmt.Printf("%s\n", blue("🔍 Generating a more detailed commit message..."))
						message, err = generateCommitMessage(config, diff+"\n\nPlease provide a more detailed commit message with additional context and explanations.", model)
						if err != nil {
							log.Fatalf("%s %v", red("Error generating detailed commit message:"), err)
						}
						fmt.Printf("\n%s\n", magenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
						fmt.Printf("%s\n", blue("✨ GENERATED DETAILED COMMIT MESSAGE:"))
						fmt.Printf("%s\n", magenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
						fmt.Printf("\n%s\n\n", cyan(message))
						fmt.Printf("%s\n", magenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
					} else if response == "r" {
						fmt.Printf("%s\n", blue("🔄 Retrying with a new generation..."))
						message, err = generateCommitMessage(config, diff, model)
						if err != nil {
							log.Fatalf("%s %v", red("Error regenerating commit message:"), err)
						}
						fmt.Printf("\n%s\n", magenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
						fmt.Printf("%s\n", blue("✨ REGENERATED COMMIT MESSAGE:"))
						fmt.Printf("%s\n", magenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
						fmt.Printf("\n%s\n\n", cyan(message))
						fmt.Printf("%s\n", magenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
					} else if response == "s" {
						fmt.Printf("%s\n", blue("📝 Summarizing the commit message..."))
						summary, err := generateCommitMessage(config, "Please summarize this commit message in 50 characters or less:\n\n"+message, model)
						if err != nil {
							log.Fatalf("%s %v", red("Error summarizing commit message:"), err)
						}
						message = summary
						fmt.Printf("\n%s\n", magenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
						fmt.Printf("%s\n", blue("✨ SUMMARIZED COMMIT MESSAGE:"))
						fmt.Printf("%s\n", magenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
						fmt.Printf("\n%s\n\n", cyan(message))
						fmt.Printf("%s\n", magenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
					} else if response == "p" {
						fmt.Printf("%s\n", blue("🔍 Enter your feedback for the commit message:"))
						fmt.Print("> ")

						// Read a single line of input
						reader := bufio.NewReader(os.Stdin)
						feedbackLine, err := reader.ReadString('\n')
						if err != nil {
							log.Fatalf("%s %v", red("Error reading feedback:"), err)
						}
						feedback := strings.TrimSpace(feedbackLine)

						fmt.Printf("%s\n", blue("🎯 Generating commit message based on your feedback..."))

						// Use the feedback directly in the prompt
						promptWithGuidance := "Based on this diff:\n\n" + diff + "\n\nAnd considering this feedback: " + feedback + "\n\nGenerate an appropriate commit message."
						message, err = generateCommitMessage(config, promptWithGuidance, model)
						if err != nil {
							log.Fatalf("%s %v", red("Error generating commit message with custom guidance:"), err)
						}

						fmt.Printf("\n%s\n", magenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
						fmt.Printf("%s\n", blue("✨ FEEDBACK-BASED COMMIT MESSAGE:"))
						fmt.Printf("%s\n", magenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
						fmt.Printf("\n%s\n\n", cyan(message))
						fmt.Printf("%s\n", magenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
					} else {
						fmt.Printf("%s\n", red("❌ Invalid option. Please choose y (yes), n (no), g (generate detailed), r (retry), s (shorter), or p (custom prompt)."))
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
					log.Fatalf("%s %v", red("Invalid API key:"), err)
				}
				config.APIKey = value
			case "api_url":
				if err := validateAPIURL(value); err != nil {
					log.Fatalf("%s %v", red("Invalid API URL:"), err)
				}
				config.APIURL = value
			case "default_model":
				config.DefaultModel = value
			default:
				log.Fatalf("%s %s. Valid keys are: api_key, api_url, default_model", red("Unknown configuration key:"), key)
			}

			// Save config
			if err := saveConfig(config); err != nil {
				log.Fatalf("%s %v", red("Error saving configuration:"), err)
			}

			fmt.Printf("%s %s = %s\n", green("✅ Configuration updated:"), blue(key), cyan(value))
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
				log.Fatalf("%s %v", red("Error loading configuration:"), err)
			}

			// If no key specified, show all (except sensitive data like API key)
			if len(args) == 0 {
				fmt.Printf("%s\n", blue("📋 Current configuration:"))
				fmt.Printf("%s\n", magenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
				if config.APIKey != "" {
					fmt.Printf("%s %s\n", green("api_key:"), blue("[SET]"))
				} else {
					fmt.Printf("%s %s\n", green("api_key:"), red("[NOT SET]"))
				}
				fmt.Printf("%s %s\n", green("api_url:"), blue(config.APIURL))
				fmt.Printf("%s %s\n", green("default_model:"), blue(config.DefaultModel))
				fmt.Printf("%s\n", magenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))

				// Show config file location
				configPath, _ := getConfigPath()
				fmt.Printf("\n%s %s\n", green("💾 Configuration stored at:"), blue(configPath))
				return
			}

			// Get specific key
			key := args[0]
			switch key {
			case "api_key":
				if config.APIKey != "" {
					fmt.Printf("%s\n", blue("[SET]"))
				} else {
					fmt.Printf("%s\n", red("[NOT SET]"))
				}
			case "api_url":
				fmt.Printf("%s\n", blue(config.APIURL))
			case "default_model":
				fmt.Printf("%s\n", blue(config.DefaultModel))
			default:
				log.Fatalf("%s %s. Valid keys are: api_key, api_url, default_model", red("Unknown configuration key:"), key)
			}
		},
	}

	// Add commands to root
	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(getCmd)

	// Add flags
	rootCmd.Flags().BoolVarP(&autoCommit, "commit", "c", false, "Automatically create commit with generated message")
	rootCmd.Flags().StringVarP(&model, "model", "m", "", "OpenRouter model to use for generation (overrides default_model from config)")

	// Execute command
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("%s\n", red(err))
		os.Exit(1)
	}
}
