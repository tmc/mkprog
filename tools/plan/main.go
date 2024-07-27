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
		fmt.Println("⚠️  No goals file found. Proceeding with an empty goals list.")
		goals = []string{}
	} else {
		fmt.Println("📋 Goals loaded successfully!")
	}

	tools, err := listTools(".")
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}
	fmt.Printf("🛠️  Found %d available tools\n", len(tools))

	history, err := readFile("hist")
	if err != nil {
		fmt.Println("⚠️  No history file found. Proceeding with an empty history.")
		history = []string{}
	} else {
		fmt.Println("📜 Development history retrieved")
	}

	todos, err := readFile("todos")
	if err != nil {
		fmt.Println("⚠️  No todos file found. Proceeding with an empty todo list.")
		todos = []string{}
	} else {
		fmt.Printf("📝 Loaded %d todo items\n", len(todos))
	}

	fmt.Println("🤔 Analyzing project context and planning action graph...")
	actionGraph, err := planActionGraph(goals, tools, history, todos)
	if err != nil {
		return fmt.Errorf("failed to plan action graph: %w", err)
	}

	fmt.Println("✨ AI Assistant suggests the following action graph:")
	fmt.Printf("%s\n", actionGraph)
	return nil
}

func readFile(filename string) ([]string, error) {
	content, err := findAndReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", filename)
		}
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

		if !os.IsNotExist(err) {
			return "", err
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

func planActionGraph(goals, tools, history, todos []string) (string, error) {
	ctx := context.Background()
	fmt.Println("🤖 Connecting to AI assistant...")
	client, err := anthropic.New()
	if err != nil {
		return "", fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	prompt := fmt.Sprintf(`Based on the following information, create a graph of actions to take in the development process:

Goals:
%s

Available tools:
%s

Recent history:
%s

Current todos:
%s

Provide a graph of actions, showing dependencies and relationships between tasks. Each node should represent a specific action or task, and edges should show the order or dependencies between actions. Use a simple text-based format to represent the graph, such as:

Action1 -> Action2
Action1 -> Action3
Action2 -> Action4
Action3 -> Action4

Be comprehensive but concise, focusing on the most important actions to achieve the goals.`, strings.Join(goals, "\n"), strings.Join(tools, "\n"), strings.Join(history, "\n"), strings.Join(todos, "\n"))

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, "You are an AI assistant helping to plan a graph of actions for a software development project. Create a comprehensive action plan based on the provided context."),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	fmt.Println("💭 AI assistant is thinking...")
	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.2), llms.WithMaxTokens(500))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	return strings.TrimSpace(resp.Choices[0].Content), nil
}
