package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	verbose bool
	keep    bool
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	rootCmd := &cobra.Command{
		Use:   "try <command>",
		Short: "Safely experiment with changes in a Git repository",
		Long: `try allows developers to safely experiment with changes in a Git repository.
It creates a temporary branch, executes the given command, commits the changes,
and optionally deletes the temporary branch.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("command is required")
			}
			return tryCommand(args)
		},
	}

	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Provide more detailed output")
	rootCmd.Flags().BoolVarP(&keep, "keep", "k", false, "Keep the temporary branch instead of deleting it")

	return rootCmd.Execute()
}

func tryCommand(args []string) error {
	originalBranch, err := getCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	tempBranch := fmt.Sprintf("try-%d", time.Now().Unix())

	if err := createAndCheckoutBranch(tempBranch); err != nil {
		return fmt.Errorf("failed to create and checkout temporary branch: %w", err)
	}

	defer func() {
		if err := checkoutBranch(originalBranch); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to checkout original branch: %v\n", err)
		}
		if !keep {
			if err := deleteBranch(tempBranch); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete temporary branch: %v\n", err)
			}
		}
	}()

	output, exitStatus, err := executeCommand(args)
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	if err := commitChanges(strings.Join(args, " "), output, exitStatus); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	commitHash, err := getLastCommitHash()
	if err != nil {
		return fmt.Errorf("failed to get last commit hash: %w", err)
	}

	fmt.Println(output)
	fmt.Printf("\nCommand executed and results stored in commit %s\n", commitHash)
	if keep {
		fmt.Printf("Temporary branch '%s' has been kept\n", tempBranch)
	}

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
	if verbose {
		fmt.Printf("Creating and checking out branch: %s\n", branch)
	}
	return cmd.Run()
}

func checkoutBranch(branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	if verbose {
		fmt.Printf("Checking out branch: %s\n", branch)
	}
	return cmd.Run()
}

func deleteBranch(branch string) error {
	cmd := exec.Command("git", "branch", "-D", branch)
	if verbose {
		fmt.Printf("Deleting branch: %s\n", branch)
	}
	return cmd.Run()
}

func executeCommand(args []string) (string, int, error) {
	cmd := exec.Command(args[0], args[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitStatus := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitStatus = exitError.ExitCode()
		} else {
			return "", 0, err
		}
	}

	output := stdout.String() + stderr.String()
	return output, exitStatus, nil
}

func commitChanges(command, output string, exitStatus int) error {
	if err := exec.Command("git", "add", ".").Run(); err != nil {
		return err
	}

	commitMsg := fmt.Sprintf("Try: %s", command)
	if err := exec.Command("git", "commit", "-m", commitMsg).Run(); err != nil {
		return err
	}

	note := fmt.Sprintf("Command: %s\nExit Status: %d\nOutput:\n%s", command, exitStatus, output)
	return exec.Command("git", "notes", "add", "-m", note).Run()
}

func getLastCommitHash() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
