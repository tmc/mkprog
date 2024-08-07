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
	CustomTemplate  string
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateProject(args[0], &config)
		},
	}

	rootCmd.Flags().StringVar(&config.APIKey, "api-key", "", "API key for the AI service")
	rootCmd.Flags().StringVar(&config.OutputDir, "output", ".", "Output directory for the generated project")
	rootCmd.Flags().StringVar(&config.CustomTemplate, "template", "", "Custom template file")
	rootCmd.Flags().BoolVar(&config.DryRun, "dry-run", false, "Preview generated content without creating files")
	rootCmd.Flags().StringVar(&config.AIModel, "ai-model", "anthropic", "AI model to use (anthropic, openai, cohere)")
	rootCmd.Flags().StringVar(&config.ProjectTemplate, "project-type", "cli", "Project template (cli, web, library)")
	rootCmd.Flags().IntVar(&config.MaxTokens, "max-tokens", 8192, "Maximum number of tokens for AI response")
	rootCmd.Flags().Float64Var(&config.Temperature, "temperature", 0.1, "Temperature for AI response")

	viper.SetConfigName("mkprog")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.config/mkprog")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("MKPROG")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	if err := viper.BindPFlags(rootCmd.Flags()); err != nil {
		return err
	}

	if err := rootCmd.Execute(); err != nil {
		return err
	}

	return nil
}

func generateProject(description string, config *Config) error {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Generating project..."
	s.Start()
	defer s.Stop()

	ctx := context.Background()
	client, err := anthropic.New(anthropic.WithAPIKey(config.APIKey))
	if err != nil {
		return fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	projectStructure, err := generateProjectStructure(ctx, client, description, config)
	if err != nil {
		return fmt.Errorf("failed to generate project structure: %w", err)
	}

	if config.DryRun {
		fmt.Println("Dry run: Project structure")
		fmt.Printf("%+v\n", projectStructure)
		return nil
	}

	if err := createProjectFiles(projectStructure, config.OutputDir); err != nil {
		return fmt.Errorf("failed to create project files: %w", err)
	}

	fmt.Printf("Project generated successfully in %s\n", config.OutputDir)
	return nil
}

func generateProjectStructure(ctx context.Context, client llms.Model, description string, config *Config) (map[string]string, error) {
	prompt := fmt.Sprintf(`%s

Project Description: %s
Project Type: %s

Generate a complete Go project structure based on the above description. Provide the content for each file, including code, documentation, and configuration. The response should be a JSON object where keys are file paths and values are file contents.

Example format:
{
  "main.go": "package main\n\nfunc main() {\n\t// Main function code\n}",
  "pkg/utils/helper.go": "package utils\n\nfunc HelperFunction() {\n\t// Helper function code\n}",
  "README.md": "# Project Name\n\nProject description and usage instructions."
}

Ensure that the generated project follows Go best practices, includes proper error handling, and is well-documented.`, systemPrompt, description, config.ProjectTemplate)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(config.Temperature), llms.WithMaxTokens(config.MaxTokens))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	// Parse the JSON response and return the project structure
	// This is a simplified version; you'd need to implement proper JSON parsing
	projectStructure := make(map[string]string)
	// TODO: Implement JSON parsing of the response
	_ = resp // Placeholder to use resp variable

	return projectStructure, nil
}

func createProjectFiles(projectStructure map[string]string, outputDir string) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(projectStructure))

	for filePath, content := range projectStructure {
		wg.Add(1)
		go func(fp, c string) {
			defer wg.Done()
			fullPath := filepath.Join(outputDir, fp)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				errChan <- fmt.Errorf("failed to create directory for %s: %w", fp, err)
				return
			}
			if err := os.WriteFile(fullPath, []byte(c), 0644); err != nil {
				errChan <- fmt.Errorf("failed to write file %s: %w", fp, err)
				return
			}
		}(filePath, content)
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

func init() {
	// Initialize any global settings or plugins here
}

// TODO: Implement the following features:
// - Custom template support
// - Caching system for generated content
// - Unit tests
// - Project updating functionality
// - Plugin system
// - Interactive mode
// - Version control integration
// - Docker support
// - Code linting and formatting
// - Project documentation generation
