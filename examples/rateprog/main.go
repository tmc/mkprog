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
	programPath := flag.String("program", "", "Path to the program to evaluate")
	flag.Parse()

	if *programPath == "" {
		return fmt.Errorf("program path is required")
	}

	rules, err := getRules(*programPath)
	if err != nil {
		return fmt.Errorf("failed to get rules: %w", err)
	}

	programContent, err := os.ReadFile(*programPath)
	if err != nil {
		return fmt.Errorf("failed to read program file: %w", err)
	}

	evaluation, err := evaluateProgram(string(programContent), rules)
	if err != nil {
		return fmt.Errorf("failed to evaluate program: %w", err)
	}

	fmt.Println(evaluation)
	return nil
}

func getRules(programPath string) (string, error) {
	defaultRules := `
1. Follow idiomatic Go practices
2. Use proper error handling
3. Include appropriate comments
4. Maintain consistent code formatting
5. Use meaningful variable and function names
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
