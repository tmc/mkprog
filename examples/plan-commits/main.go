package main

import (
	"context"
	_ "embed"
	"encoding/json"
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

type CommitPlan struct {
	Commits []Commit `json:"commits"`
}

type Commit struct {
	Type    string `json:"type"`
	Scope   string `json:"scope"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	maxCommits := flag.Int("max-commits", 5, "Maximum number of commits to generate")
	useConventionalCommits := flag.Bool("conventional", false, "Use conventional commit format")
	flag.Parse()

	diff, err := getGitDiff()
	if err != nil {
		return fmt.Errorf("failed to get git diff: %w", err)
	}

	if diff == "" {
		return fmt.Errorf("no changes detected in the git repository")
	}

	client, err := anthropic.New()
	if err != nil {
		return fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	ctx := context.Background()
	commitPlan, err := generateCommitPlan(ctx, client, diff, *maxCommits, *useConventionalCommits)
	if err != nil {
		return fmt.Errorf("failed to generate commit plan: %w", err)
	}

	printCommitPlan(commitPlan, *useConventionalCommits)
	return nil
}

func getGitDiff() (string, error) {
	cmd := exec.Command("git", "diff")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func generateCommitPlan(ctx context.Context, client *anthropic.Chat, diff string, maxCommits int, useConventionalCommits bool) (CommitPlan, error) {
	prompt := fmt.Sprintf("Git diff:\n\n%s\n\nGenerate a commit plan with up to %d commits. %s", diff, maxCommits, getCommitFormatInstructions(useConventionalCommits))

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(4000))
	if err != nil {
		return CommitPlan{}, err
	}

	return parseCommitPlan(resp.Choices[0].Content)
}

func getCommitFormatInstructions(useConventionalCommits bool) string {
	if useConventionalCommits {
		return "Use conventional commit format."
	}
	return "Use a simple format with subject and body."
}

func parseCommitPlan(content string) (CommitPlan, error) {
	var plan CommitPlan
	err := json.Unmarshal([]byte(content), &plan)
	if err != nil {
		return CommitPlan{}, fmt.Errorf("failed to parse commit plan: %w", err)
	}
	return plan, nil
}

func printCommitPlan(plan CommitPlan, useConventionalCommits bool) {
	fmt.Println("Commit Plan:")
	for i, commit := range plan.Commits {
		fmt.Printf("\nCommit %d:\n", i+1)
		if useConventionalCommits {
			fmt.Printf("%s(%s): %s\n", commit.Type, commit.Scope, commit.Subject)
		} else {
			fmt.Printf("Subject: %s\n", commit.Subject)
		}
		fmt.Printf("Body: %s\n", commit.Body)
	}
}

