package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

//go:embed system-prompt.txt
var systemPrompt string

type ProgramVersion struct {
	Name        string
	Description string
	Score       float64
	TestResults map[string]bool
}

type Config struct {
	Tests []struct {
		Name    string
		Command string
	}
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	programName := flag.String("name", "", "Name of the program to generate")
	programDesc := flag.String("desc", "", "Description of the program to generate")
	iterations := flag.Int("n", 5, "Number of iterations to run tests")
	verbose := flag.Bool("v", false, "Verbose output")
	configFile := flag.String("config", "config.json", "Path to the configuration file")
	flag.Parse()

	if *programName == "" || *programDesc == "" {
		return fmt.Errorf("program name and description are required")
	}

	config, err := loadConfig(*configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %v", err)
	}

	versions := make([]ProgramVersion, *iterations)
	var wg sync.WaitGroup
	wg.Add(*iterations)

	for i := 0; i < *iterations; i++ {
		go func(index int) {
			defer wg.Done()
			version, err := generateAndTest(*programName, *programDesc, config, *verbose)
			if err != nil {
				log.Printf("Error in iteration %d: %v", index+1, err)
				return
			}
			versions[index] = version
		}(i)
	}

	wg.Wait()

	bestVersion := findBestVersion(versions)
	printSummary(versions, bestVersion)

	if err := analyzeResults(versions); err != nil {
		return fmt.Errorf("failed to analyze results: %v", err)
	}

	return nil
}

func generateAndTest(name, desc string, config Config, verbose bool) (ProgramVersion, error) {
	version := ProgramVersion{
		Name:        name,
		Description: desc,
		TestResults: make(map[string]bool),
	}

	if err := runMkprog(name, desc); err != nil {
		return version, fmt.Errorf("failed to generate program: %v", err)
	}

	if err := compileProgram(name); err != nil {
		return version, fmt.Errorf("failed to compile program: %v", err)
	}

	for _, test := range config.Tests {
		passed, err := runTest(name, test.Command)
		if err != nil {
			return version, fmt.Errorf("failed to run test '%s': %v", test.Name, err)
		}
		version.TestResults[test.Name] = passed
		if verbose {
			log.Printf("Test '%s': %v", test.Name, passed)
		}
	}

	version.Score = calculateScore(version.TestResults)

	if err := saveGitNote(name, version); err != nil {
		return version, fmt.Errorf("failed to save Git note: %v", err)
	}

	return version, nil
}

func runMkprog(name, desc string) error {
	cmd := exec.Command("mkprog", name, desc)
	return cmd.Run()
}

func compileProgram(name string) error {
	cmd := exec.Command("go", "build", "-o", name, name+".go")
	return cmd.Run()
}

func runTest(name, testCommand string) (bool, error) {
	cmd := exec.Command("sh", "-c", testCommand)
	err := cmd.Run()
	return err == nil, nil
}

func calculateScore(testResults map[string]bool) float64 {
	passedTests := 0
	for _, passed := range testResults {
		if passed {
			passedTests++
		}
	}
	return float64(passedTests) / float64(len(testResults))
}

func saveGitNote(name string, version ProgramVersion) error {
	noteContent, err := json.Marshal(version)
	if err != nil {
		return err
	}

	cmd := exec.Command("git", "notes", "add", "-m", string(noteContent), name+".go")
	return cmd.Run()
}

func findBestVersion(versions []ProgramVersion) ProgramVersion {
	bestVersion := versions[0]
	for _, v := range versions[1:] {
		if v.Score > bestVersion.Score {
			bestVersion = v
		}
	}
	return bestVersion
}

func printSummary(versions []ProgramVersion, bestVersion ProgramVersion) {
	fmt.Println("Summary of all versions:")
	for i, v := range versions {
		fmt.Printf("%d. %s (Score: %.2f)\n", i+1, v.Name, v.Score)
	}
	fmt.Printf("\nBest performing version: %s (Score: %.2f)\n", bestVersion.Name, bestVersion.Score)
}

func loadConfig(filename string) (Config, error) {
	var config Config
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(data, &config)
	return config, err
}

func analyzeResults(versions []ProgramVersion) error {
	client, err := anthropic.NewChat()
	if err != nil {
		return fmt.Errorf("failed to create Anthropic client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	analysisPrompt := fmt.Sprintf("Analyze the following test results for %d versions of a program:\n\n", len(versions))
	for i, v := range versions {
		analysisPrompt += fmt.Sprintf("Version %d:\nScore: %.2f\nTest Results: %v\n\n", i+1, v.Score, v.TestResults)
	}
	analysisPrompt += "Provide insights on the performance of different versions and suggest improvements."

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, analysisPrompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(4000))
	if err != nil {
		return fmt.Errorf("failed to generate analysis: %v", err)
	}

	fmt.Println("\nAnalysis of test results:")
	fmt.Println(resp.Choices[0].Content)

	return nil
}

