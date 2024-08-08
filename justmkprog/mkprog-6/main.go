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
		Long:  `mkprog is a tool that generates a complete Go project structure based on a user-provided description using AI language models.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateProject(args[0], &config)
		},
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

func generateProject(description string, config *Config) error {
	if config.Verbose {
		log.Println("Generating project with description:", description)
		log.Printf("Config: %+v\n", config)
	}

	client, err := anthropic.New(anthropic.WithApiKey(config.APIKey), anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35))
	if err != nil {
		return fmt.Errorf("error creating Anthropic client: %w", err)
	}

	ctx := context.Background()

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Generating project structure..."
	s.Start()

	projectStructure, err := generateProjectStructure(ctx, client, description, config)
	if err != nil {
		s.Stop()
		return fmt.Errorf("error generating project structure: %w", err)
	}

	s.Stop()

	if config.DryRun {
		fmt.Println("Dry run: Project structure")
		fmt.Println(projectStructure)
		return nil
	}

	if err := createProjectFiles(projectStructure, config); err != nil {
		return fmt.Errorf("error creating project files: %w", err)
	}

	fmt.Println("Project generated successfully!")
	return nil
}

func generateProjectStructure(ctx context.Context, client *anthropic.Client, description string, config *Config) (string, error) {
	prompt := fmt.Sprintf(`Generate a complete Go project structure based on the following description:

%s

The project should be a %s project. Include the main package, additional packages as needed, test files, README.md, and go.mod file.

Provide the project structure as a tree-like text representation, followed by the content of each file. Use the following format:

project/
├── main.go
├── pkg/
│   └── example/
│       └── example.go
├── README.md
└── go.mod

--- main.go ---
(content of main.go)

--- pkg/example/example.go ---
(content of example.go)

--- README.md ---
(content of README.md)

--- go.mod ---
(content of go.mod)

`, description, config.ProjectType)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(config.Temperature), llms.WithMaxTokens(config.MaxTokens))
	if err != nil {
		return "", fmt.Errorf("error generating content: %w", err)
	}

	return resp.Choices[0].Content, nil
}

func createProjectFiles(projectStructure string, config *Config) error {
	lines := strings.Split(projectStructure, "\n")
	var currentFile string
	var currentContent strings.Builder
	var wg sync.WaitGroup
	fileChan := make(chan struct {
		path    string
		content string
	})

	for _, line := range lines {
		if strings.HasPrefix(line, "---") && strings.HasSuffix(line, "---") {
			if currentFile != "" {
				wg.Add(1)
				go func(file, content string) {
					defer wg.Done()
					fileChan <- struct {
						path    string
						content string
					}{file, content}
				}(currentFile, currentContent.String())
				currentContent.Reset()
			}
			currentFile = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "---"), "---"))
		} else if currentFile != "" {
			currentContent.WriteString(line + "\n")
		}
	}

	if currentFile != "" {
		wg.Add(1)
		go func(file, content string) {
			defer wg.Done()
			fileChan <- struct {
				path    string
				content string
			}{file, content}
		}(currentFile, currentContent.String())
	}

	go func() {
		wg.Wait()
		close(fileChan)
	}()

	for file := range fileChan {
		fullPath := filepath.Join(config.OutputDir, file.path)
		dir := filepath.Dir(fullPath)

		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("error creating directory %s: %w", dir, err)
		}

		if err := writeFile(fullPath, file.content); err != nil {
			return fmt.Errorf("error writing file %s: %w", fullPath, err)
		}

		if config.Verbose {
			log.Printf("Created file: %s\n", fullPath)
		}
	}

	return nil
}

func writeFile(path, content string) error {
	f, err := os.Create(path)
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
