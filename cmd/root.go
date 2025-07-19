package cmd

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	git "github.com/go-git/go-git/v6"
	"github.com/spf13/cobra"
)

// getGitDiff returns the unified diff of staged or unstaged changes using go-git
func getGitDiff() (string, error) {
	// Open the repository in current directory
	repo, err := git.PlainOpen(".")
	if err != nil {
		return "", fmt.Errorf("current directory is not a git repository: %w", err)
	}
	// Determine staged vs unstaged changes
	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("unable to access worktree: %w", err)
	}
	// Prefer staged changes
	status, err := wt.Status()
	if err != nil {
		return "", fmt.Errorf("failed to get repository status: %w", err)
	}
	// Build diff via git CLI for simplicity
	// If there are staged changes, use --staged diff, otherwise fallback to unstaged
	var args []string
	for _, fs := range status {
		if fs.Staging != git.Unmodified {
			args = []string{"diff", "--staged"}
			break
		}
	}
	if len(args) == 0 {
		// no staged; use unstaged
		args = []string{"diff"}
	}
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git diff: %w", err)
	}
	if len(out) == 0 {
		return "", fmt.Errorf("no changes detected in the repository")
	}
	return string(out), nil
}

// trackCodeChanges returns a map of file paths to their diff hunks using go-git
func trackCodeChanges(_ string) (map[string]string, error) {
	// Obtain the full diff
	d, err := getGitDiff()
	if err != nil {
		return nil, err
	}
	// Get list of changed files
	files, err := getChangedFiles()
	if err != nil {
		return nil, err
	}
	// For each file, extract its diff block
	changes := make(map[string]string, len(files))
	for _, f := range files {
		// match diff header and content for this file
		re := regexp.MustCompile(`(?ms)^diff --git a/` + regexp.QuoteMeta(f) + ` b/` + regexp.QuoteMeta(f) + `(.*?)(?=^diff --git|\z)`)
		if m := re.FindString(d); m != "" {
			changes[f] = strings.TrimSpace(m)
		}
	}
	return changes, nil
}

// getChangedFiles returns the list of staged or unstaged changed files using go-git
func getChangedFiles() ([]string, error) {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return nil, fmt.Errorf("current directory is not a git repository: %w", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("unable to access worktree: %w", err)
	}
	status, err := wt.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository status: %w", err)
	}
	// Collect staged changes
	var staged []string
	for file, fs := range status {
		if fs.Staging != git.Unmodified {
			staged = append(staged, file)
		}
	}
	if len(staged) > 0 {
		return staged, nil
	}
	// No staged, collect unstaged changes
	var unstaged []string
	for file, fs := range status {
		if fs.Worktree != git.Unmodified {
			unstaged = append(unstaged, file)
		}
	}
	if len(unstaged) > 0 {
		return unstaged, nil
	}
	return nil, fmt.Errorf("no changed files detected in the repository")
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

var RootCmd = &cobra.Command{
	Use:   "rmit",
	Short: "Generate git commit messages with AI",
	Long:  "rmit uses OpenRouter to generate descriptive git commit messages based on your changes",
	Run: func(cmd *cobra.Command, args []string) {
		ws := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00")).Bold(true)
		es := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Bold(true)

		changedFiles, err := getChangedFiles()
		if err != nil {
			log.Printf("%s couldn't get changed files: %v", ws.Render("Warning:"), err)
		}

		projectInfo, err := getProjectInfo()
		if err != nil {
			log.Printf("%s couldn't get project info: %v", ws.Render("Warning:"), err)
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
			log.Fatalf("%s %v", es.Render("Error getting git diff:"), err)
		}

		prompt += fileListStr + "Changes:\n" + diff
	},
}
