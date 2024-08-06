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
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.config/mkprog")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("MKPROG")

	bindFlags(rootCmd)

	return rootCmd.Execute()
}

func bindFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *cobra.Flag) {
		if !f.Changed && viper.IsSet(f.Name) {
			val := viper.Get(f.Name)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}

func generateProject(cmd *cobra.Command, args []string) error {
	description := args[0]

	if config.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	if config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Generating project..."
	s.Start()
	defer s.Stop()

	ctx := context.Background()
	client, err := anthropic.New(anthropic.WithAPIKey(config.APIKey))
	if err != nil {
		return fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	projectStructure, err := generateProjectStructure(ctx, client, description)
	if err != nil {
		return fmt.Errorf("failed to generate project structure: %w", err)
	}

	if config.DryRun {
		return previewContent(projectStructure)
	}

	return createProject(projectStructure)
}

func generateProjectStructure(ctx context.Context, client llms.Model, description string) ([]FileContent, error) {
	prompt := fmt.Sprintf("%s\n\nProject Description: %s\nProject Template: %s", systemPrompt, description, config.ProjectTemplate)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, prompt),
		llms.TextParts(llms.ChatMessageTypeHuman, "Generate the project structure and content for the described project."),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(8000))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	var projectStructure []FileContent
	err = json.Unmarshal([]byte(resp.Choices[0].Content), &projectStructure)
	if err != nil {
		return nil, fmt.Errorf("failed to parse generated content: %w", err)
	}

	return projectStructure, nil
}

func previewContent(projectStructure []FileContent) error {
	for _, file := range projectStructure {
		fmt.Printf("=== %s ===\n", file.Name)
		fmt.Println(file.Content)
		fmt.Println()
	}
	return nil
}

func createProject(projectStructure []FileContent) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(projectStructure))

	for _, file := range projectStructure {
		wg.Add(1)
		go func(f FileContent) {
			defer wg.Done()
			if err := writeFile(f); err != nil {
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

	return nil
}

func writeFile(file FileContent) error {
	mutex.Lock()
	defer mutex.Unlock()

	filePath := filepath.Join(config.OutputDir, file.Name)
	dir := filepath.Dir(filePath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	return ioutil.WriteFile(filePath, []byte(file.Content), 0644)
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
		fmt.Printf("=== %s ===\n", file)
		fmt.Println(string(content))
		fmt.Println()
	}
}
