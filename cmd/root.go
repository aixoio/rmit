package cmd

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

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

var RootCmd = &cobra.Command{
	Use:   "rmit",
	Short: "Generate git commit messages with AI",
	Long:  "rmit uses OpenRouter to generate descriptive git commit messages based on your changes",
	Run: func(cmd *cobra.Command, args []string) {

	},
}
