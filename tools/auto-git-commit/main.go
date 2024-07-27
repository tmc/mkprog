package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

//go:embed system-prompt.txt
var systemPrompt string

//go:embed commit-prompt.txt
var commitPrompt string

//go:embed conventional-commit-prompt.txt
var conventionalCommitPrompt string

const (
	defaultAuthorName  = "Auto Git Commit"
	defaultAuthorEmail = "auto@example.com"
	gitGuidelinesFile  = ".git-commit-guidelines"
)

var (
	verbose            bool
	dryRun             bool
	path               string
	conventionalCommit bool
)

func main() {
	flag.BoolVar(&verbose, "verbose", false, "Show reasoning for commit message generation")
	flag.BoolVar(&dryRun, "dry-run", false, "Generate commit message without actually committing")
	flag.StringVar(&path, "path", "", "Optional path to focus on a subtree")
	flag.BoolVar(&conventionalCommit, "conventional", false, "Use conventional commit format (not default)")

	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Get git status and diff
	changes, err := getGitChanges()
	if err != nil {
		return fmt.Errorf("failed to get git changes: %w", err)
	}

	if strings.TrimSpace(changes) == "" {
		return fmt.Errorf("no changes to commit")
	}

	// Generate commit message
	commitMessage, reasoning, err := generateCommitMessage(changes)
	if err != nil {
		return fmt.Errorf("failed to generate commit message: %w", err)
	}

	if verbose {
		fmt.Printf("Reasoning:\n%s\n\n", reasoning)
	}

	fmt.Printf("Generated commit message:\n\n%s\n\n", commitMessage)

	if dryRun {
		fmt.Println("Dry run: commit not created.")
		return nil
	}

	// Prompt for confirmation
	fmt.Print("Do you want to commit with this message? (y/n): ")
	var response string
	_, err = fmt.Scanln(&response)
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	if strings.ToLower(strings.TrimSpace(response)) != "y" {
		fmt.Println("Commit cancelled.")
		return nil
	}

	// Open the Git repository
	repo, err := git.PlainOpen(".")
	if err != nil {
		return fmt.Errorf("failed to open Git repository: %w", err)
	}

	// Get the working tree
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Commit changes
	commitOptions := &git.CommitOptions{
		All: true,
		Author: &object.Signature{
			Name:  defaultAuthorName,
			Email: defaultAuthorEmail,
		},
	}

	if path != "" {
		commitOptions.All = false
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
		_, err = worktree.Add(absPath)
		if err != nil {
			return fmt.Errorf("failed to add path to commit: %w", err)
		}
	}

	_, err = worktree.Commit(commitMessage, commitOptions)
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	fmt.Println("Changes committed successfully.")
	return nil
}

func getGitChanges() (string, error) {
	// Get git status
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git status: %w", err)
	}

	// Get git diff
	diffCmd := exec.Command("git", "diff", "--cached")
	diffOutput, err := diffCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git diff: %w", err)
	}

	// Combine status and diff
	changes := fmt.Sprintf("Git Status:\n%s\n\nGit Diff:\n%s", statusOutput, diffOutput)
	return changes, nil
}

func generateCommitMessage(changes string) (string, string, error) {
	ctx := context.Background()

	client, err := anthropic.New()
	if err != nil {
		return "", "", fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	prompt := systemPrompt
	if conventionalCommit {
		prompt = conventionalCommitPrompt
	}

	// Read and inject .git-commit-guidelines if it exists
	guidelines, err := readGitCommitGuidelines()
	if err == nil && guidelines != "" {
		prompt += "\n\nAdditional commit guidelines:\n" + guidelines
	}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, prompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Generate a commit message for the following changes:\n\n%s", changes)),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(4000))
	if err != nil {
		return "", "", fmt.Errorf("failed to generate commit message: %w", err)
	}

	content := resp.Choices[0].Content
	parts := strings.SplitN(content, "\n\n", 2)

	if len(parts) < 2 {
		return content, "", nil
	}

	return parts[0], parts[1], nil
}

func readGitCommitGuidelines() (string, error) {
	// Find the git root directory
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to find git root: %w", err)
	}
	gitRoot := strings.TrimSpace(string(output))

	// Read the .git-commit-guidelines file
	guidelinesPath := filepath.Join(gitRoot, gitGuidelinesFile)
	content, err := ioutil.ReadFile(guidelinesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // File doesn't exist, which is okay
		}
		return "", fmt.Errorf("failed to read %s: %w", gitGuidelinesFile, err)
	}

	return string(content), nil
}
