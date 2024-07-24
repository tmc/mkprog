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
	"github.com/tmc/langchaingo/llms/openai"
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
	programPath := flag.String("program", ".", "Path to the program to evaluate (defaults to current directory)")
	flag.Parse()

	if *programPath == "." {
		var err error
		*programPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}
	}

	rules, err := getRules(*programPath)
	if err != nil {
		return fmt.Errorf("failed to get rules: %w", err)
	}

	programContent, err := readProgramContent(*programPath)
	if err != nil {
		return fmt.Errorf("failed to read program content: %w", err)
	}

	evaluation, err := evaluateProgram(programContent, rules)
	if err != nil {
		return fmt.Errorf("failed to evaluate program: %w", err)
	}

	fmt.Println(evaluation)
	return nil
}

func getRules(programPath string) (string, error) {
	defaultRules := `
1. Be unix in style and substance
2. Use proper error handling
4. Maintain consistent code formatting
`

	ruleFile := findRuleFile(programPath)
	if ruleFile == "" {
		return defaultRules, nil
	}

	userRules, err := os.ReadFile(ruleFile)
	if err != nil {
		return "", fmt.Errorf("failed to read rule file: %w", err)
	}

	return string(userRules) + defaultRules, nil
}

func findRuleFile(startPath string) string {
	dir := filepath.Dir(startPath)
	for {
		ruleFile := filepath.Join(dir, ".rateprog-rules")
		if _, err := os.Stat(ruleFile); err == nil {
			return ruleFile
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func readProgramContent(programPath string) (string, error) {
	fileInfo, err := os.Stat(programPath)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	if fileInfo.IsDir() {
		return readDirectoryContent(programPath)
	}

	content, err := os.ReadFile(programPath)
	if err != nil {
		return "", fmt.Errorf("failed to read program file: %w", err)
	}

	return string(content), nil
}

func readDirectoryContent(dirPath string) (string, error) {
	var content strings.Builder

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			fileContent, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", path, err)
			}

			content.WriteString(fmt.Sprintf("// File: %s\n", path))
			content.Write(fileContent)
			content.WriteString("\n\n")
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to read directory content: %w", err)
	}

	return content.String(), nil
}

func evaluateProgram(programContent, rules string) (string, error) {
	ctx := context.Background()
	client, err := openai.New()
	if err != nil {
		return "", fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	prompt := fmt.Sprintf("Evaluate the following Go program based on these rules:\n\n%s\n\nProgram:\n\n%s", rules, programContent)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(4000))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	var evaluation strings.Builder
	evaluation.WriteString(resp.Choices[0].Content)

	return evaluation.String(), nil
}
