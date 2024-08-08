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
		Long:  `mkprog is a tool that generates a complete Go project structure based on a user-provided description using AI.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			description := args[0]
			return generateProject(description, config)
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

func generateProject(description string, config Config) error {
	ctx := context.Background()

	client, err := anthropic.New(anthropic.WithAPIKey(config.APIKey), anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35))
	if err != nil {
		return fmt.Errorf("error creating Anthropic client: %w", err)
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Generating project structure..."
	s.Start()
	defer s.Stop()

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Generate a Go project structure for the following description: %s", description)),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(config.Temperature), llms.WithMaxTokens(config.MaxTokens))
	if err != nil {
		return fmt.Errorf("error generating content: %w", err)
	}

	s.Stop()

	if config.DryRun {
		fmt.Println("Dry run: Generated content preview")
		fmt.Println(resp.Choices[0].Content)
		return nil
	}

	files, err := parseGeneratedContent(resp.Choices[0].Content)
	if err != nil {
		return fmt.Errorf("error parsing generated content: %w", err)
	}

	if err := writeFiles(files, config.OutputDir); err != nil {
		return fmt.Errorf("error writing files: %w", err)
	}

	fmt.Println("Project generated successfully!")
	return nil
}

func parseGeneratedContent(content string) (map[string]string, error) {
	files := make(map[string]string)
	lines := strings.Split(content, "\n")
	var currentFile string
	var currentContent strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "===") && strings.HasSuffix(line, "===") {
			if currentFile != "" {
				files[currentFile] = currentContent.String()
				currentContent.Reset()
			}
			currentFile = strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(line, "==="), "==="))
		} else {
			currentContent.WriteString(line + "\n")
		}
	}

	if currentFile != "" {
		files[currentFile] = currentContent.String()
	}

	return files, nil
}

func writeFiles(files map[string]string, outputDir string) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(files))

	for filename, content := range files {
		wg.Add(1)
		go func(filename, content string) {
			defer wg.Done()
			fullPath := filepath.Join(outputDir, filename)
			dir := filepath.Dir(fullPath)

			if err := os.MkdirAll(dir, 0755); err != nil {
				errChan <- fmt.Errorf("error creating directory %s: %w", dir, err)
				return
			}

			if err := writeFile(fullPath, content); err != nil {
				errChan <- fmt.Errorf("error writing file %s: %w", fullPath, err)
				return
			}
		}(filename, content)
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

func writeFile(filename, content string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.WriteString(f, content)
	return err
}

func init() {
	if os.Getenv("_MKPROG_DUMP") != "" {
		dumpsrc()
		os.Exit(0)
	}
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
	}
}
