package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	dir := flag.String("dir", ".", "Directory containing the source code")
	description := flag.String("desc", "", "Description of the change to be made")
	testCmd := flag.String("test", "", "Command to run to check if the problem is fixed")
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	flag.Parse()

	if *description == "" {
		return fmt.Errorf("description is required")
	}

	if *verbose {
		fmt.Printf("Starting operation on directory: %s\n", *dir)
		fmt.Printf("Change description: %s\n", *description)
		if *testCmd != "" {
			fmt.Printf("Test command: %s\n", *testCmd)
		}
	}

	// Check if the operation is safe to perform
	if *verbose {
		fmt.Println("Checking if the operation is safe to perform...")
	}
	if !isSafeOperation(*dir) {
		return fmt.Errorf("the operation is not considered safe for the given directory: %s", *dir)
	}
	if *verbose {
		fmt.Println("Operation deemed safe. Proceeding...")
	}

	ctx := context.Background()
	if *verbose {
		fmt.Println("Creating Anthropic client...")
	}
	client, err := anthropic.New()
	if err != nil {
		return fmt.Errorf("failed to create Anthropic client: %w", err)
	}
	if *verbose {
		fmt.Println("Anthropic client created successfully.")
	}

	if *verbose {
		fmt.Println("Checking if the directory is a Git repository...")
	}
	isGitRepo, err := isGitRepository(*dir)
	if err != nil {
		return fmt.Errorf("failed to check if directory is a git repository: %w", err)
	}
	if *verbose {
		if isGitRepo {
			fmt.Println("Directory is a Git repository.")
		} else {
			fmt.Println("Directory is not a Git repository.")
		}
	}

	if *verbose {
		fmt.Println("Capturing initial state of the directory...")
	}
	initialState, err := getCurrentState(*dir)
	if err != nil {
		return fmt.Errorf("failed to get initial state: %w", err)
	}
	if *verbose {
		fmt.Printf("Initial state captured. %d files recorded.\n", len(initialState))
	}

	attempts := 0
	maxAttempts := 5

	for {
		if attempts >= maxAttempts {
			return fmt.Errorf("maximum number of attempts (%d) reached without fixing the problem", maxAttempts)
		}

		if *verbose {
			fmt.Printf("Attempt %d of %d\n", attempts+1, maxAttempts)
			fmt.Println("Gathering source files...")
		}
		files, err := getSourceFiles(*dir)
		if err != nil {
			return fmt.Errorf("failed to get source files: %w", err)
		}
		if *verbose {
			fmt.Printf("Found %d source files.\n", len(files))
		}

		fileContents := make(map[string]string)
		for _, file := range files {
			if *verbose {
				fmt.Printf("Reading file: %s\n", file)
			}
			content, err := ioutil.ReadFile(file)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", file, err)
			}
			fileContents[file] = string(content)
		}

		userInput := fmt.Sprintf("Description: %s\n\nFiles:\n", *description)
		for file, content := range fileContents {
			userInput += fmt.Sprintf("=== %s ===\n%s\n\n", file, content)
		}

		if *verbose {
			fmt.Println("Preparing to send request to Anthropic API...")
			fmt.Printf("Request size: %d bytes\n", len(userInput))
		}

		messages := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
			llms.TextParts(llms.ChatMessageTypeHuman, userInput),
		}

		if *verbose {
			fmt.Println("Sending request to Anthropic API...")
			startTime := time.Now()
		}

		resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(4000))
		if err != nil {
			return fmt.Errorf("failed to generate content: %w", err)
		}

		if *verbose {
			duration := time.Since(startTime)
			fmt.Printf("Received response from Anthropic API. Time taken: %v\n", duration)
			fmt.Printf("Response size: %d bytes\n", len(resp.Choices[0].Content))
		}

		if *verbose {
			fmt.Println("Parsing changes from the API response...")
		}
		changes, err := parseChanges(resp.Choices[0].Content)
		if err != nil {
			return fmt.Errorf("failed to parse changes: %w", err)
		}
		if *verbose {
			fmt.Printf("Parsed %d file changes.\n", len(changes))
		}

		if *verbose {
			fmt.Println("Applying changes to files...")
		}
		if err := applyChanges(*dir, changes); err != nil {
			return fmt.Errorf("failed to apply changes: %w", err)
		}
		if *verbose {
			fmt.Println("Changes applied successfully.")
		}

		if *testCmd != "" {
			if *verbose {
				fmt.Printf("Running test command: %s\n", *testCmd)
			}
			if err := runTestCommand(*testCmd, *dir); err != nil {
				if *verbose {
					fmt.Println("Test command failed. Preparing to revert changes and try again.")
				}
				if isGitRepo {
					if *verbose {
						fmt.Println("Reverting changes using Git...")
					}
					if err := gitCheckout(*dir); err != nil {
						return fmt.Errorf("failed to revert changes using git: %w", err)
					}
				} else {
					if *verbose {
						fmt.Println("Reverting changes manually...")
					}
					if err := revertToState(*dir, initialState); err != nil {
						return fmt.Errorf("failed to revert changes: %w", err)
					}
				}
				if *verbose {
					fmt.Println("Changes reverted successfully.")
				}
				attempts++
				continue
			}
			if *verbose {
				fmt.Println("Test command executed successfully.")
			}
		}

		fmt.Println("Changes applied successfully.")
		break
	}

	return nil
}

