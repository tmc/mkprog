package main

import (
	"context"
	_ "embed"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
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
	repoPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open git repository: %w", err)
	}

	commits, err := getCommitHistory(repo)
	if err != nil {
		return fmt.Errorf("failed to get commit history: %w", err)
	}

	contextFiles, err := findContextFiles(repoPath)
	if err != nil {
		return fmt.Errorf("failed to find context files: %w", err)
	}

	guidance, err := generateGuidance(commits, contextFiles)
	if err != nil {
		return fmt.Errorf("failed to generate guidance: %w", err)
	}

	fmt.Println(guidance)
	return nil
}

func getCommitHistory(repo *git.Repository) ([]*object.Commit, error) {
	ref, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	commitIter, err := repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, fmt.Errorf("failed to get commit iterator: %w", err)
	}

	var commits []*object.Commit
	err = commitIter.ForEach(func(c *object.Commit) error {
		commits = append(commits, c)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate over commits: %w", err)
	}

	return commits, nil
}

func findContextFiles(startPath string) ([]string, error) {
	var contextFiles []string
	for path := startPath; path != "/"; path = filepath.Dir(path) {
		contextDir := filepath.Join(path, ".git-commit-style")
		files, err := ioutil.ReadDir(contextDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read context directory: %w", err)
		}
		for _, file := range files {
			if !file.IsDir() {
				contextFiles = append(contextFiles, filepath.Join(contextDir, file.Name()))
			}
		}
		if len(contextFiles) > 0 {
			break
		}
	}
	return contextFiles, nil
}

func generateGuidance(commits []*object.Commit, contextFiles []string) (string, error) {
	ctx := context.Background()
	client, err := anthropic.New()
	if err != nil {
		return "", fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	var commitInfo strings.Builder
	for _, commit := range commits {
		commitInfo.WriteString(fmt.Sprintf("Commit: %s\nAuthor: %s\nDate: %s\nMessage: %s\n\n",
			commit.Hash, commit.Author.Name, commit.Author.When, commit.Message))
	}

	var contextContent strings.Builder
	for _, file := range contextFiles {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("failed to read context file %s: %w", file, err)
		}
		contextContent.WriteString(fmt.Sprintf("File: %s\nContent:\n%s\n\n", file, string(content)))
	}

	userInput := fmt.Sprintf("Commit history:\n%s\nContext files:\n%s\nPlease provide guidance for this repository based on the commit history and context files.", commitInfo.String(), contextContent.String())

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, userInput),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(4000))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	return resp.Choices[0].Content, nil
}

