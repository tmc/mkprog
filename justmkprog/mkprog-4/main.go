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
		temperature  = flag.Float64("temp", 0.1, "Temperature for AI generation")
		maxTokens    = flag.Int("max-tokens", 8192, "Maximum number of tokens for AI generation")
		verbose      = flag.Bool("verbose", false, "Enable verbose logging")
		inputFile    = flag.String("f", "-", "Input file (use '-' for stdin)")
		outputDir    = flag.String("o", "", "Output directory for generated files")
		runGoimports = flag.Bool("goimports", false, "Run goimports on generated Go files")
	)
	flag.Parse()

	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetOutput(io.Discard)
	}

	if len(flag.Args()) < 2 {
		return fmt.Errorf("usage: %s [flags] <program-name> <program-description>", os.Args[0])
	}

	programName := flag.Arg(0)
	programDesc := flag.Arg(1)

	input, err := readInput(*inputFile)
	if err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	client, err := anthropic.New()
	if err != nil {
		return fmt.Errorf("error creating Anthropic client: %w", err)
	}

	ctx := context.Background()
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("%s %s", programName, programDesc)),
	}

	resp, err := client.GenerateContent(ctx, messages,
		llms.WithTemperature(float32(*temperature)),
		llms.WithMaxTokens(*maxTokens),
		anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35),
	)
	if err != nil {
		return fmt.Errorf("error generating content: %w", err)
	}

	if *outputDir == "" {
		*outputDir = programName
	}
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	if err := writeFiles(*outputDir, resp.Choices[0].Content, *runGoimports); err != nil {
		return fmt.Errorf("error writing files: %w", err)
	}

	fmt.Printf("Generated program '%s' in directory: %s\n", programName, *outputDir)
	return nil
}

func readInput(filename string) (string, error) {
	if filename == "-" {
		return "", nil // No additional input needed for this program
	}
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func writeFiles(outputDir, content string, runGoimports bool) error {
	var currentFile strings.Builder
	var currentFilename string

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "=== ") && strings.HasSuffix(line, " ===") {
			if currentFilename != "" {
				if err := writeFile(outputDir, currentFilename, currentFile.String(), runGoimports); err != nil {
					return err
				}
				currentFile.Reset()
			}
			currentFilename = strings.TrimSuffix(strings.TrimPrefix(line, "=== "), " ===")
		} else {
			currentFile.WriteString(line)
			currentFile.WriteString("\n")
		}
	}

	if currentFilename != "" {
		if err := writeFile(outputDir, currentFilename, currentFile.String(), runGoimports); err != nil {
			return err
		}
	}

	return scanner.Err()
}

func writeFile(outputDir, filename, content string, runGoimports bool) error {
	filepath := filepath.Join(outputDir, filename)
	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		return fmt.Errorf("error writing file %s: %w", filename, err)
	}

	if runGoimports && strings.HasSuffix(filename, ".go") {
		cmd := exec.Command("goimports", "-w", filepath)
		if err := cmd.Run(); err != nil {
			log.Printf("Warning: goimports failed on %s: %v", filename, err)
		}
	}

	return nil
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
