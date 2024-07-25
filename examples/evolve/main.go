package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

//go:embed system-prompt.txt
var systemPrompt string

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	testFlag := flag.Bool("test", false, "Run tests after implementing changes")
	attemptsFlag := flag.Int("attempts", 1, "Number of improvement attempts")
	evaluateFlag := flag.Bool("evaluate", false, "Evaluate the changes")
	improveFlag := flag.Bool("improve", false, "Attempt to improve the changes")
	maxRecursiveAttemptsFlag := flag.Int("max-recursive-attempts", 10, "Maximum number of recursive self-improvement attempts")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		return fmt.Errorf("no description provided")
	}

	if args[0] == "--" {
		if len(args) < 3 || args[1] != "evolve" {
			return fmt.Errorf("invalid format for recursive evolution")
		}
		return recursiveEvolve(strings.Join(args[2:], " "), *maxRecursiveAttemptsFlag)
	}

	description := strings.Join(args, " ")
	return evolve(description, *testFlag, *attemptsFlag, *evaluateFlag, *improveFlag)
}

func evolve(description string, test bool, attempts int, evaluate, improve bool) error {
	client, err := anthropic.New()
	if err != nil {
		return fmt.Errorf("failed to create Anthropic client: %v", err)
	}

	ctx := context.Background()

	for i := 0; i < attempts; i++ {
		fmt.Printf("Attempt %d/%d\n", i+1, attempts)

		changes, err := generateChanges(ctx, client, description)
		if err != nil {
			return fmt.Errorf("failed to generate changes: %v", err)
		}

		if err := applyChanges(changes); err != nil {
			return fmt.Errorf("failed to apply changes: %v", err)
		}

		if test {
			if err := runTests(); err != nil {
				fmt.Printf("Tests failed: %v\n", err)
				if !improve {
					return err
				}
			} else {
				fmt.Println("Tests passed successfully")
			}
		}

		if evaluate {
			evaluation, err := evaluateChanges(ctx, client, changes)
			if err != nil {
				return fmt.Errorf("failed to evaluate changes: %v", err)
			}
			fmt.Printf("Evaluation: %s\n", evaluation)
		}

		if improve {
			improvedChanges, err := improveChanges(ctx, client, changes, evaluation)
			if err != nil {
				return fmt.Errorf("failed to improve changes: %v", err)
			}
			if err := applyChanges(improvedChanges); err != nil {
				return fmt.Errorf("failed to apply improved changes: %v", err)
			}
		}

		if err := commitChanges(description); err != nil {
			return fmt.Errorf("failed to commit changes: %v", err)
		}
	}

	return nil
}

func recursiveEvolve(task string, maxAttempts int) error {
	for i := 0; i < maxAttempts; i++ {
		fmt.Printf("Recursive evolution attempt %d/%d\n", i+1, maxAttempts)

		if err := evolve(task, true, 1, true, true); err != nil {
			fmt.Printf("Evolution attempt failed: %v\n", err)
			continue
		}

		if canPerformTask(task) {
			fmt.Printf("Successfully evolved to perform the task: %s\n", task)
			return nil
		}
	}

	return fmt.Errorf("failed to evolve to perform the task after %d attempts", maxAttempts)
}

func generateChanges(ctx context.Context, client *anthropic.Client, description string) (string, error) {
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Implement the following change: %s", description)),
	}
	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(4000))
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Content, nil
}

func applyChanges(changes string) error {
	re := regexp.MustCompile(`=== (.+) ===\n([\s\S]+?)(?:\n===|$)`)
	matches := re.FindAllStringSubmatch(changes, -1)

	for _, match := range matches {
		filename := match[1]
		content := strings.TrimSpace(match[2])

		if err := ioutil.WriteFile(filename, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %v", filename, err)
		}
	}

	return nil
}

func runTests() error {
	cmd := exec.Command("go", "test", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func evaluateChanges(ctx context.Context, client *anthropic.Client, changes string) (string, error) {
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Evaluate the following changes:\n%s", changes)),
	}
	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(2000))
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Content, nil
}

func improveChanges(ctx context.Context, client *anthropic.Client, changes, evaluation string) (string, error) {
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Improve the following changes based on the evaluation:\nChanges:\n%s\nEvaluation:\n%s", changes, evaluation)),
	}
	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(4000))
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Content, nil
}

func commitChanges(description string) error {
	if err := exec.Command("git", "add", ".").Run(); err != nil {
		return fmt.Errorf("failed to stage changes: %v", err)
	}

	commitMsg := fmt.Sprintf("Evolve: %s", description)
	if err := exec.Command("git", "commit", "-m", commitMsg).Run(); err != nil {
		return fmt.Errorf("failed to commit changes: %v", err)
	}

	return nil
}

func canPerformTask(task string) bool {
	if task == "print a readme for evolve" {
		_, err := os.Stat("README.md")
		return err == nil
	}
	// Add more task-specific checks here
	return false
}
