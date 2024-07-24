package main

import (
	"context"
	_ "embed"
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
	if len(os.Args) <= 2 {
		return fmt.Errorf("usage: %s <file> <change description>", os.Args[0])
	}

	filename := os.Args[1]
	changeDescription := strings.Join(os.Args[2:], " ")

	if !isGitClean() {
		return fmt.Errorf("git working directory is not clean")
	}

	originalContent, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	client, err := anthropic.New()
	if err != nil {
		return fmt.Errorf("error creating Anthropic client: %w", err)
	}

	ctx := context.Background()
	improvedContent, err := improveProgram(ctx, client, string(originalContent), changeDescription)
	if err != nil {
		return fmt.Errorf("error improving program: %w", err)
	}

	if err := os.WriteFile(filename, []byte(improvedContent), 0644); err != nil {
		return fmt.Errorf("error writing improved content: %w", err)
	}

	fmt.Println("Program improved successfully!")
	return nil
}

func improveProgram(ctx context.Context, client *anthropic.LLM, originalContent, changeDescription string) (string, error) {
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Original program:\n\n%s\n\nChange description: %s", originalContent, changeDescription)),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(4000))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	return resp.Choices[0].Content, nil
}

func isGitClean() bool {
	cmd := exec.Command("git", "status", "--porcelain", "--untracked-files=no", ".")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(output) == 0
}
