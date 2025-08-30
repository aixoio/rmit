package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// runGitCommand executes a git command and returns combined stdout/stderr output
func runGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git command failed (%s): %w\nOutput: %s", strings.Join(args, " "), err, string(output))
	}
	return string(output), nil
}

var commitAuto bool

func init() {
	RootCmd.Flags().BoolVarP(&commitAuto, "commit", "c", false, "Auto-commit the generated message")
}

func getGitDiff() (string, error) {
	// Check if git is installed
	_, err := exec.LookPath("git")
	if err != nil {
		return "", fmt.Errorf("git is not installed or not in PATH")
	}

	// Check if current directory is a git repository
	_, err = runGitCommand("rev-parse", "--is-inside-work-tree")
	if err != nil {
		return "", fmt.Errorf("current directory is not a git repository: %w", err)
	}

	// Get staged changes
	stagedOutput, err := runGitCommand("diff", "--staged")
	if err != nil {
		return "", fmt.Errorf("failed to get staged changes: %w", err)
	}

	// Get unstaged changes if no staged changes
	if len(stagedOutput) == 0 {
		unstagedOutput, err := runGitCommand("diff")
		if err != nil {
			return "", fmt.Errorf("failed to get unstaged changes: %w", err)
		}

		if len(unstagedOutput) == 0 {
			return "", fmt.Errorf("no changes detected in the repository")
		}

		return unstagedOutput, nil
	}

	return stagedOutput, nil
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

// getCurrentBranch gets the current git branch name
func getCurrentBranch() (string, error) {
	// Check if git is installed
	_, err := exec.LookPath("git")
	if err != nil {
		return "", fmt.Errorf("git is not installed or not in PATH")
	}

	// Get current branch
	branchOutput, err := runGitCommand("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(branchOutput), nil
}

// getChangedFiles gets the names of files that have been changed
func getChangedFiles() ([]string, error) {
	// Check if git is installed
	_, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("git is not installed or not in PATH")
	}

	// Get staged files
	stagedOutput, err := runGitCommand("diff", "--staged", "--name-only")
	if err != nil {
		return nil, fmt.Errorf("failed to get staged files: %w", err)
	}

	// Get unstaged files if no staged files
	if len(stagedOutput) == 0 {
		unstagedOutput, err := runGitCommand("diff", "--name-only")
		if err != nil {
			return nil, fmt.Errorf("failed to get unstaged files: %w", err)
		}

		if len(unstagedOutput) == 0 {
			return nil, fmt.Errorf("no changed files detected in the repository")
		}

		return strings.Split(strings.TrimSpace(unstagedOutput), "\n"), nil
	}

	return strings.Split(strings.TrimSpace(stagedOutput), "\n"), nil
}

func getProjectInfo() (string, error) {
	var projectInfo strings.Builder
	projectInfo.WriteString("Project context: ")

	// Look for specific project indicators in root and common subdirectories
	hasGoMod := false
	hasPackageJSON := false
	hasPomXML := false
	hasCMake := false
	hasPyProject := false
	hasGradle := false
	hasUV := false
	hasBun := false
	hasMakefile := false
	hasCargo := false
	hasComposer := false

	// Framework-specific indicators
	hasNextJS := false
	hasReact := false
	hasVue := false
	hasAngular := false
	hasDjango := false
	hasFlask := false
	hasFastAPI := false
	hasDocker := false
	hasKubernetes := false

	// Scan root directory
	files, err := filepath.Glob("*")
	if err != nil {
		return "", fmt.Errorf("failed to list files: %w", err)
	}

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
		case "build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts":
			hasGradle = true
		case "uv.lock", "requirements.txt", "setup.py":
			hasUV = true
		case "bun.lockb", "bunfig.toml":
			hasBun = true
		case "Makefile":
			hasMakefile = true
		case "Cargo.toml":
			hasCargo = true
		case "composer.json":
			hasComposer = true
		case "Dockerfile", "docker-compose.yml", "docker-compose.yaml":
			hasDocker = true
		case "k8s", "kubernetes":
			hasKubernetes = true
		}
	}

	// Scan common subdirectories for framework files
	subdirs := []string{"frontend", "backend", "client", "server", "src", "web", "api"}
	for _, subdir := range subdirs {
		if _, err := os.Stat(subdir); err == nil {
			subFiles, err := filepath.Glob(filepath.Join(subdir, "*"))
			if err == nil {
				for _, file := range subFiles {
					base := filepath.Base(file)
					switch base {
					case "next.config.js", "next.config.mjs", "next.config.ts":
						hasNextJS = true
					case "vue.config.js", "nuxt.config.js", "nuxt.config.ts":
						hasVue = true
					case "angular.json":
						hasAngular = true
					case "manage.py":
						hasDjango = true
					case "app.py", "main.py":
						// Check if it's Flask or FastAPI by reading content
						if content, err := os.ReadFile(file); err == nil {
							contentStr := string(content)
							if strings.Contains(contentStr, "from flask import") || strings.Contains(contentStr, "import flask") {
								hasFlask = true
							}
							if strings.Contains(contentStr, "from fastapi import") || strings.Contains(contentStr, "import fastapi") {
								hasFastAPI = true
							}
						}
					}
				}
			}
		}
	}

	// Check package.json for React if not already detected
	if hasPackageJSON && !hasReact {
		if content, err := os.ReadFile("package.json"); err == nil {
			contentStr := string(content)
			if strings.Contains(contentStr, "\"react\"") || strings.Contains(contentStr, "'react'") {
				hasReact = true
			}
		}
	}

	// Build project info string
	var projectTypes []string

	if hasGoMod {
		projectTypes = append(projectTypes, "Go")
	}
	if hasPackageJSON {
		if hasNextJS {
			projectTypes = append(projectTypes, "Next.js")
		} else if hasReact {
			projectTypes = append(projectTypes, "React")
		} else if hasVue {
			projectTypes = append(projectTypes, "Vue.js")
		} else if hasAngular {
			projectTypes = append(projectTypes, "Angular")
		} else {
			projectTypes = append(projectTypes, "Node.js")
		}
	}
	if hasPomXML {
		projectTypes = append(projectTypes, "Java/Maven")
	}
	if hasCMake {
		projectTypes = append(projectTypes, "C/C++ with CMake")
	}
	if hasPyProject {
		if hasDjango {
			projectTypes = append(projectTypes, "Django")
		} else if hasFlask {
			projectTypes = append(projectTypes, "Flask")
		} else if hasFastAPI {
			projectTypes = append(projectTypes, "FastAPI")
		} else {
			projectTypes = append(projectTypes, "Python")
		}
	}
	if hasGradle {
		projectTypes = append(projectTypes, "Java/Gradle")
	}
	if hasUV {
		projectTypes = append(projectTypes, "Python with uv")
	}
	if hasBun {
		projectTypes = append(projectTypes, "JavaScript with Bun")
	}
	if hasMakefile {
		projectTypes = append(projectTypes, "C/C++ with Makefile")
	}
	if hasCargo {
		projectTypes = append(projectTypes, "Rust")
	}
	if hasComposer {
		projectTypes = append(projectTypes, "PHP with Composer")
	}
	if hasDocker {
		projectTypes = append(projectTypes, "Dockerized")
	}
	if hasKubernetes {
		projectTypes = append(projectTypes, "Kubernetes")
	}

	if len(projectTypes) > 0 {
		projectInfo.WriteString(strings.Join(projectTypes, ", "))
	} else {
		projectInfo.WriteString("Unknown project type")
	}

	return projectInfo.String(), nil
}

