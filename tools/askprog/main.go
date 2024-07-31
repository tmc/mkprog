package main

import (
	"context"
	_ "embed"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: %s <question>", os.Args[0])
	}

	question := strings.Join(os.Args[1:], " ")
	codebase, err := collectCodebase(".")
	if err != nil {
		return fmt.Errorf("failed to collect codebase: %w", err)
	}

	ctx := context.Background()
	client, err := anthropic.New()
	if err != nil {
		return fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Codebase:\n\n%s\n\nQuestion: %s", codebase, question)),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(4000))
	if err != nil {
		return fmt.Errorf("failed to generate content: %w", err)
	}

	fmt.Println(resp.Choices[0].Content)
	return nil
}

func collectCodebase(root string) (string, error) {
	var codebase strings.Builder

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && (info.Name() == ".git" || info.Name() == "vendor") {
			return filepath.SkipDir
		}

		if !info.IsDir() && isSourceFile(info.Name()) {
			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			codebase.WriteString(fmt.Sprintf("File: %s\n\n", path))
			codebase.Write(content)
			codebase.WriteString("\n\n")
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return codebase.String(), nil
}

func isSourceFile(filename string) bool {
	validExtensions := map[string]bool{
		".go":   true,
		".js":   true,
		".py":   true,
		".java": true,
		".c":    true,
		".cpp":  true,
		".h":    true,
		".md":   true,
	}
	return validExtensions[filepath.Ext(filename)]
}
