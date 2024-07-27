package main

import (
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
	if len(os.Args) < 2 {
		return fmt.Errorf("please provide a topic for the haiku")
	}

	topic := strings.Join(os.Args[1:], " ")

	ctx := context.Background()
	client, err := anthropic.New()
	if err != nil {
		return fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Generate a haiku about: %s", topic)),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.7), llms.WithMaxTokens(100))
	if err != nil {
		return fmt.Errorf("failed to generate content: %w", err)
	}

	fmt.Println(resp.Choices[0].Content)
	return nil
}

