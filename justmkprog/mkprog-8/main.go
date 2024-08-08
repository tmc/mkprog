package main

import (
	"context"
	"fmt"
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
	TemplateFile    string
	DryRun          bool
	AIModel         string
	ProjectTemplate string
	MaxTokens       int
	Temperature     float64
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
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
			return generateProject(args[0], &config)
		},
	}

	rootCmd.Flags().StringVar(&config.APIKey, "api-key", "", "API key for the AI service")
	rootCmd.Flags().StringVarP(&config.OutputDir, "output", "o", "", "Output directory for the generated project")
	rootCmd.Flags().StringVar(&config.TemplateFile, "template", "", "Custom template file")
	rootCmd.Flags().BoolVar(&config.DryRun, "dry-run", false, "Preview generated content without creating files")
	rootCmd.Flags().StringVar(&config.AIModel, "ai-model", "anthropic", "AI model to use (anthropic, openai, cohere)")
	rootCmd.Flags().StringVar(&config.ProjectTemplate, "project-type", "cli", "Project template (cli, web, library)")
	rootCmd.Flags().IntVar(&config.MaxTokens, "max-tokens", 8192, "Maximum number of tokens for AI generation")
	rootCmd.Flags().Float64Var(&config.Temperature, "temperature", 0.1, "Temperature for AI generation")

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
	if config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	if config.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	client, err := anthropic.New(anthropic.WithAPIKey(config.APIKey))
	if err != nil {
		return fmt.Errorf("error creating Anthropic client: %w", err)
	}

	ctx := context.Background()

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Generating project structure..."
	s.Start()
	defer s.Stop()

	projectStructure, err := generateProjectStructure(ctx, client, description, config)
	if err != nil {
		return fmt.Errorf("error generating project structure: %w", err)
	}

	s.Suffix = " Generating project files..."
	files, err := generateProjectFiles(ctx, client, projectStructure, config)
	if err != nil {
		return fmt.Errorf("error generating project files: %w", err)
	}

	if config.DryRun {
		return previewGeneratedContent(files)
	}

	s.Suffix = " Writing project files..."
	if err := writeProjectFiles(files, config.OutputDir); err != nil {
		return fmt.Errorf("error writing project files: %w", err)
	}

	fmt.Printf("Project generated successfully in %s\n", config.OutputDir)
	return nil
}

func generateProjectStructure(ctx context.Context, client llms.Model, description string, config *Config) (string, error) {
	prompt := fmt.Sprintf(`%s

Project Description: %s
Project Type: %s

Generate a project structure for the described Go project. Include main package, additional packages, test files, README.md, and go.mod. Return the structure as a tree-like text representation.`, systemPrompt, description, config.ProjectTemplate)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(config.Temperature), llms.WithMaxTokens(config.MaxTokens))
	if err != nil {
		return "", fmt.Errorf("error generating project structure: %w", err)
	}

	return resp.Choices[0].Content, nil
}

func generateProjectFiles(ctx context.Context, client llms.Model, projectStructure string, config *Config) (map[string]string, error) {
	files := make(map[string]string)
	var wg sync.WaitGroup
	var mu sync.Mutex

	fileList := strings.Split(projectStructure, "\n")
	for _, file := range fileList {
		if strings.TrimSpace(file) == "" {
			continue
		}
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			content, err := generateFileContent(ctx, client, file, config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error generating content for %s: %v\n", file, err)
				return
			}
			mu.Lock()
			files[file] = content
			mu.Unlock()
		}(file)
	}

	wg.Wait()
	return files, nil
}

func generateFileContent(ctx context.Context, client llms.Model, file string, config *Config) (string, error) {
	prompt := fmt.Sprintf(`%s

Generate the content for the file: %s
Project Type: %s

Provide only the file content without any additional formatting or explanations.`, systemPrompt, file, config.ProjectTemplate)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(config.Temperature), llms.WithMaxTokens(config.MaxTokens))
	if err != nil {
		return "", fmt.Errorf("error generating file content: %w", err)
	}

	return resp.Choices[0].Content, nil
}

func writeProjectFiles(files map[string]string, outputDir string) error {
	for file, content := range files {
		fullPath := filepath.Join(outputDir, file)
		dir := filepath.Dir(fullPath)

		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("error creating directory %s: %w", dir, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("error writing file %s: %w", fullPath, err)
		}
	}

	return nil
}

func previewGeneratedContent(files map[string]string) error {
	for file, content := range files {
		fmt.Printf("=== %s ===\n", file)
		fmt.Println(content)
		fmt.Println()
	}
	return nil
}

func dumpsrc() {
	files := []string{"main.go", "go.mod", "system-prompt.txt"}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", file, err)
			continue
		}
		fmt.Printf("=== %s ===\n", file)
		fmt.Println(string(content))
		fmt.Println()
	}
}

func init() {
	if os.Getenv("_MKPROG_DUMP") != "" {
		dumpsrc()
		os.Exit(0)
	}
}
