package main

import (
	"context"
	"fmt"
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
	TemplateFile    string
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
	rootCmd.Flags().StringVarP(&config.TemplateFile, "template", "t", "", "Custom template file")
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
	client, err := anthropic.New(anthropic.WithAPIKey(config.APIKey))
	if err != nil {
		return fmt.Errorf("error creating Anthropic client: %w", err)
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Generating project structure..."
	s.Start()
	defer s.Stop()

	projectStructure, err := generateProjectStructure(ctx, client, config)
	if err != nil {
		return fmt.Errorf("error generating project structure: %w", err)
	}

	if config.DryRun {
		fmt.Println("Dry run: Project structure")
		for _, file := range projectStructure {
			fmt.Printf("=== %s ===\n%s\n\n", file.Name, file.Content)
		}
		return nil
	}

	if err := createProjectFiles(config.OutputDir, projectStructure); err != nil {
		return fmt.Errorf("error creating project files: %w", err)
	}

	fmt.Printf("Project generated successfully in %s\n", config.OutputDir)
	return nil
}

type ProjectFile struct {
	Name    string
	Content string
}

func generateProjectStructure(ctx context.Context, client llms.Model, config Config) ([]ProjectFile, error) {
	prompt := fmt.Sprintf(`Generate a complete Go project structure based on the following description:

Description: %s
Project Type: %s

Provide the content for each file in the project, including:
1. main.go
2. Additional package files
3. Test files
4. README.md
5. go.mod

Ensure that the generated code follows Go best practices and includes proper documentation.`, config.Description, config.ProjectTemplate)

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
		return nil, fmt.Errorf("error generating content: %w", err)
	}

	content := resp.Choices[0].Content

	var projectFiles []ProjectFile
	lines := strings.Split(content, "\n")
	var currentFile *ProjectFile

	for _, line := range lines {
		if strings.HasPrefix(line, "===") && strings.HasSuffix(line, "===") {
			if currentFile != nil {
				projectFiles = append(projectFiles, *currentFile)
			}
			fileName := strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(line, "==="), "==="))
			currentFile = &ProjectFile{Name: fileName}
		} else if currentFile != nil {
			currentFile.Content += line + "\n"
		}
	}

	if currentFile != nil {
		projectFiles = append(projectFiles, *currentFile)
	}

	return projectFiles, nil
}

func createProjectFiles(outputDir string, files []ProjectFile) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(files))

	for _, file := range files {
		wg.Add(1)
		go func(f ProjectFile) {
			defer wg.Done()
			filePath := filepath.Join(outputDir, f.Name)
			dir := filepath.Dir(filePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				errChan <- fmt.Errorf("error creating directory for %s: %w", f.Name, err)
				return
			}
			if err := os.WriteFile(filePath, []byte(f.Content), 0644); err != nil {
				errChan <- fmt.Errorf("error writing file %s: %w", f.Name, err)
				return
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
