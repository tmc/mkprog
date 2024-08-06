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

var (
	cfgFile      string
	projectDesc  string
	outputDir    string
	apiKey       string
	templateFile string
	dryRun       bool
	aiModel      string
	projectType  string
	verbose      bool
	temperature  float64
	maxTokens    int
)

var rootCmd = &cobra.Command{
	Use:   "mkprog",
	Short: "Generate a complete Go project structure",
	Long: `mkprog is a CLI tool that generates a complete Go project structure based on a user-provided description.
It uses AI models to generate code and documentation for the project.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.mkprog.yaml)")
	rootCmd.Flags().StringVarP(&projectDesc, "description", "d", "", "Project description")
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory")
	rootCmd.Flags().StringVarP(&apiKey, "api-key", "k", "", "API key for AI model")
	rootCmd.Flags().StringVarP(&templateFile, "template", "t", "", "Custom template file")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Dry run (preview generated content)")
	rootCmd.Flags().StringVarP(&aiModel, "ai-model", "m", "anthropic", "AI model to use (anthropic, openai, cohere)")
	rootCmd.Flags().StringVarP(&projectType, "project-type", "p", "cli", "Project template (cli, web, library)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.Flags().Float64VarP(&temperature, "temperature", "", 0.1, "AI model temperature")
	rootCmd.Flags().IntVarP(&maxTokens, "max-tokens", "", 8192, "Maximum number of tokens for AI response")

	viper.BindPFlag("description", rootCmd.Flags().Lookup("description"))
	viper.BindPFlag("output", rootCmd.Flags().Lookup("output"))
	viper.BindPFlag("api-key", rootCmd.Flags().Lookup("api-key"))
	viper.BindPFlag("template", rootCmd.Flags().Lookup("template"))
	viper.BindPFlag("dry-run", rootCmd.Flags().Lookup("dry-run"))
	viper.BindPFlag("ai-model", rootCmd.Flags().Lookup("ai-model"))
	viper.BindPFlag("project-type", rootCmd.Flags().Lookup("project-type"))
	viper.BindPFlag("verbose", rootCmd.Flags().Lookup("verbose"))
	viper.BindPFlag("temperature", rootCmd.Flags().Lookup("temperature"))
	viper.BindPFlag("max-tokens", rootCmd.Flags().Lookup("max-tokens"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".mkprog")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	if projectDesc == "" {
		return fmt.Errorf("project description is required")
	}

	if outputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}

	client, err := anthropic.New(anthropic.WithApiKey(apiKey), anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35))
	if err != nil {
		return fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Generating project structure..."
	s.Start()

	projectStructure, err := generateProjectStructure(client)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to generate project structure: %w", err)
	}

	s.Stop()

	if dryRun {
		fmt.Println("Dry run: Generated project structure")
		fmt.Printf("%+v\n", projectStructure)
		return nil
	}

	if err := createProjectFiles(projectStructure); err != nil {
		return fmt.Errorf("failed to create project files: %w", err)
	}

	fmt.Println("Project generated successfully!")
	return nil
}

func generateProjectStructure(client llms.Model) (map[string]string, error) {
	ctx := context.Background()

	systemPrompt := fmt.Sprintf(`You are an AI assistant specialized in creating Go project structures. Generate a complete project structure for a Go project based on the following description:

Description: %s

Project Type: %s

Include the following files and directories:
1. main.go (if applicable)
2. Additional package directories and files
3. Test files for each package
4. README.md
5. go.mod

Provide the content for each file as a JSON object where the keys are file paths and the values are the file contents.`, projectDesc, projectType)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, "Generate the project structure and file contents."),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(temperature), llms.WithMaxTokens(maxTokens))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	var projectStructure map[string]string
	if err := json.Unmarshal([]byte(resp.Choices[0].Content), &projectStructure); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project structure: %w", err)
	}

	return projectStructure, nil
}

func createProjectFiles(projectStructure map[string]string) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(projectStructure))

	for filePath, content := range projectStructure {
		wg.Add(1)
		go func(fp, c string) {
			defer wg.Done()
			if err := writeFile(fp, c); err != nil {
				errChan <- fmt.Errorf("failed to write file %s: %w", fp, err)
			}
		}(filePath, content)
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

func writeFile(filePath, content string) error {
	fullPath := filepath.Join(outputDir, filePath)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := ioutil.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	if verbose {
		fmt.Printf("Created file: %s\n", fullPath)
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
	files := []string{"main.go", "go.mod"}
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
