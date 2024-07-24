package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Println("🚀 Starting the development planning process...")

	goals, err := readFile("goals")
	if err != nil {
		return fmt.Errorf("failed to read goals: %w", err)
	}
	fmt.Println("📋 Goals loaded successfully!")

	tools, err := listTools(".")
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}
	fmt.Printf("🛠️  Found %d available tools\n", len(tools))

	history, err := readFile("hist")
	if err != nil {
		return fmt.Errorf("failed to read history: %w", err)
	}
	fmt.Println("📜 Development history retrieved")

	todos, err := readFile("todos")
	if err != nil {
		return fmt.Errorf("failed to read todos: %w", err)
	}
	fmt.Printf("📝 Loaded %d todo items\n", len(todos))

	fmt.Println("🤔 Analyzing project context and planning next action...")
	action, err := planNextAction(goals, tools, history, todos)
	if err != nil {
		return fmt.Errorf("failed to plan next action: %w", err)
	}

	fmt.Println("✨ AI Assistant suggests the following next action:")
	fmt.Printf("👉 %s\n", action)
	return nil
}

func readFile(filename string) ([]string, error) {
	content, err := findAndReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %w", filename, err)
	}
	return strings.Split(content, "\n"), nil
}

func findAndReadFile(filename string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	fmt.Printf("🔍 Searching for %s file...\n", filename)
	for {
		filePath := filepath.Join(dir, filename)
		content, err := os.ReadFile(filePath)
		if err == nil {
			fmt.Printf("📂 Found %s in %s\n", filename, dir)
			return string(content), nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("%s file not found in any parent directory", filename)
		}
		dir = parent
	}
}

func listTools(dir string) ([]string, error) {
	var tools []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	fmt.Println("🔧 Scanning for available tools...")
	for _, entry := range entries {
		if !entry.IsDir() && isExecutable(entry) {
			tools = append(tools, entry.Name())
			fmt.Printf("  - %s\n", entry.Name())
		}
	}
	return tools, nil
}

func isExecutable(entry os.DirEntry) bool {
	info, err := entry.Info()
	if err != nil {
		return false
	}
	return info.Mode()&0111 != 0 || strings.HasSuffix(entry.Name(), "prog") || entry.Name() == "plan"
}

func planNextAction(goals, tools, history, todos []string) (string, error) {
	ctx := context.Background()
	fmt.Println("🤖 Connecting to AI assistant...")
	client, err := anthropic.New()
	if err != nil {
		return "", fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	prompt := fmt.Sprintf(`Based on the following information, suggest the next action to take in the development process:

Goals:
%s

Available tools:
%s

Recent history:
%s

Current todos:
%s

Provide a single, specific command to run or action to take as the next step. Be concise and direct.`, goals, tools, history, todos)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, "You are an AI assistant helping to plan the next action in a software development project. Suggest the most appropriate next step based on the provided context."),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	fmt.Println("💭 AI assistant is thinking...")
	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.2), llms.WithMaxTokens(100))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	return strings.TrimSpace(resp.Choices[0].Content), nil
}