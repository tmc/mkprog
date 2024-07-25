package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: try <command>")
	}

	command := strings.Join(os.Args[1:], " ")
	originalBranch, err := getCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	newBranch := fmt.Sprintf("try-%d", time.Now().Unix())
	if err := createAndCheckoutBranch(newBranch); err != nil {
		return fmt.Errorf("failed to create and checkout new branch: %w", err)
	}

	defer func() {
		if err := checkoutBranch(originalBranch); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to switch back to original branch: %v\n", err)
		}
		if err := deleteBranch(newBranch); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete temporary branch: %v\n", err)
		}
	}()

	output, exitStatus, err := executeCommand(command)
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	fmt.Print(output)

	commitMsg := fmt.Sprintf("Try: %s", command)
	commitHash, err := createCommit(commitMsg)
	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	noteContent := fmt.Sprintf("Command: %s\nExit Status: %d\nOutput:\n%s", command, exitStatus, output)
	if err := addGitNote(commitHash, noteContent); err != nil {
		return fmt.Errorf("failed to add git note: %w", err)
	}

	fmt.Printf("\nOperation summary:\n")
	fmt.Printf("Command: %s\n", command)
	fmt.Printf("Exit Status: %d\n", exitStatus)
	fmt.Printf("Result stored in commit: %s\n", commitHash)

	return nil
}

func getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func createAndCheckoutBranch(branch string) error {
	cmd := exec.Command("git", "checkout", "-b", branch)
	return cmd.Run()
}

func checkoutBranch(branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	return cmd.Run()
}

func deleteBranch(branch string) error {
	cmd := exec.Command("git", "branch", "-D", branch)
	return cmd.Run()
}

func executeCommand(command string) (string, int, error) {
	cmd := exec.Command("sh", "-c", command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	output := stdout.String() + stderr.String()
	exitStatus := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitStatus = exitError.ExitCode()
		} else {
			return output, -1, err
		}
	}

	return output, exitStatus, nil
}

func createCommit(message string) (string, error) {
	cmd := exec.Command("git", "commit", "--allow-empty", "-m", message)
	if err := cmd.Run(); err != nil {
		return "", err
	}

	cmd = exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func addGitNote(commitHash, content string) error {
	cmd := exec.Command("git", "notes", "add", "-m", content, commitHash)
	return cmd.Run()
}
