package main

import (
	"context"
	"fmt"
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

var config Config

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
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

	rootCmd.Flags().StringVar(&config.OutputDir, "output", "", "Output directory for the generated project")
	rootCmd.Flags().StringVar(&config.APIKey, "api-key", "", "API key for the AI service")
	rootCmd.Flags().StringVar(&config.CustomTemplate, "template", "", "Custom template file")
	rootCmd.Flags().BoolVar(&config.DryRun, "dry-run", false, "Preview generated content without creating files")
	rootCmd.Flags().StringVar(&config.AIModel, "ai-model", "anthropic", "AI model to use (anthropic, openai, cohere)")
	rootCmd.Flags().StringVar(&config.ProjectTemplate, "project-type", "cli", "Project template (cli, web, library)")
	rootCmd.Flags().IntVar(&config.MaxTokens, "max-tokens", 8192, "Maximum number of tokens for AI generation")
	rootCmd.Flags().Float64Var(&config.Temperature, "temperature", 0.1, "Temperature for AI generation")

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

	ctx := context.Background()
	client, err := anthropic.New(anthropic.WithAPIKey(config.APIKey), anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35))
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
		for _, file := range projectStructure {
			fmt.Printf("File: %s\nContent:\n%s\n\n", file.Path, file.Content)
		}
		return nil
	}

	if err := createProjectFiles(projectStructure); err != nil {
		return fmt.Errorf("error creating project files: %w", err)
	}

	fmt.Printf("Project generated successfully in %s\n", config.OutputDir)
	return nil
}

type ProjectFile struct {
	Path    string
	Content string
}

func generateProjectStructure(ctx context.Context, client *anthropic.Chat, description string) ([]ProjectFile, error) {
	prompt := fmt.Sprintf(`Generate a complete Go project structure based on the following description:

%s

Provide the content for each file in the project, including:
1. main.go
2. Additional package files
3. Test files
4. README.md
5. go.mod

For each file, use the following format:
===filename===
(file content)

Ensure that the generated code follows Go best practices and is idiomatic.`, description)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(config.Temperature), llms.WithMaxTokens(config.MaxTokens))
	if err != nil {
		return nil, fmt.Errorf("error generating content: %w", err)
	}

	content := resp.Choices[0].Content

	var files []ProjectFile
	parts := strings.Split(content, "===")
	for i := 1; i < len(parts); i += 2 {
		filename := strings.TrimSpace(parts[i])
		fileContent := strings.TrimSpace(parts[i+1])
		files = append(files, ProjectFile{
			Path:    filename,
			Content: fileContent,
		})
	}

	return files, nil
}

