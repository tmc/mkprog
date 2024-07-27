package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

//go:embed system-prompt.txt
var systemPrompt string

type Result struct {
	Attempt int    `json:"attempt"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	planFile := flag.String("plan", "", "Path to the plan file")
	maxAttempts := flag.Int("attempts", 10, "Maximum number of improvement attempts")
	parallelism := flag.Int("parallel", 3, "Number of parallel executions")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()

	if *planFile == "" {
		return fmt.Errorf("plan file is required")
	}

	plan, err := os.ReadFile(*planFile)
	if err != nil {
		return fmt.Errorf("failed to read plan file: %w", err)
	}

	results := make(chan Result, *maxAttempts)
	var wg sync.WaitGroup

	for i := 0; i < *parallelism; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for attempt := 1; attempt <= *maxAttempts; attempt++ {
				select {
				case results <- executeInDocker(workerID, attempt, string(plan), *verbose):
				default:
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(results)

	allResults := []Result{}
	for result := range results {
		allResults = append(allResults, result)
	}

	conclusion, err := analyzeResults(allResults)
	if err != nil {
		return fmt.Errorf("failed to analyze results: %w", err)
	}

	fmt.Println("Conclusion:")
	fmt.Println(conclusion)

	return nil
}

func executeInDocker(workerID, attempt int, plan string, verbose bool) Result {
	containerName := fmt.Sprintf("parallelprog_worker_%d_%d", workerID, attempt)

	if verbose {
		log.Printf("Worker %d: Starting attempt %d\n", workerID, attempt)
	}

	// Create a temporary directory for the plan file
	tmpDir, err := os.MkdirTemp("", "parallelprog")
	if err != nil {
		return Result{Attempt: attempt, Error: fmt.Sprintf("failed to create temp dir: %v", err)}
	}
	defer os.RemoveAll(tmpDir)

	planPath := filepath.Join(tmpDir, "plan.txt")
	if err := os.WriteFile(planPath, []byte(plan), 0644); err != nil {
		return Result{Attempt: attempt, Error: fmt.Sprintf("failed to write plan file: %v", err)}
	}

	if verbose {
		log.Printf("Worker %d: Running Docker container for attempt %d\n", workerID, attempt)
	}

	// Run the Docker container
	cmd := exec.Command("docker", "run", "--rm", "--name", containerName,
		"-v", fmt.Sprintf("%s:/plan.txt", planPath),
		"alpine", "sh", "-c", "cat /plan.txt && echo 'Executed plan in Docker'")

	output, err := cmd.CombinedOutput()
	if err != nil {
		if verbose {
			log.Printf("Worker %d: Docker execution failed for attempt %d: %v\n", workerID, attempt, err)
		}
		return Result{Attempt: attempt, Error: fmt.Sprintf("Docker execution failed: %v", err)}
	}

	if verbose {
		log.Printf("Worker %d: Completed attempt %d\n", workerID, attempt)
	}

	return Result{Attempt: attempt, Output: string(output)}
}

func analyzeResults(results []Result) (string, error) {
	ctx := context.Background()
	client, err := anthropic.New()
	if err != nil {
		return "", fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	resultsJSON, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal results: %w", err)
	}

	prompt := fmt.Sprintf("Analyze the following results from parallel improvement attempts and provide a concise conclusion:\n\n%s", string(resultsJSON))

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(1000))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	return strings.TrimSpace(resp.Choices[0].Content), nil
}