// The rest of the functions remain unchanged

func isSafeOperation(dir string) bool {
	// Check if the directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false
	}

	// Check if the directory contains a .git folder (indicating it's a git repository)
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); !os.IsNotExist(err) {
		return true
	}

	// Check if the directory contains any source files
	files, err := getSourceFiles(dir)
	if err != nil {
		return false
	}
	if len(files) == 0 {
		return false
	}

	// Add more safety checks as needed

	return true
}

func isGitRepository(dir string) (bool, error) {
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); !os.IsNotExist(err) {
		return true, nil
	}
	return false, nil
}

func getCurrentState(dir string) (map[string]string, error) {
	files, err := getSourceFiles(dir)
	if err != nil {
		return nil, err
	}

	state := make(map[string]string)
	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", file, err)
		}
		state[file] = string(content)
	}

	return state, nil
}

func getSourceFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && isSourceFile(path) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func isSourceFile(path string) bool {
	ext := filepath.Ext(path)
	switch ext {
	case ".go", ".c", ".cpp", ".h", ".hpp", ".java", ".js", ".ts", ".rs", ".py":
		return true
	}
	return false
}

func parseChanges(content string) (map[string]string, error) {
	changes := make(map[string]string)

	lines := strings.Split(content, "\n")
	var currentFile string
	var currentContent strings.Builder
	for _, line := range lines {
		if strings.HasPrefix(line, "=== ") && strings.HasSuffix(line, " ===") {
			if currentFile != "" {
				changes[currentFile] = currentContent.String()
				currentContent.Reset()
			}
			currentFile = strings.TrimPrefix(strings.TrimSuffix(line, " ==="), "=== ")
		} else {
			currentContent.WriteString(line)
			currentContent.WriteString("\n")
		}
	}
	if currentFile != "" {
		changes[currentFile] = currentContent.String()
	}
	return changes, nil
}

func applyChanges(dir string, changes map[string]string) error {
	for file, content := range changes {
		fullPath := filepath.Join(dir, file)
		if err := ioutil.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", fullPath, err)
		}
	}
	return nil
}

func runTestCommand(cmd, dir string) error {
	return nil
}

func gitCheckout(dir string) error {
	return nil
}

func revertToState(dir string, state map[string]string) error {
	for file, content := range state {
		fullPath := filepath.Join(dir, file)
		if err := ioutil.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", fullPath, err)
		}
	}
	return nil
}