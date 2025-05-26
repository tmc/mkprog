package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

//go:embed system-prompt.txt
var systemPrompt string

// Version of mkprog
const Version = "0.2.0"

// ProgramMetadata contains information about a generated program
type ProgramMetadata struct {
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	SystemPrompt    string    `json:"system_prompt"`
	UserPrompt      string    `json:"user_prompt"`
	GenerationTime  time.Time `json:"generation_time"`
	GeneratedBy     string    `json:"generated_by"`
	GeneratorPath   string    `json:"generator_path,omitempty"`
	ToolsAvailable  []string  `json:"tools_available,omitempty"`
	MkprogVersion   string    `json:"mkprog_version"`
}

// DiscoverTools finds available tools in the mkprog toolkit
func discoverTools() []string {
	var tools []string
	toolsDir := filepath.Join(filepath.Dir(os.Args[0]), "tools")
	
	// Try to find tools in the standard locations
	possibleToolsDirs := []string{
		toolsDir,
		"/usr/local/bin",
		"/usr/bin",
		os.Getenv("GOPATH") + "/bin",
	}
	
	// Add current directory's parent if binary is in tools directory
	binDir := filepath.Dir(os.Args[0])
	if strings.Contains(binDir, "tools") {
		possibleToolsDirs = append(possibleToolsDirs, filepath.Dir(binDir))
	}
	
	for _, dir := range possibleToolsDirs {
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		
		for _, file := range files {
			if strings.HasPrefix(file.Name(), "mkprog") || strings.Contains(file.Name(), "prog") {
				if file.Name() != "mkprog" && file.Name() != filepath.Base(os.Args[0]) {
					// Check if it's executable
					path := filepath.Join(dir, file.Name())
					info, err := os.Stat(path)
					if err == nil && info.Mode()&0111 != 0 {
						tools = append(tools, file.Name())
					}
				}
			}
		}
	}
	
	return tools
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	temperature := flag.Float64("temp", 0.1, "Set the temperature for AI generation (0.0 to 1.0)")
	embedPrompt := flag.Bool("embed-prompt", true, "Embed the generation prompt in a PROMPT.md file")
	embedMetadata := flag.Bool("embed-metadata", true, "Embed program metadata in a .mkprog.json file")
	listTools := flag.Bool("list-tools", false, "List available tools in the mkprog ecosystem")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()
	
	// Check for version flag
	if *showVersion {
		fmt.Printf("mkprog version %s\n", Version)
		return nil
	}
	
	// Discover available tools
	availableTools := discoverTools()
	
	// Check for list-tools flag
	if *listTools {
		fmt.Println("Available mkprog tools:")
		for _, tool := range availableTools {
			fmt.Printf("  %s\n", tool)
		}
		return nil
	}

	args := flag.Args()
	if len(args) < 2 {
		return fmt.Errorf("usage: %s <output directory> <program description>", os.Args[0])
	}

	outputDir := args[0]
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Extract the program description from args
	programDescription := strings.Join(args[1:], " ")

	ctx := context.Background()
	llm, err := anthropic.New(
		anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize language model: %w", err)
	}

	fw := &fileWriter{outputDir: outputDir}
	
	// Create metadata
	metadata := ProgramMetadata{
		Name:           filepath.Base(outputDir),
		Description:    programDescription,
		SystemPrompt:   systemPrompt,
		UserPrompt:     programDescription,
		GenerationTime: time.Now(),
		GeneratedBy:    "mkprog",
		GeneratorPath:  os.Args[0],
		ToolsAvailable: availableTools,
		MkprogVersion:  Version,
	}

	// Create PROMPT.md if requested
	if *embedPrompt {
		promptPath := filepath.Join(outputDir, "PROMPT.md")
		promptContent := fmt.Sprintf("# Generation Prompt\n\nThis program was generated using the following prompt:\n\n```\n%s\n```\n\n## System Prompt\n\n```\n%s\n```", programDescription, systemPrompt)
		if err := os.WriteFile(promptPath, []byte(promptContent), 0644); err != nil {
			return fmt.Errorf("failed to write prompt file: %w", err)
		}
		fmt.Printf("Created prompt file: %s\n", promptPath)
	}
	
	// Create metadata file if requested
	if *embedMetadata {
		metadataPath := filepath.Join(outputDir, ".mkprog.json")
		metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		if err := os.WriteFile(metadataPath, metadataJSON, 0644); err != nil {
			return fmt.Errorf("failed to write metadata file: %w", err)
		}
		fmt.Printf("Created metadata file: %s\n", metadataPath)
	}
	
	// Add commands for self-introspection to the system prompt
	enhancedSystemPrompt := systemPrompt + "\n\n" +
		"IMPORTANT: This program should support the following self-introspection flags:\n" +
		"- --show-prompt: Output the prompt used to generate this program\n" +
		"- --show-source: Output the source code of a specific file or all files\n" +
		"- --mkprog-info: Show information about the mkprog ecosystem\n\n" +
		"Available mkprog tools that can be used with this program:\n" +
		strings.Join(availableTools, ", ") + "\n"

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, enhancedSystemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, programDescription),
	}

	_, err = llm.GenerateContent(ctx,
		messages,
		llms.WithTemperature(*temperature),
		llms.WithMaxTokens(8000),
		llms.WithStreamingFunc(fw.streamContent),
	)

	if err != nil {
		return fmt.Errorf("content generation failed: %w", err)
	}

	if err := fw.close(); err != nil {
		return fmt.Errorf("failed to close last file: %w", err)
	}

	// run goimports if it's available (lookpath)
	if err := runGoImports(outputDir); err != nil {
		return fmt.Errorf("failed to run goimports: %w", err)
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

func runGoImports(dir string) error {
	_, err := exec.LookPath("goimports")
	if err != nil {
		fmt.Println("goimports not found, skipping...")
		return nil
	}
	fmt.Println("Running goimports...")
	cmd := exec.Command("goimports", "-w", dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
