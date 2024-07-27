package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
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
	temperature := flag.Float64("temp", 0.1, "Set the temperature for AI generation (0.0 to 1.0)")
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		return fmt.Errorf("usage: %s <output directory> <program description>", os.Args[0])
	}

	outputDir := args[0]
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	description := strings.Join(args[1:], " ")

	tools, err := listTools(".")
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	ctx := context.Background()
	client, err := anthropic.New()
	if err != nil {
		return fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	toolsContext := fmt.Sprintf("Available tools:\n%s", strings.Join(tools, "\n"))
	prompt := fmt.Sprintf("%s\n\n%s", description, toolsContext)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(*temperature), llms.WithMaxTokens(4000))
	if err != nil {
		return fmt.Errorf("failed to generate content: %w", err)
	}

	if err := writeOutput(outputDir, resp.Choices[0].Content); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	fmt.Printf("Program generation complete. Output directory: %s\n", outputDir)
	fmt.Printf("\nUsage:\n")
	fmt.Printf("cd %s\n", outputDir)
	fmt.Printf("go mod tidy; go run .\n\n")
	fmt.Printf("Optional: go install\n")
	fmt.Printf("Then run: %s\n", filepath.Base(outputDir))

	return nil
}

func listTools(dir string) ([]string, error) {
	var tools []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && isExecutable(info.Mode()) {
			relPath, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			tools = append(tools, relPath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tools, nil
}

func isExecutable(mode os.FileMode) bool {
	return mode&0111 != 0
}

func writeOutput(outputDir, content string) error {
	lines := strings.Split(content, "\n")
	var currentFile *os.File
	var currentFileName string

	for _, line := range lines {
		if strings.HasPrefix(line, "=== ") && strings.HasSuffix(line, " ===") {
			if currentFile != nil {
				currentFile.Close()
			}
			fileName := strings.TrimPrefix(strings.TrimSuffix(line, " ==="), "=== ")
			filePath := filepath.Join(outputDir, fileName)
			var err error
			currentFile, err = os.Create(filePath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", filePath, err)
			}
			currentFileName = fileName
			fmt.Printf("Creating file: %s\n", filePath)
		} else if currentFile != nil {
			_, err := currentFile.WriteString(line + "\n")
			if err != nil {
				return fmt.Errorf("failed to write to file %s: %w", currentFileName, err)
			}
		}
	}

	if currentFile != nil {
		currentFile.Close()
	}

	return nil
}
