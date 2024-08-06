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
	Path    string
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
		Args:  cobra.ExactArgs(1),
		RunE:  runGenerator,
	}

	rootCmd.Flags().StringVarP(&config.OutputDir, "output", "o", "", "Output directory for the generated project")
	rootCmd.Flags().StringVarP(&config.APIKey, "api-key", "k", "", "API key for the AI model")
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
		return fmt.Errorf("error binding flags: %w", err)
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
		return fmt.Errorf("error creating output directory: %w", err)
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Generating project structure..."
	s.Start()

	ctx := context.Background()
	client, err := anthropic.New(anthropic.WithAPIKey(config.APIKey))
	if err != nil {
		s.Stop()
		return fmt.Errorf("error creating Anthropic client: %w", err)
	}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Generate a Go project structure for the following description: %s", projectDescription)),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(8000), anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35))
	if err != nil {
		s.Stop()
		return fmt.Errorf("error generating content: %w", err)
	}

	s.Stop()

	var files []FileContent
	if err := json.Unmarshal([]byte(resp.Choices[0].Content), &files); err != nil {
		return fmt.Errorf("error parsing generated content: %w", err)
	}

	if config.DryRun {
		fmt.Println("Dry run: Generated files:")
		for _, file := range files {
			fmt.Printf("=== %s ===\n%s\n\n", file.Path, file.Content)
		}
		return nil
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(files))

	for _, file := range files {
		wg.Add(1)
		go func(f FileContent) {
			defer wg.Done()
			if err := writeFile(f.Path, f.Content); err != nil {
				errChan <- fmt.Errorf("error writing file %s: %w", f.Path, err)
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

	fmt.Println("Project structure generated successfully!")
	return nil
}

func writeFile(path string, content string) error {
	fullPath := filepath.Join(config.OutputDir, path)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating directory %s: %w", dir, err)
	}

	mutex.Lock()
	defer mutex.Unlock()

	cacheKey := fmt.Sprintf("%s:%s", path, content)
	if cachedContent, ok := cache[cacheKey]; ok {
		content = cachedContent
	} else {
		cache[cacheKey] = content
	}

	return ioutil.WriteFile(fullPath, []byte(content), 0644)
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
