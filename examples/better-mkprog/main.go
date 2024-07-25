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
	"regexp"
	"sort"
	"sync"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

//go:embed system-prompt.txt
var systemPrompt string

type ProgramVersion struct {
	Name        string
	Description string
	Score       float64
	TestResults string
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	progName := flag.String("name", "", "Name of the program to generate")
	progDesc := flag.String("desc", "", "Description of the program to generate")
	iterations := flag.Int("n", 5, "Number of iterations to run tests")
	verbose := flag.Bool("v", false, "Verbose output")
	flag.Parse()

	if *progName == "" || *progDesc == "" {
		return fmt.Errorf("program name and description are required")
	}

	client, err := anthropic.New()
	if err != nil {
		return fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	versions := make([]ProgramVersion, *iterations)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := 0; i < *iterations; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			version, err := generateAndTest(client, *progName, *progDesc, *verbose)
			if err != nil {
				log.Printf("Error in iteration %d: %v", index, err)
				return
			}
			mu.Lock()
			versions[index] = version
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Score > versions[j].Score
	})

	bestVersion := versions[0]
	fmt.Printf("Best performing version:\n")
	fmt.Printf("Name: %s\n", bestVersion.Name)
	fmt.Printf("Description: %s\n", bestVersion.Description)
	fmt.Printf("Score: %.2f\n", bestVersion.Score)

	fmt.Printf("\nSummary of all versions:\n")
	for i, v := range versions {
		fmt.Printf("%d. Name: %s, Score: %.2f\n", i+1, v.Name, v.Score)
	}

	return analyzeResults(client, versions)
}

func generateAndTest(client llms.Model, name, desc string, verbose bool) (ProgramVersion, error) {
	version := ProgramVersion{
		Name:        name,
		Description: desc,
	}

	if verbose {
		log.Printf("Generating program: %s", name)
	}

	err := generateProgram(client, name, desc)
	if err != nil {
		return version, fmt.Errorf("failed to generate program: %w", err)
	}

	if verbose {
		log.Printf("Compiling program: %s", name)
	}

	err = compileProgram(name)
	if err != nil {
		return version, fmt.Errorf("failed to compile program: %w", err)
	}

	if verbose {
		log.Printf("Running tests for program: %s", name)
	}

	testResults, err := runTests(name)
	if err != nil {
		return version, fmt.Errorf("failed to run tests: %w", err)
	}

	version.TestResults = testResults
	version.Score = calculateScore(testResults)

	if verbose {
		log.Printf("Saving test results for program: %s", name)
	}

	err = saveTestResults(name, testResults)
	if err != nil {
		return version, fmt.Errorf("failed to save test results: %w", err)
	}

	return version, nil
}

func generateProgram(client llms.Model, name, desc string) error {
	ctx := context.Background()
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("%s %s", name, desc)),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(4000))
	if err != nil {
		return err
	}

	content := resp.Choices[0].Content
	re := regexp.MustCompile(`(?s)=== (.+?) ===\n(.+?)(?:(?:\n===)|$)`)
	matches := re.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		filename := match[1]
		fileContent := match[2]
		err := os.WriteFile(filename, []byte(fileContent), 0644)
		if err != nil {
			return fmt.Errorf("failed to write file %s: %w", filename, err)
		}
	}

	return nil
}

func compileProgram(name string) error {
	cmd := exec.Command("go", "build", "-o", name)
	return cmd.Run()
}

func runTests(name string) (string, error) {
	cmd := exec.Command("go", "test", "-v", "./...")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tests failed: %w", err)
	}
	return string(output), nil
}

func calculateScore(testResults string) float64 {
	re := regexp.MustCompile(`--- PASS: `)
	passedTests := len(re.FindAllString(testResults, -1))
	return float64(passedTests)
}

func saveTestResults(name, results string) error {
	cmd := exec.Command("git", "notes", "add", "-m", results, name)
	return cmd.Run()
}

func analyzeResults(client llms.Model, versions []ProgramVersion) error {
	ctx := context.Background()
	versionsJSON, err := json.Marshal(versions)
	if err != nil {
		return fmt.Errorf("failed to marshal versions: %w", err)
	}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, "You are an AI assistant specialized in analyzing test results for Go programs."),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Analyze the following test results and provide insights: %s", string(versionsJSON))),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(2000))
	if err != nil {
		return fmt.Errorf("failed to generate analysis: %w", err)
	}

	fmt.Printf("\nAnalysis of test results:\n%s\n", resp.Choices[0].Content)
	return nil
}
