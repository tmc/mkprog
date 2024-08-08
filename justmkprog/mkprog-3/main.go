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
	OutputDir    string
	APIKey       string
	TemplateFile string
	DryRun       bool
	AIModel      string
	ProjectType  string
	Temperature  float64
	MaxTokens    int
	Verbose      bool
}

var config Config

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	rootCmd := &cobra.Command{
		Use:   "mkprog [project description]",
		Short: "Generate a Go project structure based on a description",
		Long:  `mkprog is a tool that generates a complete Go project structure based on a user-provided description using AI.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runGenerate,
	}

	rootCmd.Flags().StringVarP(&config.OutputDir, "output", "o", "", "Output directory for the generated project")
	rootCmd.Flags().StringVarP(&config.APIKey, "api-key", "k", "", "API key for the AI service")
	rootCmd.Flags().StringVarP(&config.TemplateFile, "template", "t", "", "Custom template file")
	rootCmd.Flags().BoolVarP(&config.DryRun, "dry-run", "d", false, "Perform a dry run without creating files")
	rootCmd.Flags().StringVarP(&config.AIModel, "ai-model", "m", "anthropic", "AI model to use (anthropic, openai, cohere)")
	rootCmd.Flags().StringVarP(&config.ProjectType, "project-type", "p", "cli", "Project template (cli, web, library)")
	rootCmd.Flags().Float64VarP(&config.Temperature, "temperature", "", 0.1, "AI model temperature")
	rootCmd.Flags().IntVarP(&config.MaxTokens, "max-tokens", "", 8192, "Maximum number of tokens for AI response")
	rootCmd.Flags().BoolVarP(&config.Verbose, "verbose", "v", false, "Enable verbose logging")

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

func runGenerate(cmd *cobra.Command, args []string) error {
	projectDescription := args[0]

	if config.Verbose {
		log.Println("Verbose logging enabled")
		log.Printf("Project description: %s", projectDescription)
		log.Printf("Output directory: %s", config.OutputDir)
		log.Printf("AI Model: %s", config.AIModel)
		log.Printf("Project Type: %s", config.ProjectType)
	}

	if config.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	if config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	client, err := anthropic.New(anthropic.WithAPIKey(config.APIKey), anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35))
	if err != nil {
		return fmt.Errorf("error creating Anthropic client: %w", err)
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Generating project structure..."
	s.Start()
	defer s.Stop()

	ctx := context.Background()
	projectStructure, err := generateProjectStructure(ctx, client, projectDescription)
	if err != nil {
		return fmt.Errorf("error generating project structure: %w", err)
	}

	if config.DryRun {
		fmt.Println("Dry run: Project structure")
		fmt.Println(projectStructure)
		return nil
	}

	if err := createProjectFiles(projectStructure); err != nil {
		return fmt.Errorf("error creating project files: %w", err)
	}

	fmt.Printf("Project generated successfully in %s\n", config.OutputDir)
	return nil
}

func generateProjectStructure(ctx context.Context, client llms.ChatLLM, description string) (string, error) {
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Generate a Go project structure for the following description: %s", description)),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(config.Temperature), llms.WithMaxTokens(config.MaxTokens))
	if err != nil {
		return "", fmt.Errorf("error generating content: %w", err)
	}

	return resp.Choices[0].Content, nil
}

func createProjectFiles(projectStructure string) error {
	files := parseProjectStructure(projectStructure)

	var wg sync.WaitGroup
	errChan := make(chan error, len(files))

	for filename, content := range files {
		wg.Add(1)
		go func(filename, content string) {
			defer wg.Done()
			if err := writeFile(filename, content); err != nil {
				errChan <- fmt.Errorf("error writing file %s: %w", filename, err)
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

func parseProjectStructure(structure string) map[string]string {
	files := make(map[string]string)
	lines := strings.Split(structure, "\n")
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
			currentContent.WriteString(line)
			currentContent.WriteString("\n")
		}
	}

	if currentFile != "" {
		files[currentFile] = currentContent.String()
	}

	return files
}

func writeFile(filename, content string) error {
	fullPath := filepath.Join(config.OutputDir, filename)
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

func init() {
	if os.Getenv("_MKPROG_DUMP") != "" {
		dumpsrc()
		os.Exit(0)
	}
}

func dumpsrc() {
	fmt.Println("=== main.go ===")
	data, _ := os.ReadFile("main.go")
	fmt.Println(string(data))

	fmt.Println("=== go.mod ===")
	data, _ = os.ReadFile("go.mod")
	fmt.Println(string(data))

	fmt.Println("=== system-prompt.txt ===")
	data, _ = os.ReadFile("system-prompt.txt")
	fmt.Println(string(data))
}
