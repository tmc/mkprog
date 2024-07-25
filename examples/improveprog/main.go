package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"io/fs"
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
var dryRun bool

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.StringVar(&dir, "dir", ".", "Directory to operate in")
	flag.BoolVar(&dryRun, "dry-run", false, "Perform a dry run without making changes")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		return fmt.Errorf("usage: %s [-verbose] [-dir <directory>] [-dry-run] <change description>", os.Args[0])
	}

	changeDescription := strings.Join(args, " ")

	if verbose {
		fmt.Printf("Directory: %s\nChange description: %s\nDry run: %v\n", dir, changeDescription, dryRun)
	}

	// Change to the specified directory
	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("error changing to directory %s: %w", dir, err)
	}

	if !isGitClean() {
		return fmt.Errorf("git working directory is not clean")
	}

	client, err := anthropic.New()
	if err != nil {
		return fmt.Errorf("error creating Anthropic client: %w", err)
	}

	ctx := context.Background()

	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		originalContent, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("error reading file %s: %w", path, err)
		}

		improvedContent, reasoning, err := improveProgram(ctx, client, string(originalContent), changeDescription)
		if err != nil {
			return fmt.Errorf("error improving program %s: %w", path, err)
		}

		if verbose {
			fmt.Printf("Reasoning for %s:\n%s\n", path, reasoning)
		}

		if dryRun {
			fmt.Printf("Dry run: Would improve %s\n", path)
		} else {
			if err := os.WriteFile(path, []byte(improvedContent), 0644); err != nil {
				return fmt.Errorf("error writing improved content to %s: %w", path, err)
			}
			fmt.Printf("Improved %s successfully!\n", path)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("error processing files: %w", err)
	}

	if dryRun {
		fmt.Println("Dry run completed. No changes were made.")
	} else {
		fmt.Println("All programs in the directory improved successfully!")
	}
	return nil
}

func improveProgram(ctx context.Context, client *anthropic.LLM, originalContent, changeDescription string) (string, string, error) {
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Original program:\n\n%s\n\nChange description: %s\n\nPlease use <anthinking> tags to show your reasoning process.", originalContent, changeDescription)),
	}

	if verbose {
		fmt.Println("Sending request to Anthropic API...")
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(4000))
	if err != nil {
		return "", "", fmt.Errorf("failed to generate content: %w", err)
	}

	if verbose {
		fmt.Println("Received response from Anthropic API")
	}

	content := resp.Choices[0].Content
	improvedProgram, reasoning := extractProgramAndReasoning(content)

	return improvedProgram, reasoning, nil
}

func extractProgramAndReasoning(content string) (string, string) {
	parts := strings.Split(content, "<anthinking>")
	if len(parts) < 2 {
		return content, ""
	}

	program := strings.TrimSpace(parts[0])
	reasoning := strings.TrimSpace(strings.TrimSuffix(parts[1], "</anthinking>"))

	return program, reasoning
}

func isGitClean() bool {
	cmd := exec.Command("git", "status", "--porcelain", "--untracked-files=no", ".")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(output) == 0
}