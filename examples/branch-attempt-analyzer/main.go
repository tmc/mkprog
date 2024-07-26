package main

import (
	"context"
	"flag"
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

//go:embed system-prompt.txt
var systemPrompt string

type RunResult struct {
	Attempts []AttemptResult
	Analysis string
}

type AttemptResult struct {
	Output string
	Error  error
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	attempts := flag.Int("attempts", 10, "Number of attempts per run")
	runs := flag.Int("runs", 1, "Number of meta-comparison runs")
	branchName := flag.String("branch", "attempt-branch", "Name of the branch to use for attempts")
	flag.Parse()

	if flag.NArg() < 1 {
		return fmt.Errorf("please provide a task command as an argument")
	}
	taskCommand := strings.Join(flag.Args(), " ")

	results := make([]RunResult, *runs)
	for i := 0; i < *runs; i++ {
		result, err := performRun(*attempts, *branchName, taskCommand)
		if err != nil {
			return fmt.Errorf("error in run %d: %w", i+1, err)
		}
		results[i] = result
		fmt.Printf("Run %d completed. Analysis:\n%s\n\n", i+1, result.Analysis)
	}

	if *runs > 1 {
		metaAnalysis, err := performMetaAnalysis(results)
		if err != nil {
			return fmt.Errorf("error in meta-analysis: %w", err)
		}
		fmt.Printf("Meta-analysis of %d runs:\n%s\n", *runs, metaAnalysis)
	}

	return nil
}

func performRun(attempts int, branchName, taskCommand string) (RunResult, error) {
	originalBranch, err := getCurrentBranch()
	if err != nil {
		return RunResult{}, fmt.Errorf("failed to get current branch: %w", err)
	}

	if err := createAndCheckoutBranch(branchName); err != nil {
		return RunResult{}, fmt.Errorf("failed to create and checkout branch: %w", err)
	}

	defer func() {
		if err := checkoutAndDeleteBranch(originalBranch, branchName); err != nil {
			log.Printf("Warning: failed to cleanup branch: %v", err)
		}
	}()

	attemptResults := make([]AttemptResult, attempts)
	for i := 0; i < attempts; i++ {
		output, err := exec.Command("sh", "-c", taskCommand).CombinedOutput()
		attemptResults[i] = AttemptResult{
			Output: string(output),
			Error:  err,
		}
	}

	analysis, err := analyzeResults(attemptResults)
	if err != nil {
		return RunResult{}, fmt.Errorf("failed to analyze results: %w", err)
	}

	return RunResult{
		Attempts: attemptResults,
		Analysis: analysis,
	}, nil
}

func getCurrentBranch() (string, error) {
	output, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func createAndCheckoutBranch(branchName string) error {
	if err := exec.Command("git", "checkout", "-b", branchName).Run(); err != nil {
		return err
	}
	return nil
}

func checkoutAndDeleteBranch(originalBranch, branchName string) error {
	if err := exec.Command("git", "checkout", originalBranch).Run(); err != nil {
		return fmt.Errorf("failed to checkout original branch: %w", err)
	}
	if err := exec.Command("git", "branch", "-D", branchName).Run(); err != nil {
		return fmt.Errorf("failed to delete temporary branch: %w", err)
	}
	return nil
}

func analyzeResults(results []AttemptResult) (string, error) {
	client, err := anthropic.NewChat()
	if err != nil {
		return "", fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	ctx := context.Background()
	resultStr := formatResultsForAnalysis(results)
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Analyze the following attempt results:\n\n%s", resultStr)),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(1000))
	if err != nil {
		return "", fmt.Errorf("failed to generate analysis: %w", err)
	}

	return resp.Choices[0].Content, nil
}

func formatResultsForAnalysis(results []AttemptResult) string {
	var sb strings.Builder
	for i, result := range results {
		sb.WriteString(fmt.Sprintf("Attempt %d:\n", i+1))
		sb.WriteString(fmt.Sprintf("Output: %s\n", result.Output))
		if result.Error != nil {
			sb.WriteString(fmt.Sprintf("Error: %v\n", result.Error))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func performMetaAnalysis(results []RunResult) (string, error) {
	client, err := anthropic.NewChat()
	if err != nil {
		return "", fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	ctx := context.Background()
	resultStr := formatRunResultsForMetaAnalysis(results)
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Perform a meta-analysis of the following run results:\n\n%s", resultStr)),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(1500))
	if err != nil {
		return "", fmt.Errorf("failed to generate meta-analysis: %w", err)
	}

	return resp.Choices[0].Content, nil
}

func formatRunResultsForMetaAnalysis(results []RunResult) string {
	var sb strings.Builder
	for i, result := range results {
		sb.WriteString(fmt.Sprintf("Run %d Analysis:\n%s\n\n", i+1, result.Analysis))
	}
	return sb.String()
}

