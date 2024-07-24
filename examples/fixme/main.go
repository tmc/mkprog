package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
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

	// Remove the "--" if it's the first argument
	if args[0] == "--" {
		args = args[1:]
	}

	if len(args) == 0 {
		return fmt.Errorf("no command specified")
	}

	// Find the index of the second "--" if it exists
	descriptionIndex := -1
	for i, arg := range args {
		if arg == "--" {
			descriptionIndex = i
			break
		}
	}

	var command string
	var commandArgs []string
	var description string

	if descriptionIndex != -1 {
		command = args[0]
		commandArgs = args[1:descriptionIndex]
		description = strings.Join(args[descriptionIndex+1:], " ")
	} else {
		command = args[0]
		commandArgs = args[1:]
	}

	llm, err := openai.New()
	if err != nil {
		return fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	fixHistory := []string{}

	for {
		output, err := runCommand(command, commandArgs...)
		if err == nil {
			// If the command succeeds (exits 0), exit the loop
			return nil
		}

		fmt.Printf("Command failed. Error: %v\nOutput: %s\n", err, output)

		suggestion, err := getSuggestion(llm, command, commandArgs, err.Error(), output, description, fixHistory)
		if err != nil {
			return fmt.Errorf("failed to get suggestion: %w", err)
		}

		fmt.Println("Suggested command:")
		fmt.Println(suggestion)

		// Add the suggestion to the fix history
		fixHistory = append(fixHistory, suggestion)

		// Ask the user if they want to continue
		fmt.Print("Do you want to apply this suggestion? (y/n): ")
		var response string
		fmt.Scanln(&response)

		if strings.ToLower(response) != "y" {
			return nil
		}

		// Parse the suggestion and update the command and arguments
		suggestedArgs := strings.Fields(suggestion)
		if len(suggestedArgs) > 1 && suggestedArgs[0] == "fixprog" {
			command = suggestedArgs[1]
			commandArgs = suggestedArgs[2:]
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

func getSuggestion(llm llms.LLM, command string, args []string, errMsg, output, description string, fixHistory []string) (string, error) {
	ctx := fmt.Sprintf("Command: %s %s\nError: %s\nOutput: %s\nDescription: %s",
		command, strings.Join(args, " "), errMsg, output, description)

	historyContext := strings.Join(fixHistory, "\n")

	prompt := fmt.Sprintf(`Given the following context of a failed command execution:

%s

Fix history:
%s

Suggest an appropriate 'fixprog' invocation to address the issue.
If a description is provided, use it to guide your suggestion.
Include the -hist flag with smart fixme instructions based on the fix history.
Provide only the suggested command without any additional explanation.`, ctx, historyContext)

	response, err := llm.Call(context.Background(), prompt)
	if err != nil {
		return "", fmt.Errorf("AI request failed: %w", err)
	}

	// Clean up the response
	suggestion := strings.TrimSpace(response)
	suggestion = strings.Split(suggestion, "\n")[0] // Take only the first line

	return suggestion, nil
}
