package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 3 || os.Args[1] != "--" {
		return fmt.Errorf("Usage: %s -- <command> [args...]", os.Args[0])
	}

	command := strings.Join(os.Args[2:], " ")

	// Run the command
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run command: %v", err)
	}

	// Create a git commit with the command
	if err := createGitCommit(command); err != nil {
		return fmt.Errorf("failed to create git commit: %v", err)
	}

	return nil
}

func createGitCommit(command string) error {
	// Check if we're in a git repository
	if err := exec.Command("git", "rev-parse", "--is-inside-work-tree").Run(); err != nil {
		return fmt.Errorf("not in a git repository")
	}

	// Stage all changes
	stageCmd := exec.Command("git", "add", ".")
	if err := stageCmd.Run(); err != nil {
		return fmt.Errorf("failed to stage changes: %v", err)
	}

	// Create a commit with the command as the message
	commitCmd := exec.Command("git", "commit", "-m", fmt.Sprintf("Run: %s", command))
	commitCmd.Env = append(os.Environ(), "GIT_NOTES_REF=refs/notes/metadata")
	if err := commitCmd.Run(); err != nil {
		return fmt.Errorf("failed to create commit: %v", err)
	}

	// Add a git note with the command
	noteCmd := exec.Command("git", "notes", "--ref=metadata", "add", "-m", command)
	if err := noteCmd.Run(); err != nil {
		return fmt.Errorf("failed to add git note: %v", err)
	}

	fmt.Println("Git commit and note created successfully.")
	return nil
}
