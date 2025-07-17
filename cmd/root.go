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

	status, err := worktree.Status()
	if err != nil {
		return "", fmt.Errorf("failed to get status: %w", err)
	}

	var diffBuilder strings.Builder
	for path, stat := range status {
		switch {
		case stat.Staging == git.Added:
			diffBuilder.WriteString(fmt.Sprintf("A  %s (staged)\n", path))
		case stat.Staging == git.Deleted:
			diffBuilder.WriteString(fmt.Sprintf("D  %s (staged)\n", path))
		case stat.Staging == git.Modified:
			diffBuilder.WriteString(fmt.Sprintf("M  %s (staged)\n", path))
		case stat.Worktree == git.Untracked:
			diffBuilder.WriteString(fmt.Sprintf("?? %s\n", path))
		case stat.Worktree == git.Modified:
			diffBuilder.WriteString(fmt.Sprintf(" M %s\n", path))
		case stat.Worktree == git.Deleted:
			diffBuilder.WriteString(fmt.Sprintf(" D %s\n", path))
		}
	}
	return diffBuilder.String(), nil
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
