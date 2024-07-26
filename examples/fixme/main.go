package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]
	if len(args) == 0 {
		return fmt.Errorf("usage: fixme [--] <command> [args...] [-- <description>]")
	}

	if args[0] == "--" {
		args = args[1:]
	}

	if len(args) == 0 {
		return fmt.Errorf("no command specified")
	}

	var useHistory bool
	var description string
	var command string
	var commandArgs []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-hist":
			useHistory = true
		case "-desc":
			if i+1 < len(args) {
				description = args[i+1]
				i++
			} else {
				return fmt.Errorf("-desc flag requires a value")
			}
		default:
			command = args[i]
			commandArgs = args[i+1:]
			i = len(args)
		}
	}

	llm, err := anthropic.New()
	if err != nil {
		return fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	fixHistory := []string{}
	failedSuggestions := make(map[string]bool)

	for {
		output, err := runCommand(command, commandArgs...)
		if err == nil {
			return nil
		}

		fmt.Printf("Command failed. Error: %v\nOutput: %s\n", err, output)

		suggestion, fixDescription, err := getSuggestion(llm, command, commandArgs, err.Error(), output, description, fixHistory, failedSuggestions)
		if err != nil {
			return fmt.Errorf("failed to get suggestion: %w", err)
		}

		fmt.Println("Suggested command:")
		fmt.Println(suggestion)
		fmt.Println("Fix description:")
		fmt.Println(fixDescription)

		if useHistory {
			fixHistory = append(fixHistory, suggestion)
		}

		fmt.Print("Do you want to apply this suggestion? (y/n): ")
		var response string
		fmt.Scanln(&response)

		if strings.ToLower(response) != "y" {
			return nil
		}

		failedSuggestions[suggestion] = true

		suggestedArgs := strings.Fields(suggestion)
		if len(suggestedArgs) > 0 {
			command = suggestedArgs[0]
			commandArgs = suggestedArgs[1:]
		} else {
			fmt.Println("Invalid suggestion format. Exiting.")
			return nil
		}
	}
}

func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

const systemPrompt = `
You are an AI assistant specialized in debugging Go module issues. Your task is to analyze command outputs, especially those related to 'go get', 'go mod tidy', and other Go module commands. Consider the following when suggesting fixes:

1. Package paths may change between major versions (e.g., golang.org/x/... to go.uber.org/...).
2. Some packages might be split into sub-modules in newer versions.
3. Always check if a specific version constraint is needed (e.g., @v1.2.3 instead of @latest).
4. Consider suggesting 'go mod tidy' to clean up and update dependencies.
5. For version conflicts, suggest updating to compatible versions or using replace directives in go.mod.
6. If a package is not found, suggest checking for typos or searching for alternative import paths.

Provide concise, actionable suggestions that can be directly executed as shell commands. Explain your reasoning briefly but clearly.
`

func getSuggestion(llm llms.LLM, command string, args []string, errMsg, output, description string, fixHistory []string, failedSuggestions map[string]bool) (string, string, error) {
	ctx := fmt.Sprintf("Command: %s %s\nError: %s\nOutput: %s\nDescription: %s",
		command, strings.Join(args, " "), errMsg, output, description)

	historyContext := strings.Join(fixHistory, "\n")

	prompt := fmt.Sprintf(`%s

fixme is a general-purpose tool that suggests fixes for failed shell commands, with a focus on Go module issues. Here's the command being run and its output:

%s

Fix history:
%s

Please suggest a fix for this command, considering the following:
1. The package structure or version might have changed in newer releases.
2. Consider suggesting alternative versions or package paths.
3. Avoid repeating previously failed suggestions.
4. If dealing with Go modules, consider using 'go mod tidy' or 'go get' with specific versions.

Provide the suggested command on the first line and a brief description of the fix on subsequent lines.`, systemPrompt, ctx, historyContext)

	response, err := llm.Call(context.Background(), prompt)
	if err != nil {
		return "", "", fmt.Errorf("AI request failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(response), "\n")
	suggestion := lines[0]
	fixDescription := ""

	if len(lines) > 1 {
		fixDescription = strings.Join(lines[1:], "\n")
	}

	if failedSuggestions[suggestion] {
		return getSuggestion(llm, command, args, errMsg, output, description, fixHistory, failedSuggestions)
	}

	return suggestion, fixDescription, nil
}
