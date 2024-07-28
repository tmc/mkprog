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
	"sync"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

//go:embed system-prompt.txt
var systemPrompt string

var verbose bool
var dir string
var dryRun bool
var concurrency int
var fileExtensions string

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
	flag.IntVar(&concurrency, "concurrency", 5, "Number of concurrent file processing")
	flag.StringVar(&fileExtensions, "extensions", ".go,.py,.js,.java,.cpp", "Comma-separated list of file extensions to process")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		return fmt.Errorf("usage: %s [-verbose] [-dir <directory>] [-dry-run] [-concurrency <num>] [-extensions <ext1,ext2,...>] <change description>", os.Args[0])
	}

	changeDescription := strings.Join(args, " ")

	if verbose {
		fmt.Printf("Directory: %s\nChange description: %s\nDry run: %v\nConcurrency: %d\nFile extensions: %s\n", dir, changeDescription, dryRun, concurrency, fileExtensions)
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

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)
	errChan := make(chan error, 1)

	extensions := strings.Split(fileExtensions, ",")

	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !hasValidExtension(path, extensions) {
			return nil
		}

		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := processFile(ctx, client, path, changeDescription); err != nil {
				select {
				case errChan <- fmt.Errorf("error processing file %s: %w", path, err):
				default:
				}
			}
		}(path)

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking directory: %w", err)
	}

	wg.Wait()
	close(errChan)

	if err, ok := <-errChan; ok {
		return err
	}

	if dryRun {
		fmt.Println("Dry run completed. No changes were made.")
	} else {
		fmt.Println("All programs in the directory improved successfully!")
	}
	return nil
}

func hasValidExtension(path string, extensions []string) bool {
	ext := filepath.Ext(path)
	for _, validExt := range extensions {
		if ext == strings.TrimSpace(validExt) {
			return true
		}
	}
	return false
}

func processFile(ctx context.Context, client *anthropic.LLM, path, changeDescription string) error {
	originalContent, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading file %s: %w", path, err)
	}

	improvedContent, reasoning, err := improveProgram(ctx, client, string(originalContent), changeDescription, filepath.Ext(path))
	if err != nil {
		return fmt.Errorf("error improving program %s: %w", path, err)
	}

	// Add a safeguard to prevent empty content
	if len(strings.TrimSpace(improvedContent)) == 0 {
		return fmt.Errorf("improved content for %s is empty, skipping update", path)
	}

	if verbose {
		fmt.Printf("Reasoning for %s:\n%s\n", path, reasoning)
	}

	if dryRun {
		fmt.Printf("Dry run: Would improve %s\n", path)
	} else {
		// Create a backup of the original file
		backupPath := path + ".bak"
		if err := os.WriteFile(backupPath, originalContent, 0644); err != nil {
			return fmt.Errorf("error creating backup file %s: %w", backupPath, err)
		}

		if err := os.WriteFile(path, []byte(improvedContent), 0644); err != nil {
			return fmt.Errorf("error writing improved content to %s: %w", path, err)
		}
		fmt.Printf("Improved %s successfully!\n", path)
	}
	return nil
}

func improveProgram(ctx context.Context, client *anthropic.LLM, originalContent, changeDescription, fileExtension string) (string, string, error) {
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Original program (%s):\n\n%s\n\nChange description: %s\n\nPlease use <anthinking> tags to show your reasoning process.", fileExtension, originalContent, changeDescription)),
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

	// Add a safeguard to prevent empty content
	if len(strings.TrimSpace(improvedProgram)) == 0 {
		return originalContent, reasoning, fmt.Errorf("improved program is empty, keeping original content")
	}

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
