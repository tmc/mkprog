package main

import (
	"bufio"
	"context"
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

//go:embed system-prompt.txt
var systemPrompt string

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var commitType, commitScope string
	var interactive bool

	rootCmd := &cobra.Command{
		Use:   "mkcommit",
		Short: "Generate a Git commit message based on repository context",
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateCommit(commitType, commitScope, interactive)
		},
	}

	rootCmd.Flags().StringVarP(&commitType, "type", "t", "", "Commit type (e.g., feat, fix, docs)")
	rootCmd.Flags().StringVarP(&commitScope, "scope", "s", "", "Commit scope")
	rootCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive mode")

	return rootCmd.Execute()
}

func generateCommit(commitType, commitScope string, interactive bool) error {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return fmt.Errorf("failed to open Git repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("failed to get worktree status: %w", err)
	}

	if status.IsClean() {
		return fmt.Errorf("working tree is clean, nothing to commit")
	}

	changedFiles, err := getChangedFiles(status)
	if err != nil {
		return fmt.Errorf("failed to get changed files: %w", err)
	}

	recentCommits, err := getRecentCommits(repo, 10)
	if err != nil {
		return fmt.Errorf("failed to get recent commits: %w", err)
	}

	commitMessage, err := generateCommitMessage(changedFiles, recentCommits, commitType, commitScope)
	if err != nil {
		return fmt.Errorf("failed to generate commit message: %w", err)
	}

	if interactive {
		commitMessage, err = promptUserForConfirmation(commitMessage)
		if err != nil {
			return fmt.Errorf("failed to get user confirmation: %w", err)
		}
	}

	fmt.Printf("Suggested commit message:\n\n%s\n\n", commitMessage)
	fmt.Printf("To create the commit, run:\n\ngit commit -m \"%s\"\n", strings.ReplaceAll(commitMessage, "\"", "\\\""))

	return nil
}

func getChangedFiles(status git.Status) ([]string, error) {
	var changedFiles []string
	for file, fileStatus := range status {
		if fileStatus.Staging != git.Unmodified || fileStatus.Worktree != git.Unmodified {
			changedFiles = append(changedFiles, file)
		}
	}
	return changedFiles, nil
}

func getRecentCommits(repo *git.Repository, count int) ([]string, error) {
	var commits []string
	iter, err := repo.Log(&git.LogOptions{})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	err = iter.ForEach(func(c *object.Commit) error {
		if len(commits) >= count {
			return nil
		}
		commits = append(commits, c.Message)
		return nil
	})

	return commits, err
}

func generateCommitMessage(changedFiles, recentCommits []string, commitType, commitScope string) (string, error) {
	ctx := context.Background()
	client, err := anthropic.New()
	if err != nil {
		return "", fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	prompt := fmt.Sprintf("Changed files:\n%s\n\nRecent commits:\n%s\n\nCommit type: %s\nCommit scope: %s\n\nGenerate a suitable commit message:",
		strings.Join(changedFiles, "\n"),
		strings.Join(recentCommits, "\n"),
		commitType,
		commitScope)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(4000))
	if err != nil {
		return "", fmt.Errorf("failed to generate commit message: %w", err)
	}

	return resp.Choices[0].Content, nil
}

func promptUserForConfirmation(commitMessage string) (string, error) {
	fmt.Printf("Suggested commit message:\n\n%s\n\nDo you want to use this message? (y/n/e to edit): ", commitMessage)
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		return "", err
	}

	switch strings.ToLower(response) {
	case "y", "yes":
		return commitMessage, nil
	case "n", "no":
		return "", fmt.Errorf("user rejected the commit message")
	case "e", "edit":
		return promptUserForEdit(commitMessage)
	default:
		return promptUserForConfirmation(commitMessage)
	}
}

func promptUserForEdit(commitMessage string) (string, error) {
	fmt.Println("Enter your edited commit message (type 'done' on a new line when finished):")
	var lines []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "done" {
			break
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return strings.Join(lines, "\n"), nil
}
