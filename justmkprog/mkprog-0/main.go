package main

import (
	"context"
	"fmt"
	"io"
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
	MaxTokens       int
	Temperature     float64
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
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
	rootCmd.Flags().IntVarP(&config.MaxTokens, "max-tokens", "x", 8192, "Maximum number of tokens for AI generation")
	rootCmd.Flags().Float64VarP(&config.Temperature, "temperature", "T", 0.1, "Temperature for AI generation")

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

	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	client, err := anthropic.New(anthropic.WithApiKey(config.APIKey), anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35))
	if err != nil {
		return fmt.Errorf("error creating Anthropic client: %w", err)
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Generating project structure..."
	s.Start()
	defer s.Stop()

	ctx := context.Background()
	projectStructure, err := generateProjectStructure(ctx, client, config)
	if err != nil {
		return fmt.Errorf("error generating project structure: %w", err)
	}

	if config.DryRun {
		fmt.Println("Dry run: Project structure")
		fmt.Println(projectStructure)
		return nil
	}

	files, err := generateFiles(ctx, client, config, projectStructure)
	if err != nil {
		return fmt.Errorf("error generating files: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(files))

	for path, content := range files {
		wg.Add(1)
		go func(path, content string) {
			defer wg.Done()
			if err := writeFile(config.OutputDir, path, content); err != nil {
				errChan <- fmt.Errorf("error writing file %s: %w", path, err)
			}
		}(path, content)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	fmt.Printf("Project generated successfully in %s\n", config.OutputDir)
	return nil
}

func generateProjectStructure(ctx context.Context, client llms.Model, config Config) (string, error) {
	prompt := fmt.Sprintf(`Generate a project structure for a Go project with the following description:

%s

The project should use the %s template. Provide a list of files and directories that should be created for this project.`, config.Description, config.ProjectTemplate)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(config.Temperature), llms.WithMaxTokens(config.MaxTokens))
	if err != nil {
		return "", fmt.Errorf("error generating project structure: %w", err)
	}

	return resp.Choices[0].Content, nil
}

func generateFiles(ctx context.Context, client llms.Model, config Config, projectStructure string) (map[string]string, error) {
	files := make(map[string]string)

	fileList := strings.Split(projectStructure, "\n")
	for _, file := range fileList {
		file = strings.TrimSpace(file)
		if file == "" || strings.HasSuffix(file, "/") {
			continue
		}

		prompt := fmt.Sprintf(`Generate the content for the file %s in a Go project with the following description:

%s

The project uses the %s template. Provide only the file content, without any additional formatting or explanations.`, file, config.Description, config.ProjectTemplate)

		messages := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
			llms.TextParts(llms.ChatMessageTypeHuman, prompt),
		}

		resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(config.Temperature), llms.WithMaxTokens(config.MaxTokens))
		if err != nil {
			return nil, fmt.Errorf("error generating content for file %s: %w", file, err)
		}

		files[file] = resp.Choices[0].Content
	}

	return files, nil
}

