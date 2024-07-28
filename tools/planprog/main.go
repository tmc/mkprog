package main

import (
	"bufio"
	"context"
	_ "embed"
	"fmt"
	"os"
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
	ctx := context.Background()

	client, err := anthropic.New()
	if err != nil {
		return fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	fmt.Println("Welcome to PlanProg! Let's work on defining your program.")
	fmt.Println("Please provide a brief description of the program you want to plan:")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	initialDescription := scanner.Text()

	for {
		messages := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
			llms.TextParts(llms.ChatMessageTypeHuman, initialDescription),
		}

		resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(4000))
		if err != nil {
			return fmt.Errorf("failed to generate content: %w", err)
		}

		fmt.Println("\nAI Feedback:")
		fmt.Println(resp.Choices[0].Content)

		fmt.Println("\nDo you want to provide more details or clarify anything? (yes/no)")
		scanner.Scan()
		answer := strings.ToLower(scanner.Text())

		if answer != "yes" {
			break
		}

		fmt.Println("Please provide additional information or clarification:")
		scanner.Scan()
		additionalInfo := scanner.Text()
		initialDescription += "\n\nAdditional information: " + additionalInfo
	}

	fmt.Println("\nGreat! Your program description is now sufficiently defined. You can use this as a starting point for your development process.")

	return nil
}
