package main

import (
	"bufio"
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/exec"
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

type config struct {
	OutputDir      string
	APIKey         string
	TemplateFile   string
	DryRun         bool
	AIModel        string
	ProjectType    string
	Description    string
	Temperature    float64
	MaxTokens      int
	Verbose        bool
	Interactive    bool
	InitGit        bool
	GenerateDocker bool
}

var cfg config

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	rootCmd := &cobra.Command{
		Use:   "mkprog [description]",
		Short: "Generate a Go project structure based on a description",
		Long:  `mkprog is a CLI tool that generates a complete Go project structure based on a user-provided description using AI-powered code generation.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runGenerate,
	}

	rootCmd.Flags().StringVarP(&cfg.OutputDir, "output", "o", "", "Output directory for the generated project")
	rootCmd.Flags().StringVarP(&cfg.APIKey, "api-key", "k", "", "API key for the AI service")
	rootCmd.Flags().StringVarP(&cfg.TemplateFile, "template", "t", "", "Custom template file")
	rootCmd.Flags().BoolVarP(&cfg.DryRun, "dry-run", "d", false, "Perform a dry run without creating files")
	rootCmd.Flags().StringVarP(&cfg.AIModel, "ai-model", "m", "anthropic", "AI model to use (anthropic, openai, cohere)")
	rootCmd.Flags().StringVarP(&cfg.ProjectType, "project-type", "p", "cli", "Project template (cli, web, library)")
	rootCmd.Flags().Float64VarP(&cfg.Temperature, "temperature", "", 0.1, "AI model temperature")
	rootCmd.Flags().IntVarP(&cfg.MaxTokens, "max-tokens", "", 8192, "Maximum number of tokens for AI response")
	rootCmd.Flags().BoolVarP(&cfg.Verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.Flags().BoolVarP(&cfg.Interactive, "interactive", "i", false, "Enable interactive mode")
	rootCmd.Flags().BoolVar(&cfg.InitGit, "init-git", false, "Initialize Git repository")
	rootCmd.Flags().BoolVar(&cfg.GenerateDocker, "generate-docker", false, "Generate Dockerfile and docker-compose.yml")

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
	cfg.Description = args[0]

	if cfg.Interactive {
		if err := runInteractiveMode(); err != nil {
			return err
		}
	}

	if cfg.Verbose {
		log.Println("Configuration:")
		log.Printf("  Description: %s", cfg.Description)
		log.Printf("  Output Directory: %s", cfg.OutputDir)
		log.Printf("  AI Model: %s", cfg.AIModel)
		log.Printf("  Project Type: %s", cfg.ProjectType)
		log.Printf("  Dry Run: %v", cfg.DryRun)
	}

	if cfg.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	if cfg.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	client, err := anthropic.New(anthropic.WithApiKey(cfg.APIKey), anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35))
	if err != nil {
		return fmt.Errorf("error creating Anthropic client: %w", err)
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Generating project structure..."
	s.Start()

	projectStructure, err := generateProjectStructure(client)
	if err != nil {
		s.Stop()
		return fmt.Errorf("error generating project structure: %w", err)
	}

	s.Stop()

	if cfg.DryRun {
		fmt.Println("Dry run: Project structure")
		fmt.Println(projectStructure)
		return nil
	}

	if err := createProjectFiles(projectStructure); err != nil {
		return fmt.Errorf("error creating project files: %w", err)
	}

	if cfg.InitGit {
		if err := initGitRepository(); err != nil {
			return fmt.Errorf("error initializing Git repository: %w", err)
		}
	}

	if cfg.GenerateDocker {
		if err := generateDockerFiles(client); err != nil {
			return fmt.Errorf("error generating Docker files: %w", err)
		}
	}

	fmt.Printf("Project generated successfully in %s\n", cfg.OutputDir)
	return nil
}

func generateProjectStructure(client llms.Model) (string, error) {
	ctx := context.Background()
	prompt := fmt.Sprintf(`%s

Project Description: %s
Project Type: %s

Generate a complete Go project structure based on the above description and project type. Include the following:

1. Main package with necessary files
2. Additional packages as needed
3. Test files for each package
4. README.md content
5. go.mod file content

Provide the output in the following format:

===filename===
file content
===filename===
file content
...

Ensure that the generated code follows Go best practices and includes proper documentation.`, systemPrompt, cfg.Description, cfg.ProjectType)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(cfg.Temperature), llms.WithMaxTokens(cfg.MaxTokens))
	if err != nil {
		return "", fmt.Errorf("error generating content: %w", err)
	}

	return resp.Choices[0].Content, nil
}

func createProjectFiles(projectStructure string) error {
	files := strings.Split(projectStructure, "===")
	var wg sync.WaitGroup
	errChan := make(chan error, len(files))

	for i := 1; i < len(files); i += 2 {
		filename := strings.TrimSpace(files[i])
		content := strings.TrimSpace(files[i+1])

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

func writeFile(filename, content string) error {
	fullPath := filepath.Join(cfg.OutputDir, filename)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating directory %s: %w", dir, err)
	}

	return os.WriteFile(fullPath, []byte(content), 0644)
}

func initGitRepository() error {
	if err := runCommand(cfg.OutputDir, "git", "init"); err != nil {
		return fmt.Errorf("error initializing Git repository: %w", err)
	}

	if err := runCommand(cfg.OutputDir, "git", "add", "."); err != nil {
		return fmt.Errorf("error adding files to Git repository: %w", err)
	}

	if err := runCommand(cfg.OutputDir, "git", "commit", "-m", "Initial commit"); err != nil {
		return fmt.Errorf("error creating initial commit: %w", err)
	}

	return nil
}

func generateDockerFiles(client llms.Model) error {
	ctx := context.Background()
	prompt := fmt.Sprintf(`Generate a Dockerfile and docker-compose.yml file for the following Go project:

Project Description: %s
Project Type: %s

Provide the output in the following format:

===Dockerfile===
Dockerfile content
===docker-compose.yml===
docker-compose.yml content`, cfg.Description, cfg.ProjectType)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(cfg.Temperature), llms.WithMaxTokens(cfg.MaxTokens))
	if err != nil {
		return fmt.Errorf("error generating Docker files: %w", err)
	}

	dockerFiles := strings.Split(resp.Choices[0].Content, "===")
	for i := 1; i < len(dockerFiles); i += 2 {
		filename := strings.TrimSpace(dockerFiles[i])
		content := strings.TrimSpace(dockerFiles[i+1])

		if err := writeFile(filename, content); err != nil {
			return fmt.Errorf("error writing file %s: %w", filename, err)
		}
	}

	return nil
}

func runCommand(dir, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runInteractiveMode() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter project description: ")
	cfg.Description, _ = reader.ReadString('\n')
	cfg.Description = strings.TrimSpace(cfg.Description)

	fmt.Print("Enter output directory: ")
	cfg.OutputDir, _ = reader.ReadString('\n')
	cfg.OutputDir = strings.TrimSpace(cfg.OutputDir)

	fmt.Print("Enter AI model (anthropic, openai, cohere): ")
	cfg.AIModel, _ = reader.ReadString('\n')
	cfg.AIModel = strings.TrimSpace(cfg.AIModel)

	fmt.Print("Enter project type (cli, web, library): ")
	cfg.ProjectType, _ = reader.ReadString('\n')
	cfg.ProjectType = strings.TrimSpace(cfg.ProjectType)

	fmt.Print("Initialize Git repository? (y/n): ")
	initGit, _ := reader.ReadString('\n')
	cfg.InitGit = strings.ToLower(strings.TrimSpace(initGit)) == "y"

	fmt.Print("Generate Docker files? (y/n): ")
	genDocker, _ := reader.ReadString('\n')
	cfg.GenerateDocker = strings.ToLower(strings.TrimSpace(genDocker)) == "y"

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