func writeFile(baseDir, path, content string) error {
	fullPath := filepath.Join(baseDir, path)
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
		return fmt.Errorf("error writing content to file %s: %w", fullPath, err)
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
	fmt.Println(systemPrompt)
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
	github.com/pkoukk/tiktoken-go v0.1.5 // indirect
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
	"io"
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
	MaxTokens       int
	Temperature     float64
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var config Config

	rootCmd := &cobra.Command{
		Use:   "mkprog [flags] <project description>",
		Short: "Generate a Go project structure based on a description",
		Long:  ` + "`" + `mkprog is a tool that generates a complete Go project structure based on a user-provided description using AI-powered code generation.` + "`" + `,
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
	rootCmd.Flags().IntVarP(&config.MaxTokens, "max-tokens", "x", 8192, "Maximum number of tokens for AI generation")
	rootCmd.Flags().Float64VarP(&config.Temperature, "temperature", "T", 0.1, "Temperature for AI generation")

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

	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	client, err := anthropic.New(anthropic.WithApiKey(config.APIKey), anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35))
	if err != nil {
		return fmt.Errorf("error creating Anthropic client: %w", err)
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Generating project structure..."
	s.Start()
	defer s.Stop()

	ctx := context.Background()
	projectStructure, err := generateProjectStructure(ctx, client, config)
	if err != nil {
		return fmt.Errorf("error generating project structure: %w", err)
	}

	if config.DryRun {
		fmt.Println("Dry run: Project structure")
		fmt.Println(projectStructure)
		return nil
	}

	files, err := generateFiles(ctx, client, config, projectStructure)
	if err != nil {
		return fmt.Errorf("error generating files: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(files))

	for path, content := range files {
		wg.Add(1)
		go func(path, content string) {
			defer wg.Done()
			if err := writeFile(config.OutputDir, path, content); err != nil {
				errChan <- fmt.Errorf("error writing file %s: %w", path, err)
			}
		}(path, content)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	fmt.Printf("Project generated successfully in %s\n", config.OutputDir)
	return nil
}

func generateProjectStructure(ctx context.Context, client llms.Model, config Config) (string, error) {
	prompt := fmt.Sprintf(` + "`" + `Generate a project structure for a Go project with the following description:

%s

The project should use the %s template. Provide a list of files and directories that should be created for this project.` + "`" + `, config.Description, config.ProjectTemplate)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(config.Temperature), llms.WithMaxTokens(config.MaxTokens))
	if err != nil {
		return "", fmt.Errorf("error generating project structure: %w", err)
	}

	return resp.Choices[0].Content, nil
}

func generateFiles(ctx context.Context, client llms.Model, config Config, projectStructure string) (map[string]string, error) {
	files := make(map[string]string)

	fileList := strings.Split(projectStructure, "\n")
	for _, file := range fileList {
		file = strings.TrimSpace(file)
		if file == "" || strings.HasSuffix(file, "/") {
			continue
		}

		prompt := fmt.Sprintf(` + "`" + `Generate the content for the file %s in a Go project with the following description:

%s

The project uses the %s template. Provide only the file content, without any additional formatting or explanations.` + "`" + `, file, config.Description, config.ProjectTemplate)

		messages := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
			llms.TextParts(llms.ChatMessageTypeHuman, prompt),
		}

		resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(config.Temperature), llms.WithMaxTokens(config.MaxTokens))
		if err != nil {
			return nil, fmt.Errorf("error generating content for file %s: %w", file, err)
		}

		files[file] = resp.Choices[0].Content
	}

	return files, nil
}

func writeFile(baseDir, path, content string) error {
	fullPath := filepath.Join(baseDir, path)
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
		return fmt.Errorf("error writing content to file %s: %w", fullPath, err)
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
	fmt.Println(systemPrompt)
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
	github.com/pkoukk/tiktoken-go v0.1.5 // indirect
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

var mainGoContent = ` + "`" + `package main

import (
	"context"
	"fmt"
	"io"
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
	MaxTokens       int
	Temperature     float64
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var config Config

	rootCmd := &cobra.Command{
		Use:   "mkprog [flags] <project description>",
		Short: "Generate a Go project structure based on a description",
		Long:  ` + "``" + `mkprog is a tool that generates a complete Go project structure based on a user-provided description using AI-powered code generation.` + "``" + `,
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
	rootCmd.Flags().IntVarP(&config.MaxTokens, "max-tokens", "x", 8192, "Maximum number of tokens for AI generation")
	rootCmd.Flags().Float64VarP(&config.Temperature, "temperature", "T", 0.1, "Temperature for AI generation")

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

	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	client, err := anthropic.New(anthropic.WithApiKey(config.APIKey), anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35))
	if err != nil {
		return fmt.Errorf("error creating Anthropic client: %w", err)
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Generating project structure..."
	s.Start()
	defer s.Stop()

	ctx := context.Background()
	projectStructure, err := generateProjectStructure(ctx, client, config)
	if err != nil {
		return fmt.Errorf("error generating project structure: %w", err)
	}

	if config.DryRun {
		fmt.Println("Dry run: Project structure")
		fmt.Println(projectStructure)
		return nil
	}

	files, err := generateFiles(ctx, client,