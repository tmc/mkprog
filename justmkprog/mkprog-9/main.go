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
		runGoimports = flag.Bool("goimports", false, "Run goimports on generated Go files")
	)
	flag.Parse()

	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetOutput(io.Discard)
	}

	if os.Getenv("_MKPROG_DUMP") != "" {
		return dumpsrc()
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

	stream, err := client.GenerateContentStream(ctx, messages,
		llms.WithTemperature(float32(*temperature)),
		llms.WithMaxTokens(*maxTokens),
	)
	if err != nil {
		return fmt.Errorf("failed to generate content: %w", err)
	}
	defer stream.Close()

	var currentFile string
	var fileContent strings.Builder
	fileWriter := &FileWriter{}

	for {
		chunk, err := stream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading from stream: %w", err)
		}

		content := chunk.Content()
		lines := strings.Split(content, "\n")

		for _, line := range lines {
			if strings.HasPrefix(line, "=== ") && strings.HasSuffix(line, " ===") {
				if currentFile != "" {
					if err := fileWriter.WriteFile(currentFile, fileContent.String()); err != nil {
						return fmt.Errorf("failed to write file %s: %w", currentFile, err)
					}
					fileContent.Reset()
				}
				currentFile = strings.TrimSuffix(strings.TrimPrefix(line, "=== "), " ===")
			} else {
				fileContent.WriteString(line + "\n")
			}
		}

		fmt.Print(content)
	}

	if currentFile != "" {
		if err := fileWriter.WriteFile(currentFile, fileContent.String()); err != nil {
			return fmt.Errorf("failed to write file %s: %w", currentFile, err)
		}
	}

	if *runGoimports {
		if err := runGoimportsOnFiles(fileWriter.Files()); err != nil {
			return fmt.Errorf("failed to run goimports: %w", err)
		}
	}

	return nil
}

func readInput(inputFile string) (string, error) {
	var input string
	var err error

	if inputFile == "-" {
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			input = scanner.Text()
		}
		err = scanner.Err()
	} else {
		var data []byte
		data, err = os.ReadFile(inputFile)
		if err == nil {
			input = string(data)
		}
	}

	if err != nil {
		return "", err
	}

	return input, nil
}

type FileWriter struct {
	files map[string]string
}

func (fw *FileWriter) WriteFile(filename, content string) error {
	if fw.files == nil {
		fw.files = make(map[string]string)
	}
	fw.files[filename] = content
	return os.WriteFile(filename, []byte(content), 0644)
}

func (fw *FileWriter) Files() []string {
	var files []string
	for file := range fw.files {
		files = append(files, file)
	}
	return files
}

func runGoimportsOnFiles(files []string) error {
	for _, file := range files {
		if filepath.Ext(file) == ".go" {
			cmd := exec.Command("goimports", "-w", file)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("goimports failed on file %s: %w", file, err)
			}
		}
	}
	return nil
}

func dumpsrc() error {
	files := []string{"main.go", "go.mod", "system-prompt.txt"}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}
		fmt.Printf("=== %s ===\n%s\n", file, content)
	}
	return nil
}
