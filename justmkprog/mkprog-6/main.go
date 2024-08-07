package main

import (
	"context"
	_ "embed"
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
	OutputDir       string
	APIKey          string
	TemplateFile    string
	DryRun          bool
	AIModel         string
	ProjectType     string
	Description     string
	Temperature     float64
	MaxTokens       int
	Verbose         bool
	InteractiveMode bool
}

var config Config

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	rootCmd := &cobra.Command{
		Use:   "mkprog [description]",
		Short: "Generate a Go project structure based on a description",
		Long:  `mkprog is a tool that generates a complete Go project structure based on a user-provided description using AI-powered code generation.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runGenerator,
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
	rootCmd.Flags().BoolVarP(&config.InteractiveMode, "interactive", "i", false, "Enable interactive mode")

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

func runGenerator(cmd *cobra.Command, args []string) error {
	config.Description = args[0]

	if config.InteractiveMode {
		if err := runInteractiveMode(); err != nil {
			return err
		}
	}

	if config.Verbose {
		log.Println("Configuration:")
		log.Printf("  Output Directory: %s", config.OutputDir)
		log.Printf("  AI Model: %s", config.AIModel)
		log.Printf("  Project Type: %s", config.ProjectType)
		log.Printf("  Dry Run: %v", config.DryRun)
		log.Printf("  Description: %s", config.Description)
	}

	if config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	if config.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	if !config.DryRun {
		if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	client, err := anthropic.New(anthropic.WithApiKey(config.APIKey))
	if err != nil {
		return fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Generating project structure..."
	s.Start()

	projectStructure, err := generateProjectStructure(client)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to generate project structure: %w", err)
	}

	s.Stop()

	if config.DryRun {
		fmt.Println("Dry run: Project structure")
		fmt.Println(projectStructure)
		return nil
	}

	if err := createProjectFiles(projectStructure); err != nil {
		return fmt.Errorf("failed to create project files: %w", err)
	}

	fmt.Printf("Project generated successfully in %s\n", config.OutputDir)
	return nil
}

func generateProjectStructure(client llms.Model) (string, error) {
	ctx := context.Background()
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Generate a Go project structure for a %s project with the following description: %s", config.ProjectType, config.Description)),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(config.Temperature), llms.WithMaxTokens(config.MaxTokens))
	if err != nil {
		return "", err
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
			filePath := filepath.Join(config.OutputDir, filename)
			dir := filepath.Dir(filePath)

			if err := os.MkdirAll(dir, 0755); err != nil {
				errChan <- fmt.Errorf("failed to create directory %s: %w", dir, err)
				return
			}

			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				errChan <- fmt.Errorf("failed to write file %s: %w", filePath, err)
				return
			}

			if config.Verbose {
				log.Printf("Created file: %s", filePath)
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

func parseProjectStructure(projectStructure string) map[string]string {
	files := make(map[string]string)
	lines := strings.Split(projectStructure, "\n")
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

func runInteractiveMode() error {
	fmt.Println("Interactive Mode")
	fmt.Println("----------------")

	if config.OutputDir == "" {
		fmt.Print("Enter output directory: ")
		fmt.Scanln(&config.OutputDir)
	}

	if config.ProjectType == "" {
		fmt.Print("Enter project type (cli, web, library): ")
		fmt.Scanln(&config.ProjectType)
	}

	if config.Description == "" {
		fmt.Print("Enter project description: ")
		fmt.Scanln(&config.Description)
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
	fmt.Println(string(mainSrc))
	fmt.Println("=== go.mod ===")
	fmt.Println(string(goModSrc))
	fmt.Println("=== system-prompt.txt ===")
	fmt.Println(systemPrompt)
}

//go:embed main.go
var mainSrc []byte

//go:embed go.mod
var goModSrc []byte
