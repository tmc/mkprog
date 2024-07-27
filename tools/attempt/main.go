package main

import (
	"context"
	"encoding/json"
	_ "embed"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

//go:embed system-prompt.txt
var systemPrompt string

type Attempt struct {
	ID     int    `json:"id"`
	Goal   string `json:"goal"`
	Tools  string `json:"tools"`
	Output string `json:"output"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 4 {
		return fmt.Errorf("usage: %s <goal> <tools> <num_attempts>", os.Args[0])
	}

	goal := os.Args[1]
	tools := os.Args[2]
	numAttempts, err := strconv.Atoi(os.Args[3])
	if err != nil {
		return fmt.Errorf("invalid number of attempts: %v", err)
	}

	workDir, err := ioutil.TempDir("", "attempt-")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(workDir)

	repo, err := git.PlainInit(workDir, false)
	if err != nil {
		return fmt.Errorf("failed to initialize git repository: %v", err)
	}

	for i := 1; i <= numAttempts; i++ {
		attempt := Attempt{
			ID:    i,
			Goal:  goal,
			Tools: tools,
		}

		output, err := executeAttempt(workDir, attempt)
		if err != nil {
			return fmt.Errorf("failed to execute attempt %d: %v", i, err)
		}

		attempt.Output = output
		if err := saveAttempt(workDir, attempt); err != nil {
			return fmt.Errorf("failed to save attempt %d: %v", i, err)
		}

		if err := commitAttempt(repo, attempt); err != nil {
			return fmt.Errorf("failed to commit attempt %d: %v", i, err)
		}
	}

	fmt.Printf("Completed %d attempts. Results stored in %s\n", numAttempts, workDir)
	return nil
}

func executeAttempt(workDir string, attempt Attempt) (string, error) {
	ctx := context.Background()
	client, err := anthropic.New()
	if err != nil {
		return "", fmt.Errorf("failed to create Anthropic client: %v", err)
	}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Goal: %s\nTools: %s\nAttempt: %d", attempt.Goal, attempt.Tools, attempt.ID)),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.7), llms.WithMaxTokens(2000))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %v", err)
	}

	return resp.Choices[0].Content, nil
}

func saveAttempt(workDir string, attempt Attempt) error {
	filename := filepath.Join(workDir, fmt.Sprintf("attempt_%d.json", attempt.ID))
	data, err := json.MarshalIndent(attempt, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal attempt data: %v", err)
	}

	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write attempt file: %v", err)
	}

	return nil
}

func commitAttempt(repo *git.Repository, attempt Attempt) error {
	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %v", err)
	}

	filename := fmt.Sprintf("attempt_%d.json", attempt.ID)
	if _, err := w.Add(filename); err != nil {
		return fmt.Errorf("failed to stage file: %v", err)
	}

	commitMsg := fmt.Sprintf("Attempt %d: %s", attempt.ID, truncate(attempt.Goal, 50))
	_, err = w.Commit(commitMsg, &git.CommitOptions{})
	if err != nil {
		return fmt.Errorf("failed to commit: %v", err)
	}

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