// makeCommit creates a git commit with the provided message
func makeCommit(message string) error {
	// Stage all changes
	_, err := runGitCommand("add", ".")
	if err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Create commit
	_, err = runGitCommand("commit", "-m", message)
	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	return nil
}

func generateCommitMessage(m string) (string, error) {
	changedFiles, err := getChangedFiles()
	if err != nil {
		return "", fmt.Errorf("couldn't get changed files: %w", err)
	}

	projectInfo, err := getProjectInfo()
	if err != nil {
		return "", fmt.Errorf("couldn't get project info: %w", err)
	}

	currentBranch, err := getCurrentBranch()
	if err != nil {
		// Don't fail if we can't get branch info, just continue without it
		currentBranch = "unknown"
	}

	var fileListStr string
	if len(changedFiles) > 0 {
		fileListStr = fmt.Sprintf("Changed files: %s\n\n", strings.Join(changedFiles, ", "))
	}

	prompt := "Generate a short, concise git commit message based on the following changes. " +
		"Follow the conventional commit format (e.g., feat:, fix:, docs:, style:, refactor:, test:, chore:). " +
		"Keep it under 50 characters if possible. " +
		"Only respond with the commit message, nothing else.\n\n"

	if projectInfo != "" {
		prompt += "Project context: " + projectInfo + "\n\n"
	}

	if currentBranch != "unknown" && currentBranch != "main" && currentBranch != "master" {
		prompt += fmt.Sprintf("Current branch: %s\n\n", currentBranch)
	}

	diff, err := getGitDiff()
	if err != nil {
		return "", fmt.Errorf("error getting git diff: %w", err)
	}

	// Use trackCodeChanges for structured analysis
	structuredChanges, err := trackCodeChanges(diff)
	if err == nil && len(structuredChanges) > 0 {
		prompt += "Structured changes by file:\n"
		for file, changes := range structuredChanges {
			if changes != "" {
				prompt += fmt.Sprintf("File: %s\n%s\n", file, strings.TrimSpace(changes))
			}
		}
		prompt += "\n"
	}

	prompt += fileListStr + "Full diff:\n" + diff

	client := openai.NewClient(
		option.WithBaseURL(viper.GetString("api_url")),
		option.WithAPIKey(viper.GetString("api_key")),
	)

	chatCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
		Model: m,
	})
	if err != nil {
		return "", fmt.Errorf("error calling AI service: %w", err)
	}

	return chatCompletion.Choices[0].Message.Content, nil
}

