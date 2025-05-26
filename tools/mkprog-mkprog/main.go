package main

import (
	"context"
	_ "embed"
	"flag"
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
	var (
		outputFile  string
		temperature float64
		verbose     bool
	)

	flag.StringVar(&outputFile, "o", "-", "Output file for the generated system prompt (use - for stdout)")
	flag.Float64Var(&temperature, "temp", 0.1, "Set the temperature for AI generation (0.0 to 1.0)")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		return fmt.Errorf("usage: %s [flags] <description of the mkprog tool to create>", os.Args[0])
	}

	ctx := context.Background()
	llm, err := anthropic.New(
		anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize language model: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Generating mkprog system prompt for: %s\n", strings.Join(args, " "))
	}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, strings.Join(args, " ")),
	}

	resp, err := llm.GenerateContent(ctx,
		messages,
		llms.WithTemperature(temperature),
		llms.WithMaxTokens(8000),
	)
	if err != nil {
		return fmt.Errorf("content generation failed: %w", err)
	}

	output := resp.GetContent()

	if outputFile == "-" {
		fmt.Print(output)
	} else {
		if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output to file %s: %w", outputFile, err)
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "System prompt written to: %s\n", outputFile)
		}
	}

	return nil
}