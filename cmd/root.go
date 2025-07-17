package cmd

import (
	"fmt"
	"strings"

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

	// First try to get staged diff (index to HEAD)
	stagedChanges, err := worktree.Diff(&git.DiffOptions{Staged: true})
	if err != nil {
		return "", fmt.Errorf("failed to get staged diff: %w", err)
	}

	if stagedChanges.Len() > 0 {
		patch, err := stagedChanges.Patch()
		if err != nil {
			return "", fmt.Errorf("failed to generate patch: %w", err)
		}
		return patch.String(), nil
	}

	// If no staged changes, get unstaged diff (worktree to index)
	unstagedChanges, err := worktree.Diff(&git.DiffOptions{Staged: false})
	if err != nil {
		return "", fmt.Errorf("failed to get unstaged diff: %w", err)
	}

	if unstagedChanges.Len() == 0 {
		return "", fmt.Errorf("no changes detected in the repository")
	}

	patch, err := unstagedChanges.Patch()
	if err != nil {
		return "", fmt.Errorf("failed to generate patch: %w", err)
	}
	return patch.String(), nil
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