var RootCmd = &cobra.Command{
	Use:   "rmit",
	Short: "Generate git commit messages with AI",
	Long:  "rmit uses OpenRouter to generate descriptive git commit messages based on your changes",
	Run: func(cmd *cobra.Command, args []string) {
		es := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Bold(true)

		model := viper.GetString("default_model")

	retry_start:

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		prompt := ""
		var mutex sync.Mutex
		var generationErr error

		go func() {
			defer cancel()

			p, e := generateCommitMessage(model)
			if e != nil {
				mutex.Lock()
				generationErr = e
				mutex.Unlock()
				return
			}

			mutex.Lock()
			prompt = p
			mutex.Unlock()
		}()
		spinner.New().Title("Generating commit message...").Context(ctx).Run()

		mutex.Lock()
		if generationErr != nil {
			fmt.Fprintln(os.Stderr, es.Render("Error: "+generationErr.Error()))
			mutex.Unlock()
			return
		}
		mutex.Unlock()

		gts := lipgloss.NewStyle().Foreground(lipgloss.Color("#006affff")).Bold(true)
		gs := lipgloss.NewStyle().MarginLeft(3).MarginBottom(1).Foreground(lipgloss.Color("#00c3c3ff"))

		fmt.Println(gts.Render("Generated commit message:"))
		fmt.Println(gs.Render(prompt))

		if commitAuto {
			if err := makeCommit(prompt); err != nil {
				fmt.Fprintln(os.Stderr, es.Render("Error: "+err.Error()))
				return
			}
			fmt.Println("Successfully committed with message:", prompt)
		} else {
			commit := "Y"

			form := huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
						Title("Choose an option").
						Options(
							huh.NewOption("Yes", "Y"),
							huh.NewOption("Retry", "R"),
							huh.NewOption("Quit", "Q"),
						).
						Value(&commit),
				),
			)

			form.Run()

			switch commit {
			case "Y":
				if err := makeCommit(prompt); err != nil {
					fmt.Fprintln(os.Stderr, es.Render("Error: "+err.Error()))
					return
				}
			case "R":
				goto retry_start
			case "Q":
				quitStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
				fmt.Println(quitStyle.Render("Goodbye!"))
			}
		}

	},
}
