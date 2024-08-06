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
		log.Printf("Run goimports: %v", *runGoimports)
	}

	input, err := readInput(*inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	client, err := anthropic.New()
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
		anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35),
	)
	if err != nil {
		return fmt.Errorf("failed to generate content: %w", err)
	}

	if *outputDir == "" {
		*outputDir = "generated_program"
	}
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := writeGeneratedFiles(*outputDir, resp.Choices[0].Content); err != nil {
		return fmt.Errorf("failed to write generated files: %w", err)
	}

	if *runGoimports {
		if err := runGoimportsOnDir(*outputDir); err != nil {
			return fmt.Errorf("failed to run goimports: %w", err)
		}
	}

	fmt.Printf("Program generated successfully in %s\n", *outputDir)
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

func writeGeneratedFiles(outputDir, content string) error {
	lines := strings.Split(content, "\n")
	var currentFile *os.File
	var currentFileName string

	for _, line := range lines {
		if strings.HasPrefix(line, "=== ") && strings.HasSuffix(line, " ===") {
			if currentFile != nil {
				currentFile.Close()
			}
			currentFileName = strings.TrimPrefix(strings.TrimSuffix(line, " ==="), "=== ")
			filePath := filepath.Join(outputDir, currentFileName)
			var err error
			currentFile, err = os.Create(filePath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", filePath, err)
			}
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

func runGoimportsOnDir(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			cmd := exec.Command("goimports", "-w", path)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("goimports failed on %s: %w", path, err)
			}
		}
		return nil
	})
}

func dumpsrc() {
	if os.Getenv("_MKPROG_DUMP") != "" {
		fmt.Println("=== main.go ===")
		content, _ := os.ReadFile("main.go")
		fmt.Println(string(content))

		fmt.Println("=== go.mod ===")
		content, _ = os.ReadFile("go.mod")
		fmt.Println(string(content))

		fmt.Println("=== system-prompt.txt ===")
		content, _ = os.ReadFile("system-prompt.txt")
		fmt.Println(string(content))
	}
}
