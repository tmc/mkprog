package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

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
	numCommits := flag.Int("commits", 10, "Number of recent commits to analyze")
	branch := flag.String("branch", "", "Analyze a specific branch instead of the current one")
	verbose := flag.Bool("verbose", false, "Provide more detailed output")
	flag.Parse()

	ctx := context.Background()

	gitData, err := collectGitData(*numCommits, *branch, *verbose)
	if err != nil {
		return fmt.Errorf("failed to collect git data: %w", err)
	}

	analysis, err := analyzeWithAI(ctx, gitData)
	if err != nil {
		return fmt.Errorf("failed to analyze with AI: %w", err)
	}

	fmt.Println("Analysis Results:")
	fmt.Println(analysis)

	return nil
}

func collectGitData(numCommits int, branch string, verbose bool) (string, error) {
	var data strings.Builder

	// Get recent commits
	commits, err := getRecentCommits(numCommits, branch)
	if err != nil {
		return "", fmt.Errorf("failed to get recent commits: %w", err)
	}
	data.WriteString(fmt.Sprintf("Recent Commits:\n%s\n", commits))

	// Get try attempts from Git notes
	tryAttempts, err := getTryAttempts()
	if err != nil {
		return "", fmt.Errorf("failed to get try attempts: %w", err)
	}
	data.WriteString(fmt.Sprintf("Try Attempts:\n%s\n", tryAttempts))

	// Get current branch status
	branchStatus, err := getBranchStatus(branch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch status: %w", err)
	}
	data.WriteString(fmt.Sprintf("Branch Status:\n%s\n", branchStatus))

	if verbose {
		fmt.Println("Collected Git Data:")
		fmt.Println(data.String())
	}

	return data.String(), nil
}

func getRecentCommits(numCommits int, branch string) (string, error) {
	args := []string{"log", "--oneline", fmt.Sprintf("-%d", numCommits)}
	if branch != "" {
		args = append(args, branch)
	}
	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git log command failed: %w", err)
	}
	return string(output), nil
}

func getTryAttempts() (string, error) {
	cmd := exec.Command("git", "notes", "show")
	output, err := cmd.Output()
	if err != nil {
		// If there are no notes, don't treat it as an error
		if strings.Contains(err.Error(), "No note found") {
			return "No try attempts found.", nil
		}
		return "", fmt.Errorf("git notes command failed: %w", err)
	}
	return string(output), nil
}

func getBranchStatus(branch string) (string, error) {
	if branch == "" {
		// Get current branch name
		cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get current branch: %w", err)
		}
		branch = strings.TrimSpace(string(output))
	}

	// Get branch status
	cmd := exec.Command("git", "status", "--short", "--branch")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git status command failed: %w", err)
	}
	return fmt.Sprintf("Branch: %s\n%s", branch, string(output)), nil
}

func analyzeWithAI(ctx context.Context, gitData string) (string, error) {
	client, err := anthropic.New()
	if err != nil {
		return "", fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	prompt := fmt.Sprintf("%s\n\nGit Repository Data:\n%s\n\nPlease analyze the provided Git repository data, focusing on patterns in the 'try' attempts, and suggest potential edits or improvements based on the commit history and branch status.", systemPrompt, gitData)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(4000))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	// The response content is now directly accessible as a string
	return resp.Choices[0].Content, nil
}
