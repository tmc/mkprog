package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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

type FileContent struct {
	Name    string
	Content string
}

var (
	config Config
	cache  = make(map[string]string)
	mutex  = &sync.Mutex{}
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	rootCmd := &cobra.Command{
		Use:   "mkprog [project description]",
		Short: "Generate a Go project structure based on a description",
		Long:  `mkprog is a tool that generates a complete Go project structure based on a user-provided description using AI-powered code generation.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runGenerator,
	}

	rootCmd.Flags().StringVarP(&config.OutputDir, "output", "o", "", "Output directory for the generated project")
	rootCmd.Flags().StringVarP(&config.APIKey, "api-key", "k", "", "API key for the AI service")
	rootCmd.Flags().StringVarP(&config.CustomTemplate, "template", "t", "", "Custom template file")
	rootCmd.Flags().BoolVarP(&config.DryRun, "dry-run", "d", false, "Preview generated content without creating files")
	rootCmd.Flags().StringVarP(&config.AIModel, "ai-model", "m", "anthropic", "AI model to use (anthropic, openai, cohere)")
	rootCmd.Flags().StringVarP(&config.ProjectTemplate, "project-type", "p", "cli", "Project template (cli, web, library)")

	viper.SetConfigName("mkprog")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.config/mkprog")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("MKPROG")

	if err := viper.BindPFlags(rootCmd.Flags()); err != nil {
		return err
	}

	return rootCmd.Execute()
}

func runGenerator(cmd *cobra.Command, args []string) error {
	projectDescription := args[0]

	if config.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	if config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Generating project structure..."
	s.Start()

	files, err := generateProjectStructure(projectDescription)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to generate project structure: %w", err)
	}

	s.Stop()

	if config.DryRun {
		fmt.Println("Dry run: Generated files:")
		for _, file := range files {
			fmt.Printf("=== %s ===\n%s\n\n", file.Name, file.Content)
		}
		return nil
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(files))

	for _, file := range files {
		wg.Add(1)
		go func(f FileContent) {
			defer wg.Done()
			if err := writeFile(f.Name, f.Content); err != nil {
				errChan <- fmt.Errorf("failed to write file %s: %w", f.Name, err)
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

	fmt.Println("Project generated successfully!")
	return nil
}

func generateProjectStructure(description string) ([]FileContent, error) {
	ctx := context.Background()
	client, err := anthropic.New(anthropic.WithAPIKey(config.APIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	prompt := fmt.Sprintf(`%s

Project Description: %s
Project Template: %s

Generate a complete Go project structure based on the given description and template. Provide the content for each file, including:

1. main.go
2. Additional package files
3. Test files
4. README.md
5. go.mod

Output the result as a JSON array of objects, where each object has "Name" and "Content" fields representing the file name and its content, respectively.`, systemPrompt, description, config.ProjectTemplate)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(8000), anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	var files []FileContent
	if err := json.Unmarshal([]byte(resp.Content), &files); err != nil {
		return nil, fmt.Errorf("failed to parse generated content: %w", err)
	}

	return files, nil
}

func writeFile(name, content string) error {
	mutex.Lock()
	defer mutex.Unlock()

	filePath := filepath.Join(config.OutputDir, name)
	dir := filepath.Dir(filePath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	return ioutil.WriteFile(filePath, []byte(content), 0644)
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
		content, err := ioutil.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", file, err)
			continue
		}
		fmt.Printf("=== %s ===\n%s\n\n", file, content)
	}
}
