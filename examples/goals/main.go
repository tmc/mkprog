package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	goalsFile, err := findGoalsFile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	file, err := os.Open(goalsFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening .goals file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			fmt.Printf("%s: %s\n", strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading .goals file: %v\n", err)
		os.Exit(1)
	}
}

func findGoalsFile() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goalsFile := filepath.Join(dir, ".goals")
		if _, err := os.Stat(goalsFile); err == nil {
			return goalsFile, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf(".goals file not found in any parent directory")
		}
		dir = parent
	}
}