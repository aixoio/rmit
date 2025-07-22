package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aixoio/rmit/editor"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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
	hasGradle := false
	hasUV := false
	hasBun := false
	hasMakefile := false
	hasCargo := false
	hasComposer := false

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
	if hasGradle {
		projectInfo.WriteString("Java/Gradle project. ")
	}
	if hasUV {
		projectInfo.WriteString("Python/uv project. ")
	}
	if hasBun {
		projectInfo.WriteString("JavaScript/Bun project. ")
	}
	if hasMakefile {
		projectInfo.WriteString("C/C++ project with Makefile. ")
	}
	if hasCargo {
		projectInfo.WriteString("Rust project. ")
	}
	if hasComposer {
		projectInfo.WriteString("PHP/Composer project. ")
	}

	return projectInfo.String(), nil
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
	if err := commitCmd.Run(); err != nil {
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

	var fileListStr string
	if len(changedFiles) > 0 {
		fileListStr = fmt.Sprintf("Changed files: %s\n\n", strings.Join(changedFiles, ", "))
	}

	prompt := "Generate a short, concise git commit message based on the following changes. " +
		"Follow the conventional commit format (e.g., feat:, fix:, docs:, style:, refactor:, test:, chore:). " +
		"Keep it under 50 characters if possible. " +
		"Only respond with the commit message, nothing else.\n\n"

	if projectInfo != "" {
		prompt += "Project information: " + projectInfo + "\n\n"
	}

	diff, err := getGitDiff()
	if err != nil {
		return "", fmt.Errorf("error getting git diff: %w", err)
	}

	prompt += fileListStr + "Changes:\n" + diff

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
		return "", fmt.Errorf("error getting git diff: %w", err)
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

		commit := "Y"

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Choose an option").
					Options(
						huh.NewOption("Yes", "Y"),
						huh.NewOption("Edit", "EM"),
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
		case "EM":
			editor.StartEditor(prompt)
		case "Q":
			quitStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
			fmt.Println(quitStyle.Render("Goodbye!"))
		}

	},
}
