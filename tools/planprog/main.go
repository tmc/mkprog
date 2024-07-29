package main

import (
	"bufio"
	"context"
	_ "embed"
	"fmt"
	"log"
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

	fmt.Println("Welcome to planprog! Let's work on defining your program.")
	fmt.Println("Please provide a brief description of the program you want to plan:")
	fmt.Println("You must send EOF (Ctrl+D) to finish.")

	scanner := bufio.NewScanner(os.Stdin)
	var initialDescription string
	for scanner.Scan() {
		initialDescription += scanner.Text() + "\n"
	}
	initialDescription = strings.TrimSpace(initialDescription)
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(os.Stderr, "Program description recieved, working on enhancing it...\n")

	// TODO: consider interactive (or non-interactive) refinement loop.
	for i := 0; i < 1; i++ {
		messages := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
			llms.TextParts(llms.ChatMessageTypeHuman, initialDescription),
		}

		resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(4000))
		if err != nil {
			return fmt.Errorf("failed to generate content: %w", err)
		}

		fmt.Println("\nImproved program description:")
		fmt.Println(resp.Choices[0].Content)

		fmt.Println("Please provide additional information or clarification:")
		scanner.Scan()
		additionalInfo := scanner.Text()
		initialDescription += "\n\nAdditional information: " + additionalInfo
	}
	fmt.Println("\nGreat! Your program description is ready.")

	return nil
}
