package main

import (
	"bytes"
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

	ctx := context.Background()
	llm, err := anthropic.New()
	if err != nil {
		return fmt.Errorf("failed to initialize language model: %w", err)
	}

	fw := &fileWriter{outputDir: outputDir}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, strings.Join(args, " ")),
	}

	_, err = llm.GenerateContent(ctx,
		messages,
		llms.WithTemperature(*temperature),
		llms.WithMaxTokens(4000),
		llms.WithStreamingFunc(fw.streamContent),
	)

	if err != nil {
		return fmt.Errorf("content generation failed: %w", err)
	}

	if err := fw.close(); err != nil {
		return fmt.Errorf("failed to close last file: %w", err)
	}

	fmt.Printf("Program generation complete. Output directory: %s\n", outputDir)
	fmt.Printf("\nUsage:\n")
	fmt.Printf("cd %s\n", outputDir)
	fmt.Printf("go mod tidy; go run .\n\n")
	fmt.Printf("Optional: go install\n")
	fmt.Printf("Then run: %s\n", filepath.Base(outputDir))
	return nil
}

var fileNameRe = regexp.MustCompile(`(?m)^=== (.*) ===$`)

type fileWriter struct {
	currentFile *os.File
	buffer      bytes.Buffer
	outputDir   string
}

func (fw *fileWriter) streamContent(ctx context.Context, chunk []byte) error {
	fw.buffer.Write(chunk)

	for {
		line, err := fw.buffer.ReadBytes('\n')
		if err != nil {
			// If we don't have a full line, put it back in the buffer and wait for more data
			fw.buffer.Write(line)
			break
		}

		if match := fileNameRe.FindSubmatch(line); match != nil {
			// We found a new file header
			if fw.currentFile != nil {
				if err := fw.currentFile.Close(); err != nil {
					return fmt.Errorf("failed to close file: %w", err)
				}
			}

			fileName := string(match[1])
			fullPath := filepath.Join(fw.outputDir, fileName)
			fw.currentFile, err = os.Create(fullPath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", fullPath, err)
			}
			fmt.Printf("Creating file: %s\n", fullPath)
		} else if fw.currentFile != nil {
			// Write the line to the current file
			if _, err := fw.currentFile.Write(line); err != nil {
				return fmt.Errorf("failed to write to file: %w", err)
			}
		}
	}

	return nil
}

func (fw *fileWriter) close() error {
	if fw.currentFile != nil {
		// Write any remaining content in the buffer
		if _, err := fw.currentFile.Write(fw.buffer.Bytes()); err != nil {
			return fmt.Errorf("failed to write final content: %w", err)
		}
		if err := fw.currentFile.Close(); err != nil {
			return fmt.Errorf("failed to close final file: %w", err)
		}
		fw.currentFile = nil
		fw.buffer.Reset()
	}
	return nil
}
