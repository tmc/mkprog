package main

import (
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

	output, err := runCommand(command, commandArgs...)
	if err == nil {
		// If the command succeeds (exits 0), do nothing and exit
		return nil
	}

	fmt.Printf("Command failed. Error: %v\nOutput: %s\n", err, output)

	// If there's an error, suggest an improveprog or fixprog invocation
	llm, err := openai.New()
	if err != nil {
		return fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	suggestion, err := getSuggestion(llm, command, commandArgs, err.Error(), output, description)
	if err != nil {
		return fmt.Errorf("failed to get suggestion: %w", err)
	}

	fmt.Println("Suggested command:")
	fmt.Println(suggestion)
	return nil
}

func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func getSuggestion(llm llms.LLM, command string, args []string, errMsg, output, description string) (string, error) {
	context := fmt.Sprintf("Command: %s %s\nError: %s\nOutput: %s\nDescription: %s",
		command, strings.Join(args, " "), errMsg, output, description)

	prompt := fmt.Sprintf(`Given the following context of a failed command execution:

%s

Suggest an appropriate 'improveprog' or 'fixprog' invocation to address the issue.
If a description is provided, use it to guide your suggestion.
Provide only the suggested command without any additional explanation.`, context)

	response, err := llm.Call(prompt)
	if err != nil {
		return "", fmt.Errorf("AI request failed: %w", err)
	}

	// Clean up the response
	suggestion := strings.TrimSpace(response)
	suggestion = strings.Split(suggestion, "\n")[0] // Take only the first line

	return suggestion, nil
}