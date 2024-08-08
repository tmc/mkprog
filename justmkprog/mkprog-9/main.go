package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

//go:embed system-prompt.txt
var systemPrompt string

type Config struct {
	APIKey          string
	OutputDir       string
	Model           string
	ProjectTemplate string
	CustomTemplate  string
	DryRun          bool
	Temperature     float64
	MaxTokens       int
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	var config Config

	rootCmd := &cobra.Command{
		Use:   "mkprog [project description]",
		Short: "Generate a Go project structure based on a description",
		Long:  `mkprog is a tool that generates a complete Go project structure based on a user-provided description using AI-powered code generation.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			description := args[0]
			return generateProject(description, &config)
		},
	}

	rootCmd.Flags().StringVar(&config.APIKey, "api-key", "", "API key for the AI service")
	rootCmd.Flags().StringVar(&config.OutputDir, "output", ".", "Output directory for the generated project")
	rootCmd.Flags().StringVar(&config.Model, "model", "anthropic", "AI model to use (anthropic, openai, cohere)")
	rootCmd.Flags().StringVar(&config.ProjectTemplate, "template", "cli", "Project template (cli, web, library)")
	rootCmd.Flags().StringVar(&config.CustomTemplate, "custom-template", "", "Path to a custom template file")
	rootCmd.Flags().BoolVar(&config.DryRun, "dry-run", false, "Preview generated content without creating files")
	rootCmd.Flags().Float64Var(&config.Temperature, "temperature", 0.1, "Temperature for AI generation")
	rootCmd.Flags().IntVar(&config.MaxTokens, "max-tokens", 8192, "Maximum number of tokens for AI generation")

	viper.SetConfigName("mkprog")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.config/mkprog")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("MKPROG")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	if err := viper.BindPFlags(rootCmd.Flags()); err != nil {
		return fmt.Errorf("error binding flags: %w", err)
	}

	if err := rootCmd.Execute(); err != nil {
		return err
	}

	return nil
}

func generateProject(description string, config *Config) error {
	ctx := context.Background()

	client, err := anthropic.New(anthropic.WithAPIKey(config.APIKey), anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35))
	if err != nil {
		return fmt.Errorf("error creating Anthropic client: %w", err)
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Generating project structure..."
	s.Start()
	defer s.Stop()

	projectStructure, err := generateProjectStructure(ctx, client, description, config)
	if err != nil {
		return fmt.Errorf("error generating project structure: %w", err)
	}

	if config.DryRun {
		fmt.Println("Dry run: Generated project structure")
		for _, file := range projectStructure {
			fmt.Printf("File: %s\nContent:\n%s\n\n", file.Path, file.Content)
		}
		return nil
	}

	if err := createProjectFiles(projectStructure, config.OutputDir); err != nil {
		return fmt.Errorf("error creating project files: %w", err)
	}

	fmt.Printf("Project generated successfully in %s\n", config.OutputDir)
	return nil
}

type ProjectFile struct {
	Path    string
	Content string
}

func generateProjectStructure(ctx context.Context, client llms.Model, description string, config *Config) ([]ProjectFile, error) {
	prompt := fmt.Sprintf(`%s

Project Description: %s
Project Template: %s

Generate a complete Go project structure based on the given description and template. Include the following files:

1. main.go
2. go.mod
3. README.md
4. Any additional packages or files needed for the project

For each file, provide the file path and its content. Use the following format for each file:

===FILE_PATH===
FILE_CONTENT
===END_FILE===

Ensure that the generated code follows Go best practices and is well-documented.`, systemPrompt, description, config.ProjectTemplate)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(config.Temperature), llms.WithMaxTokens(config.MaxTokens))
	if err != nil {
		return nil, fmt.Errorf("error generating content: %w", err)
	}

	content := resp.Choices[0].Content

	var projectFiles []ProjectFile
	files := strings.Split(content, "===FILE_PATH===")
	for _, file := range files[1:] {
		parts := strings.SplitN(file, "===END_FILE===", 2)
		if len(parts) != 2 {
			continue
		}
		path := strings.TrimSpace(parts[0])
		content := strings.TrimSpace(parts[1])
		projectFiles = append(projectFiles, ProjectFile{Path: path, Content: content})
	}

	return projectFiles, nil
}

func createProjectFiles(files []ProjectFile, outputDir string) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(files))

	for _, file := range files {
		wg.Add(1)
		go func(f ProjectFile) {
			defer wg.Done()
			fullPath := filepath.Join(outputDir, f.Path)
			dir := filepath.Dir(fullPath)

			if err := os.MkdirAll(dir, 0755); err != nil {
				errChan <- fmt.Errorf("error creating directory %s: %w", dir, err)
				return
			}

			if err := os.WriteFile(fullPath, []byte(f.Content), 0644); err != nil {
				errChan <- fmt.Errorf("error writing file %s: %w", fullPath, err)
				return
			}
		}(file)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
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
		fmt.Printf("=== %s ===\n%s\n\n", file, content)
	}
}

func init() {
	if os.Getenv("_MKPROG_DUMP") != "" {
		dumpsrc()
		os.Exit(0)
	}
}
