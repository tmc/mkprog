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
		temperature  = flag.Float64("temp", 0.1, "Temperature for AI generation")
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

	if *verbose {
		log.Printf("Generating content for program: %s\nDescription: %s", programName, programDesc)
	}

	userInput := fmt.Sprintf("%s %q", programName, programDesc)
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, userInput),
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
		log.Println("Content generation completed")
	}

	if *outputDir != "" {
		if err := os.MkdirAll(*outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
		if err := writeFiles(*outputDir, resp.Choices[0].Content, *runGoimports); err != nil {
			return fmt.Errorf("failed to write files: %w", err)
		}
		if *verbose {
			log.Printf("Files written to %s", *outputDir)
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
	input, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(input), nil
}

func writeFiles(outputDir, content string, runGoimports bool) error {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var currentFile *os.File
	var currentFileName string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "=== ") && strings.HasSuffix(line, " ===") {
			if currentFile != nil {
				currentFile.Close()
			}
			currentFileName = strings.TrimSuffix(strings.TrimPrefix(line, "=== "), " ===")
			filePath := filepath.Join(outputDir, currentFileName)
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

	if runGoimports {
		if err := runGoimportsOnFiles(outputDir); err != nil {
			return fmt.Errorf("failed to run goimports: %w", err)
		}
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
