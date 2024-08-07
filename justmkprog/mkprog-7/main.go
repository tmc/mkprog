package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
}

type ProjectStructure struct {
	Files map[string]string
}

var config Config
var rootCmd = &cobra.Command{
	Use:   "mkprog [project description]",
	Short: "Generate a Go project structure based on a description",
	Long:  `mkprog is a CLI tool that generates a complete Go project structure based on a user-provided description using AI-powered code generation.`,
	Args:  cobra.ExactArgs(1),
	RunE:  run,
}

func init() {
	rootCmd.Flags().StringVarP(&config.OutputDir, "output", "o", "", "Output directory for the generated project")
	rootCmd.Flags().StringVarP(&config.APIKey, "api-key", "k", "", "API key for the AI service")
	rootCmd.Flags().StringVarP(&config.CustomTemplate, "template", "t", "", "Custom template file")
	rootCmd.Flags().BoolVarP(&config.DryRun, "dry-run", "d", false, "Perform a dry run without creating files")
	rootCmd.Flags().StringVarP(&config.AIModel, "ai-model", "m", "anthropic", "AI model to use (anthropic, openai, cohere)")
	rootCmd.Flags().StringVarP(&config.ProjectTemplate, "project-type", "p", "cli", "Project template (cli, web, library)")

	viper.SetConfigName("mkprog")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.config/mkprog")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

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

	if config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	if config.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	client, err := anthropic.New(anthropic.WithApiKey(config.APIKey))
	if err != nil {
		return fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	ctx := context.Background()

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Generating project structure..."
	s.Start()

	projectStructure, err := generateProjectStructure(ctx, client, projectDescription)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to generate project structure: %w", err)
	}

	s.Stop()

	if config.DryRun {
		fmt.Println("Dry run: Project structure")
		for filename, content := range projectStructure.Files {
			fmt.Printf("=== %s ===\n%s\n\n", filename, content)
		}
		return nil
	}

	err = createProjectFiles(projectStructure)
	if err != nil {
		return fmt.Errorf("failed to create project files: %w", err)
	}

	fmt.Println("Project generated successfully in:", config.OutputDir)
	return nil
}

func generateProjectStructure(ctx context.Context, client *anthropic.Chat, description string) (*ProjectStructure, error) {
	prompt := fmt.Sprintf(`%s

Project Description: %s

Generate a complete Go project structure based on the above description. Provide the content for each file, including code, documentation, and configuration. The output should be in JSON format with the following structure:

{
  "files": {
    "filename1": "content1",
    "filename2": "content2",
    ...
  }
}

Ensure that the generated project follows Go best practices, includes proper error handling, and implements the requested features.`, systemPrompt, description)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(8000))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	var projectStructure ProjectStructure
	err = json.Unmarshal([]byte(resp.Choices[0].Content), &projectStructure)
	if err != nil {
		return nil, fmt.Errorf("failed to parse generated content: %w", err)
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
			fullPath := filepath.Join(config.OutputDir, filename)
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
			fmt.Printf("Error reading %s: %v\n", file, err)
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
