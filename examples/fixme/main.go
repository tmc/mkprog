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

	// Parse flags
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
			i = len(args) // Exit the loop
		}
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

		suggestion, fixDescription, err := getSuggestion(llm, command, commandArgs, err.Error(), output, description, fixHistory)
		if err != nil {
			return fmt.Errorf("failed to get suggestion: %w", err)
		}

		fmt.Println("Suggested command:")
		fmt.Println(suggestion)
		fmt.Println("Fix description:")
		fmt.Println(fixDescription)

		// Add the suggestion to the fix history if useHistory is true
		if useHistory {
			fixHistory = append(fixHistory, suggestion)
		}

		// Ask the user if they want to continue
		fmt.Print("Do you want to apply this suggestion? (y/n): ")
		var response string
		fmt.Scanln(&response)

		if strings.ToLower(response) != "y" {
			return nil
		}

		// Parse the suggestion and update the command and arguments
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

func getSuggestion(llm llms.LLM, command string, args []string, errMsg, output, description string, fixHistory []string) (string, string, error) {
	ctx := fmt.Sprintf("Command: %s %s\nError: %s\nOutput: %s\nDescription: %s",
		command, strings.Join(args, " "), errMsg, output, description)

	historyContext := strings.Join(fixHistory, "\n")

	prompt := fmt.Sprintf(`fixme is a general-purpose tool that suggests fixes for failed shell commands. Here's the command being run and its output:

%s

Fix history:
%s

Please suggest a fix for this command, explaining your reasoning. The suggestion should be in the format of a shell command that can be directly executed.

Provide the suggested command on the first line and a brief description of the fix on subsequent lines.`, ctx, historyContext)

	response, err := llm.Call(context.Background(), prompt)
	if err != nil {
		return "", "", fmt.Errorf("AI request failed: %w", err)
	}

	// Clean up the response
	lines := strings.Split(strings.TrimSpace(response), "\n")
	suggestion := lines[0]
	fixDescription := ""

	if len(lines) > 1 {
		fixDescription = strings.Join(lines[1:], "\n")
	}

	return suggestion, fixDescription, nil
}
