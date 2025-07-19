package cmd

import (
	"fmt"
	"strings"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
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

	// Get staged changes first
	status, err := worktree.Status()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree status: %w", err)
	}

	hasStaged := false
	for _, fileStatus := range status {
		if fileStatus.Staging != 0 {
			hasStaged = true
			break
		}
	}

	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	headCommit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD commit: %w", err)
	}

	headTree, err := headCommit.Tree()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD tree: %w", err)
	}

	var diff *object.Tree
	if hasStaged {
		// Get staged changes
		index, err := repo.Storer.Index()
		if err != nil {
			return "", fmt.Errorf("failed to get index: %w", err)
		}

		indexTree, err := index.Tree()
		if err != nil {
			return "", fmt.Errorf("failed to get index tree: %w", err)
		}

		diff, err = headTree.Diff(indexTree)
		if err != nil {
			return "", fmt.Errorf("failed to get staged diff: %w", err)
		}
	} else {
		// Get unstaged changes
		diff, err = worktree.Diff(headTree)
		if err != nil {
			return "", fmt.Errorf("failed to get unstaged diff: %w", err)
		}
	}

	patch, err := diff.Patch()
	if err != nil {
		return "", fmt.Errorf("failed to create patch: %w", err)
	}

	output := patch.String()
	if len(strings.TrimSpace(output)) == 0 {
		return "", fmt.Errorf("no changes detected in the repository")
	}

	return output, nil
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
