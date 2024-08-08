package main

import (
	"context"
	"fmt"
	"io"
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
	Description     string
	CustomTemplate  string
	DryRun          bool
	AIModel         string
	ProjectTemplate string
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
		Use:   "mkprog [flags] <project description>",
		Short: "Generate a Go project structure based on a description",
		Long:  `mkprog is a tool that generates a complete Go project structure based on a user-provided description using AI-powered code generation.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			config.Description = args[0]
			return generateProject(config)
		},
	}

	rootCmd.Flags().StringVarP(&config.OutputDir, "output", "o", "", "Output directory for the generated project")
	rootCmd.Flags().StringVarP(&config.APIKey, "api-key", "k", "", "API key for the AI service")
	rootCmd.Flags().StringVarP(&config.CustomTemplate, "template", "t", "", "Custom template file")
	rootCmd.Flags().BoolVarP(&config.DryRun, "dry-run", "d", false, "Perform a dry run without creating files")
	rootCmd.Flags().StringVarP(&config.AIModel, "ai-model", "m", "anthropic", "AI model to use (anthropic, openai, cohere)")
	rootCmd.Flags().StringVarP(&config.ProjectTemplate, "project-type", "p", "cli", "Project template (cli, web, library)")
	rootCmd.Flags().Float64VarP(&config.Temperature, "temperature", "", 0.1, "Temperature for AI generation")
	rootCmd.Flags().IntVarP(&config.MaxTokens, "max-tokens", "", 8192, "Maximum tokens for AI generation")

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

func generateProject(config Config) error {
	if config.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	if config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	ctx := context.Background()
	client, err := anthropic.New(anthropic.WithApiKey(config.APIKey))
	if err != nil {
		return fmt.Errorf("error creating Anthropic client: %w", err)
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Generating project structure..."
	s.Start()
	defer s.Stop()

	projectStructure := []string{
		"main.go",
		"go.mod",
		"README.md",
		"cmd/root.go",
		"internal/generator/generator.go",
		"pkg/template/template.go",
		"pkg/ai/ai.go",
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(projectStructure))

	for _, file := range projectStructure {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			content, err := generateFileContent(ctx, client, config, file)
			if err != nil {
				errChan <- fmt.Errorf("error generating content for %s: %w", file, err)
				return
			}

			if !config.DryRun {
				if err := writeFile(config.OutputDir, file, content); err != nil {
					errChan <- fmt.Errorf("error writing file %s: %w", file, err)
					return
				}
			} else {
				fmt.Printf("=== %s ===\n%s\n\n", file, content)
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

	if !config.DryRun {
		fmt.Printf("Project generated successfully in %s\n", config.OutputDir)
	}

	return nil
}

func generateFileContent(ctx context.Context, client llms.Model, config Config, file string) (string, error) {
	prompt := fmt.Sprintf("Generate the content for the file %s in a Go project with the following description:\n\n%s\n\nThe project type is %s. Please provide only the file content, without any additional explanations or markdown formatting.", file, config.Description, config.ProjectTemplate)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages,
		llms.WithTemperature(config.Temperature),
		llms.WithMaxTokens(config.MaxTokens),
		anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35),
	)
	if err != nil {
		return "", fmt.Errorf("error generating content: %w", err)
	}

	return resp.Choices[0].Content, nil
}

func writeFile(outputDir, file, content string) error {
	fullPath := filepath.Join(outputDir, file)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating directory %s: %w", dir, err)
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("error creating file %s: %w", fullPath, err)
	}
	defer f.Close()

	_, err = io.WriteString(f, content)
	if err != nil {
		return fmt.Errorf("error writing to file %s: %w", fullPath, err)
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