func createProjectFiles(files []ProjectFile) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(files))

	for _, file := range files {
		wg.Add(1)
		go func(f ProjectFile) {
			defer wg.Done()
			fullPath := filepath.Join(config.OutputDir, f.Path)
			dir := filepath.Dir(fullPath)

			if err := os.MkdirAll(dir, 0755); err != nil {
				errChan <- fmt.Errorf("error creating directory %s: %w", dir, err)
				return
			}

			if err := os.WriteFile(fullPath, []byte(f.Content), 0644); err != nil {
				errChan <- fmt.Errorf("error writing file %s: %w", fullPath, err)
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

func init() {
	if os.Getenv("_MKPROG_DUMP") != "" {
		dumpsrc()
		os.Exit(0)
	}
}

func dumpsrc() {
	fmt.Println("=== go.mod ===")
	fmt.Println(goModContent)
	fmt.Println("=== main.go ===")
	fmt.Println(mainGoContent)
	fmt.Println("=== system-prompt.txt ===")
	fmt.Println(systemPromptContent)
}

var goModContent = `module github.com/yourusername/mkprog

go 1.21

require (
	github.com/briandowns/spinner v1.23.0
	github.com/spf13/cobra v1.7.0
	github.com/spf13/viper v1.16.0
	github.com/tmc/langchaingo v0.1.13-0.20240725041451-1975058648b5
)

require (
	github.com/dlclark/regexp2 v1.10.0 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/pelletier/go-toml/v2 v2.0.8 // indirect
	github.com/pkoukk/tiktoken-go v0.1.6 // indirect
	github.com/spf13/afero v1.9.5 // indirect
	github.com/spf13/cast v1.5.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.4.2 // indirect
	golang.org/x/sys v0.8.0 // indirect
	golang.org/x/term v0.1.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)`

var mainGoContent = `package main

import (
	"context"
	"fmt"
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

var config Config

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	rootCmd := &cobra.Command{
		Use:   "mkprog [project description]",
		Short: "Generate a complete Go project structure based on a description",
		Long:  ` + "`" + `mkprog is a tool that generates a complete Go project structure based on a user-provided description using AI-powered code generation.` + "`" + `,
		Args:  cobra.ExactArgs(1),
		RunE:  runGenerator,
	}

	rootCmd.Flags().StringVar(&config.OutputDir, "output", "", "Output directory for the generated project")
	rootCmd.Flags().StringVar(&config.APIKey, "api-key", "", "API key for the AI service")
	rootCmd.Flags().StringVar(&config.CustomTemplate, "template", "", "Custom template file")
	rootCmd.Flags().BoolVar(&config.DryRun, "dry-run", false, "Preview generated content without creating files")
	rootCmd.Flags().StringVar(&config.AIModel, "ai-model", "anthropic", "AI model to use (anthropic, openai, cohere)")
	rootCmd.Flags().StringVar(&config.ProjectTemplate, "project-type", "cli", "Project template (cli, web, library)")
	rootCmd.Flags().IntVar(&config.MaxTokens, "max-tokens", 8192, "Maximum number of tokens for AI generation")
	rootCmd.Flags().Float64Var(&config.Temperature, "temperature", 0.1, "Temperature for AI generation")

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

	ctx := context.Background()
	client, err := anthropic.New(anthropic.WithAPIKey(config.APIKey), anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35))
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
		for _, file := range projectStructure {
			fmt.Printf("File: %s\nContent:\n%s\n\n", file.Path, file.Content)
		}
		return nil
	}

	if err := createProjectFiles(projectStructure); err != nil {
		return fmt.Errorf("error creating project files: %w", err)
	}

	fmt.Printf("Project generated successfully in %s\n", config.OutputDir)
	return nil
}

type ProjectFile struct {
	Path    string
	Content string
}

func generateProjectStructure(ctx context.Context, client *anthropic.Chat, description string) ([]ProjectFile, error) {
	prompt := fmt.Sprintf(` + "`" + `Generate a complete Go project structure based on the following description:

%s

Provide the content for each file in the project, including:
1. main.go
2. Additional package files
3. Test files
4. README.md
5. go.mod

For each file, use the following format:
===filename===
(file content)

Ensure that the generated code follows Go best practices and is idiomatic.` + "`" + `, description)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(config.Temperature), llms.WithMaxTokens(config.MaxTokens))
	if err != nil {
		return nil, fmt.Errorf("error generating content: %w", err)
	}

	content := resp.Choices[0].Content

	var files []ProjectFile
	parts := strings.Split(content, "===")
	for i := 1; i < len(parts); i += 2 {
		filename := strings.TrimSpace(parts[i])
		fileContent := strings.TrimSpace(parts[i+1])
		files = append(files, ProjectFile{
			Path:    filename,
			Content: fileContent,
		})
	}

	return files, nil
}

func createProjectFiles(files []ProjectFile) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(files))

	for _, file := range files {
		wg.Add(1)
		go func(f ProjectFile) {
			defer wg.Done()
			fullPath := filepath.Join(config.OutputDir, f.Path)
			dir := filepath.Dir(fullPath)

			if err := os.MkdirAll(dir, 0755); err != nil {
				errChan <- fmt.Errorf("error creating directory %s: %w", dir, err)
				return
			}

			if err := os.WriteFile(fullPath, []byte(f.Content), 0644); err != nil {
				errChan <- fmt.Errorf("error writing file %s: %w", fullPath, err)
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

func init() {
	if os.Getenv("_MKPROG_DUMP") != "" {
		dumpsrc()
		os.Exit(0)
	}
}

func dumpsrc() {
	fmt.Println("=== go.mod ===")
	fmt.Println(goModContent)
	fmt.Println("=== main.go ===")
	fmt.Println(mainGoContent)
	fmt.Println("=== system-prompt.txt ===")
	fmt.Println(systemPromptContent)
}

var goModContent = ` + "`" + `module github.com/yourusername/mkprog

go 1.21

require (
	github.com/briandowns/spinner v1.23.0
	github.com/spf13/cobra v1.7.0
	github.com/spf13/viper v1.16.0
	github.com/tmc/langchaingo v0.1.13-0.20240725041451-1975058648b5
)

require (
	github.com/dlclark/regexp2 v1.10.0 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/pelletier/go-toml/v2 v2.0.8 // indirect
	github.com/pkoukk/tiktoken-go v0.1.6 // indirect
	github.com/spf13/afero v1.9.5 // indirect
	github.com/spf13/cast v1.5.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.4.2 // indirect
	golang.org/x/sys v0.8.0 // indirect
	golang.org/x/term v0.1.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)` + "`" + `

var systemPromptContent = ` + "`" + `You are an AI assistant specialized in creating Go programs. Your task is to generate complete, functional Go code for programs based on user descriptions. Follow these guidelines:

1. Create a main package with a main() function as the entry point.
2. Implement error handling using the run() pattern to separate concerns.
3. Use appropriate libraries and packages for the requested functionality.
4. Implement command-line argument handling to accept user input.
5. Ensure the program follows Go best practices and idiomatic patterns.
6. Create a go.mod file with necessary dependencies.
7. Create a README.md file explaining the program's purpose, installation, and usage.
8. Include an MIT license file for the project.
9. Implement proper logging for debugging and monitoring.
10. Use concurrent processing when appropriate for improved performance.
11. Implement unit tests for the main functionality.
12. Add comments to explain non-obvious parts of the code.
13. Ensure all imports are used and all types are correctly referenced.
14. Maintain consistent and proper indentation throughout the entire generated code.

When generating a program, create all necessary files and code to make a complete, runnable Go project. The generated program should be fully functional, well-structured, and adhere to Go best practices.` + "`" + `
`
