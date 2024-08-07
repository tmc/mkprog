package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	CustomTemplate  string
	DryRun          bool
	AIModel         string
	ProjectTemplate string
	MaxTokens       int
	Temperature     float64
}

type ProjectStructure struct {
	Files map[string]string
}

var (
	cfg     Config
	rootCmd = &cobra.Command{
		Use:   "mkprog [project description]",
		Short: "Generate a Go project structure based on a description",
		Long:  `mkprog is a CLI tool that generates a complete Go project structure based on a user-provided description using AI-powered code generation.`,
		Args:  cobra.ExactArgs(1),
		RunE:  run,
	}
)

func init() {
	rootCmd.Flags().StringVar(&cfg.APIKey, "api-key", "", "API key for the AI service")
	rootCmd.Flags().StringVar(&cfg.OutputDir, "output-dir", "", "Output directory for the generated project")
	rootCmd.Flags().StringVar(&cfg.CustomTemplate, "custom-template", "", "Path to a custom template file")
	rootCmd.Flags().BoolVar(&cfg.DryRun, "dry-run", false, "Preview generated content without creating files")
	rootCmd.Flags().StringVar(&cfg.AIModel, "ai-model", "anthropic", "AI model to use (anthropic, openai, cohere)")
	rootCmd.Flags().StringVar(&cfg.ProjectTemplate, "project-template", "cli", "Project template (cli, web, library)")
	rootCmd.Flags().IntVar(&cfg.MaxTokens, "max-tokens", 8192, "Maximum number of tokens for AI generation")
	rootCmd.Flags().Float64Var(&cfg.Temperature, "temperature", 0.1, "Temperature for AI generation")

	viper.SetConfigName("mkprog")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.config/mkprog")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("MKPROG")
	viper.BindPFlags(rootCmd.Flags())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	projectDescription := args[0]

	if cfg.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	if cfg.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	client, err := anthropic.New(anthropic.WithAPIKey(cfg.APIKey), anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35))
	if err != nil {
		return fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Generating project structure..."
	s.Start()

	projectStructure, err := generateProjectStructure(client, projectDescription)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to generate project structure: %w", err)
	}

	s.Stop()

	if cfg.DryRun {
		fmt.Println("Dry run: Generated project structure")
		for filename, content := range projectStructure.Files {
			fmt.Printf("=== %s ===\n%s\n\n", filename, content)
		}
		return nil
	}

	if err := createProjectFiles(projectStructure); err != nil {
		return fmt.Errorf("failed to create project files: %w", err)
	}

	fmt.Println("Project generated successfully!")
	return nil
}

func generateProjectStructure(client llms.Model, description string) (*ProjectStructure, error) {
	ctx := context.Background()

	prompt := fmt.Sprintf(`%s

Project Description: %s

Generate a complete Go project structure based on the above description. Provide the content for each file in the project, including main.go, additional packages, test files, README.md, and go.mod. The output should be a JSON object with filenames as keys and file contents as values.

Use the following project template: %s

Ensure that the generated code follows Go best practices and includes proper error handling, logging, and documentation.`, systemPrompt, description, cfg.ProjectTemplate)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(cfg.Temperature), llms.WithMaxTokens(cfg.MaxTokens))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	var projectStructure ProjectStructure
	if err := json.Unmarshal([]byte(resp.Choices[0].Content), &projectStructure); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project structure: %w", err)
	}

	return &projectStructure, nil
}

func createProjectFiles(structure *ProjectStructure) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(structure.Files))

	for filename, content := range structure.Files {
		wg.Add(1)
		go func(filename, content string) {
			defer wg.Done()
			fullPath := filepath.Join(cfg.OutputDir, filename)
			dir := filepath.Dir(fullPath)

			if err := os.MkdirAll(dir, 0755); err != nil {
				errChan <- fmt.Errorf("failed to create directory %s: %w", dir, err)
				return
			}

			if err := ioutil.WriteFile(fullPath, []byte(content), 0644); err != nil {
				errChan <- fmt.Errorf("failed to write file %s: %w", fullPath, err)
				return
			}
		}(filename, content)
	}

	wg.Wait()
	close(errChan)

	var errors []string
	for err := range errChan {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors occurred while creating project files:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

func dumpsrc() {
	files := []string{"main.go", "go.mod", "system-prompt.txt"}
	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			log.Printf("Error reading %s: %v", file, err)
			continue
		}
		fmt.Printf("=== %s ===\n%s\n\n", file, string(content))
	}
}

func init() {
	if os.Getenv("_MKPROG_DUMP") != "" {
		dumpsrc()
		os.Exit(0)
	}
}
