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
	var (
		outputDir   string
		temperature float64
		verbose     bool
	)

	flag.StringVar(&outputDir, "o", "mkprog-generated", "Output directory for the generated mkprog program")
	flag.Float64Var(&temperature, "temp", 0.1, "Set the temperature for AI generation (0.0 to 1.0)")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.Parse()

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if verbose {
		fmt.Printf("Generating mkprog quine in directory: %s\n", outputDir)
	}

	ctx := context.Background()
	llm, err := anthropic.New(
		anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize language model: %w", err)
	}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, "Generate the original mkprog program"),
	}

	fw := &fileWriter{outputDir: outputDir, verbose: verbose}

	_, err = llm.GenerateContent(ctx,
		messages,
		llms.WithTemperature(temperature),
		llms.WithMaxTokens(8000),
		llms.WithStreamingFunc(fw.streamContent),
	)
	if err != nil {
		return fmt.Errorf("content generation failed: %w", err)
	}

	if err := fw.close(); err != nil {
		return fmt.Errorf("failed to close last file: %w", err)
	}

	if verbose {
		fmt.Println("Generation complete!")
		fmt.Printf("\nTo use the generated mkprog:\n")
		fmt.Printf("cd %s\n", outputDir)
		fmt.Printf("go mod tidy; go build\n")
		fmt.Printf("./mkprog <output-dir> <program-description>\n")
	}

	return nil
}

type fileWriter struct {
	currentFile *os.File
	buffer      strings.Builder
	outputDir   string
	verbose     bool
	lineBuffer  string
}

func (fw *fileWriter) streamContent(ctx context.Context, chunk []byte) error {
	// Append the new chunk to the buffer
	fw.buffer.Write(chunk)
	content := fw.buffer.String()

	// Extract lines from the content
	lines := strings.Split(content, "\n")

	// Process all complete lines except the last one (which might be incomplete)
	for i := 0; i < len(lines)-1; i++ {
		line := lines[i]
		if err := fw.processLine(line + "\n"); err != nil {
			return err
		}
	}

	// Keep the last potentially incomplete line in the buffer
	fw.buffer.Reset()
	if len(lines) > 0 {
		fw.buffer.WriteString(lines[len(lines)-1])
	}

	return nil
}

func (fw *fileWriter) processLine(line string) error {
	// Check if this line marks a new file
	if strings.HasPrefix(line, "=== ") && strings.HasSuffix(line, " ===\n") {
		// Extract the filename
		fileName := strings.TrimPrefix(line, "=== ")
		fileName = strings.TrimSuffix(fileName, " ===\n")

		// Close current file if open
		if fw.currentFile != nil {
			if err := fw.currentFile.Close(); err != nil {
				return fmt.Errorf("failed to close file: %w", err)
			}
		}

		// Create directory structure if needed
		filePath := filepath.Join(fw.outputDir, fileName)
		dirPath := filepath.Dir(filePath)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
		}

		// Create new file
		var err error
		fw.currentFile, err = os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", filePath, err)
		}

		if fw.verbose {
			fmt.Printf("Creating file: %s\n", filePath)
		}
	} else if fw.currentFile != nil {
		// Write line to current file
		if _, err := fw.currentFile.WriteString(line); err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
	}

	return nil
}

func (fw *fileWriter) close() error {
	// Write any remaining content and close the current file
	if fw.currentFile != nil {
		remaining := fw.buffer.String()
		if remaining != "" {
			if _, err := fw.currentFile.WriteString(remaining); err != nil {
				return fmt.Errorf("failed to write final content: %w", err)
			}
		}
		if err := fw.currentFile.Close(); err != nil {
			return fmt.Errorf("failed to close final file: %w", err)
		}
		fw.currentFile = nil
		fw.buffer.Reset()
	}
	return nil
}