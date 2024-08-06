package main

import (
	"bufio"
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

	if *outputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	input, err := readInput(*inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	ctx := context.Background()
	client, err := anthropic.New(ctx, anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35))
	if err != nil {
		return fmt.Errorf("failed to create Anthropic client: %w", err)
	}

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

	if err := writeFiles(*outputDir, resp.Choices[0].Content); err != nil {
		return fmt.Errorf("failed to write files: %w", err)
	}

	if *runGoimports {
		if err := runGoimportsOnDir(*outputDir); err != nil {
			return fmt.Errorf("failed to run goimports: %w", err)
		}
	}

	if *verbose {
		log.Println("Program generation completed successfully")
	}

	return nil
}

func readInput(filename string) (string, error) {
	var reader io.Reader
	if filename == "-" {
		reader = os.Stdin
	} else {
		file, err := os.Open(filename)
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

func writeFiles(outputDir, content string) error {
	var currentFile strings.Builder
	var currentFileName string

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "=== ") && strings.HasSuffix(line, " ===") {
			if currentFileName != "" {
				if err := writeFile(outputDir, currentFileName, currentFile.String()); err != nil {
					return err
				}
				currentFile.Reset()
			}
			currentFileName = strings.TrimPrefix(strings.TrimSuffix(line, " ==="), "=== ")
		} else {
			currentFile.WriteString(line)
			currentFile.WriteString("\n")
		}
	}

	if currentFileName != "" {
		if err := writeFile(outputDir, currentFileName, currentFile.String()); err != nil {
			return err
		}
	}

	return scanner.Err()
}

func writeFile(outputDir, fileName, content string) error {
	filePath := filepath.Join(outputDir, fileName)
	return os.WriteFile(filePath, []byte(content), 0644)
}

func runGoimportsOnDir(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
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
