package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

//go:embed system-prompt.txt
var systemPrompt string

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "mkprog [flags] <project description>",
	Short: "Generate a complete Go project structure based on a description",
	Long: `mkprog is a CLI tool that generates a complete Go project structure based on a user-provided description.
It uses the Anthropic API (or other selected AI model) to generate code and documentation for the project.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectDesc = args[0]
		if err := run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.mkprog.yaml)")
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "", "output directory for the generated project")
	rootCmd.Flags().StringVarP(&apiKey, "api-key", "k", "", "API key for the selected AI model")
	rootCmd.Flags().StringVarP(&templateFile, "template", "t", "", "custom template file")
	rootCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "preview generated content without creating files")
	rootCmd.Flags().StringVarP(&aiModel, "ai-model", "m", "anthropic", "AI model to use (anthropic, openai, cohere)")
	rootCmd.Flags().StringVarP(&projectType, "project-type", "p", "cli", "project template (cli, web, library)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.Flags().Float64VarP(&temperature, "temperature", "", 0.1, "AI model temperature (0.0 - 1.0)")
	rootCmd.Flags().IntVarP(&maxTokens, "max-tokens", "", 8192, "maximum number of tokens for AI response")

	viper.BindPFlag("output", rootCmd.Flags().Lookup("output"))
	viper.BindPFlag("api-key", rootCmd.Flags().Lookup("api-key"))
	viper.BindPFlag("ai-model", rootCmd.Flags().Lookup("ai-model"))
	viper.BindPFlag("project-type", rootCmd.Flags().Lookup("project-type"))
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

func run() error {
	if outputDir == "" {
		outputDir = viper.GetString("output")
	}
	if apiKey == "" {
		apiKey = viper.GetString("api-key")
	}
	if aiModel == "" {
		aiModel = viper.GetString("ai-model")
	}
	if projectType == "" {
		projectType = viper.GetString("project-type")
	}

	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}

	if outputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	if !dryRun {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
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
		fmt.Printf("%s\n", projectStructure)
		return nil
	}

	if err := createProjectFiles(client, projectStructure); err != nil {
		return fmt.Errorf("failed to create project files: %w", err)
	}

	fmt.Printf("Project generated successfully in %s\n", outputDir)
	return nil
}

func generateProjectStructure(client llms.ChatLLM) (string, error) {
	ctx := context.Background()
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Generate a project structure for a %s Go project with the following description: %s", projectType, projectDesc)),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(temperature), llms.WithMaxTokens(maxTokens))
	if err != nil {
		return "", err
	}

	return resp.Choices[0].Content, nil
}

func createProjectFiles(client llms.ChatLLM, projectStructure string) error {
	var files []string
	err := json.Unmarshal([]byte(projectStructure), &files)
	if err != nil {
		return fmt.Errorf("failed to parse project structure: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(files))

	for _, file := range files {
		wg.Add(1)
		go func(filename string) {
			defer wg.Done()
			if err := generateFile(client, filename); err != nil {
				errChan <- fmt.Errorf("failed to generate %s: %w", filename, err)
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

func generateFile(client llms.ChatLLM, filename string) error {
	ctx := context.Background()
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Generate the content for the file %s in the %s Go project with the following description: %s", filename, projectType, projectDesc)),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(temperature), llms.WithMaxTokens(maxTokens))
	if err != nil {
		return err
	}

	content := resp.Choices[0].Content

	if strings.HasSuffix(filename, ".go") {
		content, err = formatGoCode(content)
		if err != nil {
			return fmt.Errorf("failed to format Go code: %w", err)
		}
	}

	filePath := filepath.Join(outputDir, filename)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", filename, err)
	}

	if err := ioutil.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filename, err)
	}

	if verbose {
		fmt.Printf("Generated %s\n", filename)
	}

	return nil
}

func formatGoCode(code string) (string, error) {
	// TODO: Implement Go code formatting using go/format or goimports
	return code, nil
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
