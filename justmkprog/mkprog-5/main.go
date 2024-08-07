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
	MaxTokens       int
	Temperature     float64
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
	rootCmd.PersistentFlags().StringVar(&config.APIKey, "api-key", "", "API key for the AI service")
	rootCmd.PersistentFlags().StringVar(&config.OutputDir, "output-dir", "", "Output directory for the generated project")
	rootCmd.PersistentFlags().StringVar(&config.CustomTemplate, "custom-template", "", "Path to a custom template file")
	rootCmd.PersistentFlags().BoolVar(&config.DryRun, "dry-run", false, "Preview generated content without creating files")
	rootCmd.PersistentFlags().StringVar(&config.AIModel, "ai-model", "anthropic", "AI model to use (anthropic, openai, cohere)")
	rootCmd.PersistentFlags().StringVar(&config.ProjectTemplate, "project-template", "cli", "Project template (cli, web, library)")
	rootCmd.PersistentFlags().IntVar(&config.MaxTokens, "max-tokens", 8000, "Maximum number of tokens for AI generation")
	rootCmd.PersistentFlags().Float64Var(&config.Temperature, "temperature", 0.1, "Temperature for AI generation")

	viper.SetConfigName("mkprog")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.config/mkprog")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	viper.BindPFlags(rootCmd.PersistentFlags())
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

	ctx := context.Background()
	client, err := anthropic.New(anthropic.WithApiKey(config.APIKey), anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35))
	if err != nil {
		return fmt.Errorf("failed to create Anthropic client: %w", err)
	}

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
		fmt.Println("Dry run: Generated project structure")
		for filename, content := range projectStructure.Files {
			fmt.Printf("=== %s ===\n%s\n\n", filename, content)
		}
		return nil
	}

	err = writeProjectFiles(projectStructure)
	if err != nil {
		return fmt.Errorf("failed to write project files: %w", err)
	}

	fmt.Println("Project generated successfully in", config.OutputDir)
	return nil
}

func generateProjectStructure(ctx context.Context, client llms.Model, projectDescription string) (*ProjectStructure, error) {
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Generate a Go project structure for the following description: %s", projectDescription)),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(config.Temperature), llms.WithMaxTokens(config.MaxTokens))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	var projectStructure ProjectStructure
	err = json.Unmarshal([]byte(resp.Choices[0].Content), &projectStructure)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal project structure: %w", err)
	}

	return &projectStructure, nil
}

func writeProjectFiles(projectStructure *ProjectStructure) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(projectStructure.Files))

	for filename, content := range projectStructure.Files {
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
		return fmt.Errorf("errors occurred while writing files:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

func init() {
	if os.Getenv("_MKPROG_DUMP") != "" {
		dumpsrc()
		os.Exit(0)
	}
}

func dumpsrc() {
	fmt.Println("=== main.go ===")
	data, _ := ioutil.ReadFile("main.go")
	fmt.Println(string(data))

	fmt.Println("=== go.mod ===")
	data, _ = ioutil.ReadFile("go.mod")
	fmt.Println(string(data))

	fmt.Println("=== system-prompt.txt ===")
	data, _ = ioutil.ReadFile("system-prompt.txt")
	fmt.Println(string(data))
}
