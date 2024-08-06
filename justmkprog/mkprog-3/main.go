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
}

type ProjectStructure struct {
	Files map[string]string
}

var (
	config Config
	cache  = make(map[string]string)
	mutex  = &sync.Mutex{}
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	rootCmd := &cobra.Command{
		Use:   "mkprog [project description]",
		Short: "Generate a Go project structure based on a description",
		Args:  cobra.ExactArgs(1),
		RunE:  generateProject,
	}

	rootCmd.Flags().StringVarP(&config.OutputDir, "output", "o", "", "Output directory for the generated project")
	rootCmd.Flags().StringVarP(&config.APIKey, "api-key", "k", "", "API key for the AI model")
	rootCmd.Flags().StringVarP(&config.CustomTemplate, "template", "t", "", "Custom template file")
	rootCmd.Flags().BoolVarP(&config.DryRun, "dry-run", "d", false, "Preview generated content without creating files")
	rootCmd.Flags().StringVarP(&config.AIModel, "ai-model", "m", "anthropic", "AI model to use (anthropic, openai, cohere)")
	rootCmd.Flags().StringVarP(&config.ProjectTemplate, "project-type", "p", "cli", "Project template (cli, web, library)")

	viper.SetConfigName("mkprog")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.config/mkprog")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	viper.BindPFlags(rootCmd.Flags())

	return rootCmd.Execute()
}

func generateProject(cmd *cobra.Command, args []string) error {
	description := args[0]

	if config.APIKey == "" {
		config.APIKey = os.Getenv("ANTHROPIC_API_KEY")
		if config.APIKey == "" {
			return fmt.Errorf("API key not provided and ANTHROPIC_API_KEY environment variable not set")
		}
	}

	if config.OutputDir == "" {
		return fmt.Errorf("output directory not specified")
	}

	client, err := anthropic.New(anthropic.WithAPIKey(config.APIKey))
	if err != nil {
		return fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Generating project structure..."
	s.Start()

	ctx := context.Background()
	projectStructure, err := generateProjectStructure(ctx, client, description)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to generate project structure: %w", err)
	}

	s.Stop()

	if config.DryRun {
		fmt.Println("Dry run: Generated project structure")
		for filename, content := range projectStructure.Files {
			fmt.Printf("=== %s ===\n%s\n\n", filename, content)
		}
		return nil
	}

	if err := createProjectFiles(projectStructure); err != nil {
		return fmt.Errorf("failed to create project files: %w", err)
	}

	fmt.Printf("Project generated successfully in %s\n", config.OutputDir)
	return nil
}

func generateProjectStructure(ctx context.Context, client llms.Model, description string) (*ProjectStructure, error) {
	prompt := fmt.Sprintf("%s\n\nProject Description: %s\nProject Type: %s\n\nGenerate a complete project structure with file contents.", systemPrompt, description, config.ProjectTemplate)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(8000))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	content := resp.Choices[0].Content

	var projectStructure ProjectStructure
	err = json.Unmarshal([]byte(content), &projectStructure)
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

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
