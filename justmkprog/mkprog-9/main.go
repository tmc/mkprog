package main

import (
	"context"
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

var config Config

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	rootCmd := &cobra.Command{
		Use:   "mkprog [project description]",
		Short: "Generate a complete Go project structure based on a description",
		Long:  `mkprog is a tool that generates a complete Go project structure based on a user-provided description using AI-powered code generation.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runGenerator,
	}

	rootCmd.Flags().StringVar(&config.APIKey, "api-key", "", "API key for the AI service")
	rootCmd.Flags().StringVar(&config.OutputDir, "output", ".", "Output directory for the generated project")
	rootCmd.Flags().StringVar(&config.CustomTemplate, "template", "", "Custom template file")
	rootCmd.Flags().BoolVar(&config.DryRun, "dry-run", false, "Preview generated content without creating files")
	rootCmd.Flags().StringVar(&config.AIModel, "ai-model", "anthropic", "AI model to use (anthropic, openai, cohere)")
	rootCmd.Flags().StringVar(&config.ProjectTemplate, "project-type", "cli", "Project template (cli, web, library)")

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

	if config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	ctx := context.Background()
	client, err := anthropic.New(anthropic.WithAPIKey(config.APIKey))
	if err != nil {
		return fmt.Errorf("error creating Anthropic client: %w", err)
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Generating project structure..."
	s.Start()

	projectStructure, err := generateProjectStructure(ctx, client, projectDescription)
	if err != nil {
		s.Stop()
		return fmt.Errorf("error generating project structure: %w", err)
	}

	s.Stop()

	if config.DryRun {
		fmt.Println("Dry run: Generated project structure")
		fmt.Println(projectStructure)
		return nil
	}

	if err := createProjectFiles(projectStructure); err != nil {
		return fmt.Errorf("error creating project files: %w", err)
	}

	fmt.Println("Project generated successfully!")
	return nil
}

func generateProjectStructure(ctx context.Context, client llms.Model, description string) (string, error) {
	prompt := fmt.Sprintf(`%s

Project Description: %s

Generate a complete Go project structure based on the above description. Include the following:

1. Main package with main.go
2. Additional packages as needed
3. Test files
4. README.md
5. go.mod file

Provide the content for each file, including code, documentation, and comments. Use the following format for each file:

===filename===
(file content)

Ensure that the generated project follows Go best practices and includes proper error handling, logging, and documentation.`, systemPrompt, description)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(8000))
	if err != nil {
		return "", fmt.Errorf("error generating content: %w", err)
	}

	return resp.Choices[0].Content, nil
}

func createProjectFiles(projectStructure string) error {
	files := strings.Split(projectStructure, "===")
	var wg sync.WaitGroup

	for _, file := range files {
		if strings.TrimSpace(file) == "" {
			continue
		}

		parts := strings.SplitN(file, "===", 2)
		if len(parts) != 2 {
			continue
		}

		filename := strings.TrimSpace(parts[0])
		content := strings.TrimSpace(parts[1])

		wg.Add(1)
		go func(filename, content string) {
			defer wg.Done()
			if err := writeFile(filename, content); err != nil {
				log.Printf("Error writing file %s: %v", filename, err)
			}
		}(filename, content)
	}

	wg.Wait()
	return nil
}

func writeFile(filename, content string) error {
	fullPath := filepath.Join(config.OutputDir, filename)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating directory %s: %w", dir, err)
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
