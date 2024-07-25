package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

//go:embed system-prompt.txt
var systemPrompt string

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	attempts := flag.Int("attempts", 3, "number of historical points to try improving")
	depth := flag.Int("depth", 10, "how far back in history to look for improvement points")
	flag.Parse()

	if flag.NArg() == 0 {
		return fmt.Errorf("no command provided")
	}

	command := flag.Args()
	if err := executeCommand(command); err == nil {
		fmt.Println("Command executed successfully")
		return nil
	}

	fmt.Println("Command failed. Attempting to improve previous commits...")

	commits, err := getRelevantCommits(*depth)
	if err != nil {
		return fmt.Errorf("failed to get relevant commits: %w", err)
	}

	for i := 0; i < *attempts && i < len(commits); i++ {
		commit := commits[i]
		fmt.Printf("Attempting to improve commit %s\n", commit)

		if err := createBranch(commit); err != nil {
			fmt.Printf("Failed to create branch from commit %s: %v\n", commit, err)
			continue
		}

		if err := improveCode(commit); err != nil {
			fmt.Printf("Failed to improve code at commit %s: %v\n", commit, err)
			if err := cleanupBranch(); err != nil {
				fmt.Printf("Failed to cleanup branch: %v\n", err)
			}
			continue
		}

		if err := reapplyCommits(commit); err != nil {
			fmt.Printf("Failed to reapply commits: %v\n", err)
			if err := cleanupBranch(); err != nil {
				fmt.Printf("Failed to cleanup branch: %v\n", err)
			}
			continue
		}

		if err := executeCommand(command); err == nil {
			fmt.Println("Command executed successfully after improvements")
			if err := mergeBranch(); err != nil {
				return fmt.Errorf("failed to merge improved branch: %w", err)
			}
			return nil
		}

		if err := cleanupBranch(); err != nil {
			fmt.Printf("Failed to cleanup branch: %v\n", err)
		}
	}

	return fmt.Errorf("failed to improve the code after %d attempts", *attempts)
}

func executeCommand(command []string) error {
	cmd := exec.Command("try", command...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getRelevantCommits(depth int) ([]string, error) {
	cmd := exec.Command("git", "log", "-n", fmt.Sprintf("%d", depth), "--pretty=format:%H")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	commits := strings.Split(strings.TrimSpace(string(output)), "\n")
	return filterRelevantCommits(commits)
}

func filterRelevantCommits(commits []string) ([]string, error) {
	var relevantCommits []string
	for _, commit := range commits {
		cmd := exec.Command("git", "show", "--name-only", "--format=", commit)
		output, err := cmd.Output()
		if err != nil {
			return nil, err
		}

		if isRelevantCommit(string(output)) {
			relevantCommits = append(relevantCommits, commit)
		}
	}
	return relevantCommits, nil
}

func isRelevantCommit(commitInfo string) bool {
	relevantFiles := []string{".go", ".json", ".yaml", ".yml", "Dockerfile", "Makefile"}
	for _, file := range relevantFiles {
		if strings.Contains(commitInfo, file) {
			return true
		}
	}
	return false
}

func createBranch(commit string) error {
	branchName := fmt.Sprintf("timeforge-improvement-%s", time.Now().Format("20060102-150405"))
	cmd := exec.Command("git", "checkout", "-b", branchName, commit)
	return cmd.Run()
}

func improveCode(commit string) error {
	files, err := getChangedFiles(commit)
	if err != nil {
		return err
	}

	for _, file := range files {
		if err := improveFile(file); err != nil {
			return err
		}
	}

	return commitChanges(fmt.Sprintf("Improve code at %s", commit))
}

func getChangedFiles(commit string) ([]string, error) {
	cmd := exec.Command("git", "diff-tree", "--no-commit-id", "--name-only", "-r", commit)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSpace(string(output)), "\n"), nil
}

func improveFile(file string) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	improvedContent, err := improveContent(string(content))
	if err != nil {
		return err
	}

	return os.WriteFile(file, []byte(improvedContent), 0644)
}

func improveContent(content string) (string, error) {
	ctx := context.Background()
	client, err := anthropic.New()
	if err != nil {
		return "", err
	}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Improve the following code:\n\n%s", content)),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(4000))
	if err != nil {
		return "", err
	}

	return extractCodeFromResponse(resp.Choices[0].Content), nil
}

func extractCodeFromResponse(response string) string {
	re := regexp.MustCompile("```(?:go)?\n((?s).*?)\n```")
	matches := re.FindStringSubmatch(response)
	if len(matches) > 1 {
		return matches[1]
	}
	return response
}

func commitChanges(message string) error {
	cmd := exec.Command("git", "add", ".")
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("git", "commit", "-m", message)
	return cmd.Run()
}

func reapplyCommits(startCommit string) error {
	cmd := exec.Command("git", "rev-list", "--reverse", fmt.Sprintf("%s..HEAD", startCommit))
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	commits := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, commit := range commits {
		cmd := exec.Command("git", "cherry-pick", commit)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to cherry-pick commit %s: %w", commit, err)
		}
	}

	return nil
}

func mergeBranch() error {
	currentBranch, err := getCurrentBranch()
	if err != nil {
		return err
	}

	cmd := exec.Command("git", "checkout", "main")
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("git", "merge", currentBranch)
	if err := cmd.Run(); err != nil {
		return err
	}

	return cleanupBranch()
}

func getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func cleanupBranch() error {
	currentBranch, err := getCurrentBranch()
	if err != nil {
		return err
	}

	cmd := exec.Command("git", "checkout", "main")
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("git", "branch", "-D", currentBranch)
	return cmd.Run()
}
