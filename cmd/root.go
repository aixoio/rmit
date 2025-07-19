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

	// Work on this function based on the comment AI!

	/* Reference function, please recreate the same functionity as this but with go-git v6
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
	*/

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
