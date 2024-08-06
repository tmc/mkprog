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
		log.Fatalf("Error: %v", err)
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

	if flag.NArg() < 2 {
		return fmt.Errorf("usage: %s [flags] <program-name> <program-description>", os.Args[0])
	}

	programName := flag.Arg(0)
	programDesc := flag.Arg(1)

	input, err := readInput(*inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	ctx := context.Background()
	client, err := anthropic.New(anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35))
	if err != nil {
		return fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	userInput := fmt.Sprintf("%s %s\n\n%s", programName, programDesc, input)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, userInput),
	}

	if *verbose {
		log.Printf("Generating content with temperature: %.2f, max tokens: %d", *temperature, *maxTokens)
	}

	resp, err := client.GenerateContent(ctx, messages,
		llms.WithTemperature(*temperature),
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
		log.Println("Content generation complete")
	}

	if err := writeFiles(resp.Choices[0].Content, *outputDir); err != nil {
		return fmt.Errorf("failed to write files: %w", err)
	}

	if *runGoimports {
		if err := runGoimportsOnFiles(*outputDir); err != nil {
			return fmt.Errorf("failed to run goimports: %w", err)
		}
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

	var builder strings.Builder
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		builder.WriteString(scanner.Text())
		builder.WriteString("\n")
	}
	return builder.String(), scanner.Err()
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
			currentFileName = strings.TrimPrefix(strings.TrimSuffix(line, " ==="), "=== ")
			filePath := filepath.Join(outputDir, currentFileName)
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", filePath, err)
			}
			var err error
			currentFile, err = os.Create(filePath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", filePath, err)
			}
		} else if currentFile != nil {
			fmt.Fprintln(currentFile, line)
		}
	}

	if currentFile != nil {
		currentFile.Close()
	}

	return nil
}

func runGoimportsOnFiles(dir string) error {
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
		data, _ := os.ReadFile("main.go")
		fmt.Println(string(data))
		fmt.Println("=== go.mod ===")
		data, _ = os.ReadFile("go.mod")
		fmt.Println(string(data))
		fmt.Println("=== system-prompt.txt ===")
		data, _ = os.ReadFile("system-prompt.txt")
		fmt.Println(string(data))
	}
}
