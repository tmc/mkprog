package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

//go:embed system-prompt.txt
var systemPrompt string

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var (
		temperature  = flag.Float64("temperature", 0.1, "Temperature for AI generation")
		maxTokens    = flag.Int("max-tokens", 8192, "Maximum number of tokens for AI generation")
		verbose      = flag.Bool("verbose", false, "Enable verbose logging")
		inputFile    = flag.String("f", "-", "Input file (use '-' for stdin)")
		outputDir    = flag.String("o", "", "Output directory for generated files")
		runGoimports = flag.Bool("goimports", false, "Run goimports on generated Go files")
	)
	flag.Parse()

	if *verbose {
		log.Println("Verbose logging enabled")
		log.Printf("Temperature: %f", *temperature)
		log.Printf("Max Tokens: %d", *maxTokens)
		log.Printf("Input File: %s", *inputFile)
		log.Printf("Output Directory: %s", *outputDir)
		log.Printf("Run Goimports: %v", *runGoimports)
	}

	input, err := readInput(*inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	client, err := anthropic.New(anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35))
	if err != nil {
		return fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	ctx := context.Background()
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, input),
	}

	resp, err := client.GenerateContent(ctx, messages,
		llms.WithTemperature(float32(*temperature)),
		llms.WithMaxTokens(*maxTokens),
		llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			fmt.Print(string(chunk))
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to generate content: %w", err)
	}

	if *verbose {
		log.Println("Content generation completed")
	}

	if err := writeFiles(resp.Choices[0].Content, *outputDir); err != nil {
		return fmt.Errorf("failed to write files: %w", err)
	}

	if *runGoimports {
		if err := runGoimportsOnFiles(*outputDir); err != nil {
			return fmt.Errorf("failed to run goimports: %w", err)
		}
	}

	if *verbose {
		log.Println("Program execution completed successfully")
	}

	return nil
}

func readInput(inputFile string) (string, error) {
	var reader io.Reader
	if inputFile == "-" {
		reader = os.Stdin
	} else {
		file, err := os.Open(inputFile)
		if err != nil {
			return "", err
		}
		defer file.Close()
		reader = file
	}

	input, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(input), nil
}

func writeFiles(content string, outputDir string) error {
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

			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", filePath, err)
			}

			var err error
			currentFile, err = os.Create(filePath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", filePath, err)
			}
			currentFileName = fileName
		} else if currentFile != nil {
			if _, err := currentFile.WriteString(line + "\n"); err != nil {
				return fmt.Errorf("failed to write to file %s: %w", currentFileName, err)
			}
		}
	}

	if currentFile != nil {
		currentFile.Close()
	}

	return nil
}

func runGoimportsOnFiles(outputDir string) error {
	return filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			cmd := exec.Command("goimports", "-w", path)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to run goimports on %s: %w", path, err)
			}
		}
		return nil
	})
}

func dumpsrc() {
	files := []string{"main.go", "go.mod", "system-prompt.txt"}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", file, err)
			continue
		}
		fmt.Printf("=== %s ===\n%s\n", file, string(content))
	}
}

func init() {
	if os.Getenv("_MKPROG_DUMP") != "" {
		dumpsrc()
		os.Exit(0)
	}
}
