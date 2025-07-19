package cmd

import (
	"fmt"

	"github.com/go-git/go-git/v6"
	"github.com/spf13/cobra"
)

func getGitDiff() (string, error) {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return "", fmt.Errorf("current directory is not a git repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}
	// you need in update imports ai!
	// Get staged changes
	stagedDiff, err := worktree.DiffStaging()
	if err != nil {
		return "", fmt.Errorf("failed to get staged changes: %w", err)
	}

	stagedPatch, err := stagedDiff.Patch()
	if err != nil {
		return "", fmt.Errorf("failed to create staged patch: %w", err)
	}

	stagedOutput := stagedPatch.String()

	// If we have staged changes, return them
	if len(stagedOutput) > 0 {
		return stagedOutput, nil
	}

	// Get unstaged changes
	unstagedDiff, err := worktree.Diff()
	if err != nil {
		return "", fmt.Errorf("failed to get unstaged changes: %w", err)
	}

	unstagedPatch, err := unstagedDiff.Patch()
	if err != nil {
		return "", fmt.Errorf("failed to create unstaged patch: %w", err)
	}

	unstagedOutput := unstagedPatch.String()

	if len(unstagedOutput) == 0 {
		return "", fmt.Errorf("no changes detected in the repository")
	}

	return unstagedOutput, nil
}

var RootCmd = &cobra.Command{
	Use:   "rmit",
	Short: "Generate git commit messages with AI",
	Long:  "rmit uses OpenRouter to generate descriptive git commit messages based on your changes",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hi")
		fmt.Println(getGitDiff())
	},
}
