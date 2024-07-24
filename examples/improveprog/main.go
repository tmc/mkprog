package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

//go:embed system-prompt.txt
var systemPrompt string

var verbose bool
var dir string

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.StringVar(&dir, "dir", ".", "Directory to operate in")
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		return fmt.Errorf("usage: %s [-verbose] [-dir <directory>] <file> <change description>", os.Args[0])
	}

	filename := args[0]
	changeDescription := strings.Join(args[1:], " ")

	if verbose {
		fmt.Printf("Directory: %s\nFile: %s\nChange description: %s\n", dir, filename, changeDescription)
	}

	// Change to the specified directory
	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("error changing to directory %s: %w", dir, err)
	}

	if !isGitClean() {
		return fmt.Errorf("git working directory is not clean")
	}

	fullPath := filepath.Join(dir, filename)
	originalContent, err := os.ReadFile(fullPath)
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

	if err := os.WriteFile(fullPath, []byte(improvedContent), 0644); err != nil {
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

	if verbose {
		fmt.Println("Sending request to Anthropic API...")
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(4000))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	if verbose {
		fmt.Println("Received response from Anthropic API")
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