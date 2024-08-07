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
	Description    string
	OutputDir      string
	APIKey         string
	TemplateFile   string
	DryRun         bool
	AIModel        string
	ProjectType    string
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
		Use:   "mkprog [flags] <project description>",
		Short: "Generate a Go project structure based on a description",
		Long:  `mkprog is a tool that generates a complete Go project structure based on a user-provided description using AI-powered code generation.`,
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

	if cfg.Verbose {
		log.Println("Configuration:")
		log.Printf("Description: %s", cfg.Description)
		log.Printf("Output Directory: %s", cfg.OutputDir)
		log.Printf("AI Model: %s", cfg.AIModel)
		log.Printf("Project Type: %s", cfg.ProjectType)
		log.Printf("Dry Run: %v", cfg.DryRun)
		log.Printf("Interactive: %v", cfg.Interactive)
		log.Printf("Init Git: %v", cfg.InitGit)
		log.Printf("Generate Docker: %v", cfg.GenerateDocker)
	}

	if cfg.Interactive {
		if err := runInteractiveMode(); err != nil {
			return fmt.Errorf("error in interactive mode: %w", err)
		}
	}

	if cfg.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	if cfg.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	client, err := anthropic.New(anthropic.WithApiKey(cfg.APIKey))
	if err != nil {
		return fmt.Errorf("error creating Anthropic client: %w", err)
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
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

	fmt.Println("Project generated successfully!")
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

Provide the content for each file, separated by file paths. Use the following format:

===file:path/to/file.go===
(file content here)
===endfile===

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
	files := strings.Split(projectStructure, "===file:")

	var wg sync.WaitGroup
	errChan := make(chan error, len(files))

	for _, file := range files[1:] {
		wg.Add(1)
		go func(fileContent string) {
			defer wg.Done()

			parts := strings.SplitN(fileContent, "===", 2)
			if len(parts) != 2 {
				errChan <- fmt.Errorf("invalid file content format")
				return
			}

			filePath := strings.TrimSpace(parts[0])
			content := strings.TrimSpace(strings.TrimSuffix(parts[1], "endfile==="))

			fullPath := filepath.Join(cfg.OutputDir, filePath)
			dir := filepath.Dir(fullPath)

			if err := os.MkdirAll(dir, 0755); err != nil {
				errChan <- fmt.Errorf("error creating directory %s: %w", dir, err)
				return
			}

			if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
				errChan <- fmt.Errorf("error writing file %s: %w", fullPath, err)
				return
			}

			if cfg.Verbose {
				log.Printf("Created file: %s", fullPath)
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

func runInteractiveMode() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter project description: ")
	cfg.Description, _ = reader.ReadString('\n')
	cfg.Description = strings.TrimSpace(cfg.Description)

	fmt.Print("Enter output directory: ")
	cfg.OutputDir, _ = reader.ReadString('\n')
	cfg.OutputDir = strings.TrimSpace(cfg.OutputDir)

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

func initGitRepository() error {
	cmd := exec.Command("git", "init")
	cmd.Dir = cfg.OutputDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error initializing Git repository: %w", err)
	}

	if cfg.Verbose {
		log.Println("Initialized Git repository")
	}

	return nil
}

func generateDockerFiles(client llms.Model) error {
	ctx := context.Background()

	prompt := fmt.Sprintf(`Generate Dockerfile and docker-compose.yml files for the following Go project:

Project Description: %s
Project Type: %s

Provide the content for each file, separated by file paths. Use the following format:

===file:Dockerfile===
(Dockerfile content here)
===endfile===

===file:docker-compose.yml===
(docker-compose.yml content here)
===endfile===

Ensure that the generated files follow best practices for containerizing Go applications.`, cfg.Description, cfg.ProjectType)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(cfg.Temperature), llms.WithMaxTokens(cfg.MaxTokens))
	if err != nil {
		return fmt.Errorf("error generating Docker files: %w", err)
	}

	if err := createProjectFiles(resp.Choices[0].Content); err != nil {
		return fmt.Errorf("error creating Docker files: %w", err)
	}

	if cfg.Verbose {
		log.Println("Generated Dockerfile and docker-compose.yml")
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
