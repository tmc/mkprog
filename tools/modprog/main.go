package main

import (
	"bufio"
	"context"
	_ "embed"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

//go:embed system-prompt.txt
var systemPrompt string

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	cmdDir := flag.String("dir", ".", "Path to the Go command directory")
	modificationDesc := flag.String("mod", "", "Modification description")
	flag.Parse()

	if *cmdDir == "" || *modificationDesc == "" {
		flag.Usage()
		return fmt.Errorf("both -dir and -mod flags are required")
	}

	goFiles, err := findGoFiles(*cmdDir)
	if err != nil {
		return fmt.Errorf("error finding Go files: %w", err)
	}

	if len(goFiles) == 0 {
		return fmt.Errorf("no Go files found in the specified directory")
	}

	existingCode, err := combineGoFiles(goFiles)
	if err != nil {
		return fmt.Errorf("error reading Go files: %w", err)
	}

	ctx := context.Background()
	client, err := anthropic.New()
	if err != nil {
		return fmt.Errorf("error creating Anthropic client: %w", err)
	}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Existing code:\n\n%s\n\nModification description: %s", existingCode, *modificationDesc)),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(4000))
	if err != nil {
		return fmt.Errorf("error generating content: %w", err)
	}

	modifiedCode := resp.Choices[0].Content

	diff := generateDiff(existingCode, modifiedCode)
	fmt.Println("Proposed changes:")
	fmt.Println(diff)

	if !confirmChanges() {
		fmt.Println("Changes not applied.")
		return nil
	}

	backupDir := createBackup(*cmdDir)
	fmt.Printf("Original files backed up to: %s\n", backupDir)

	err = applyChanges(*cmdDir, modifiedCode)
	if err != nil {
		return fmt.Errorf("error applying changes: %w", err)
	}

	fmt.Println("Changes applied successfully.")
	return nil
}

func findGoFiles(dir string) ([]string, error) {
	var goFiles []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			goFiles = append(goFiles, path)
		}
		return nil
	})
	return goFiles, err
}

func combineGoFiles(files []string) (string, error) {
	var combined strings.Builder
	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			return "", err
		}
		combined.WriteString(fmt.Sprintf("// File: %s\n", file))
		combined.Write(content)
		combined.WriteString("\n\n")
	}
	return combined.String(), nil
}

func applyChanges(dir string, modifiedCode string) error {
	// This is a simplified implementation. In a real-world scenario,
	// you would need to parse the modified code and apply changes to individual files.
	mainFile := filepath.Join(dir, "main.go")
	return ioutil.WriteFile(mainFile, []byte(modifiedCode), 0644)
}

func generateDiff(oldCode, newCode string) string {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(oldCode),
		B:        difflib.SplitLines(newCode),
		FromFile: "Original",
		ToFile:   "Modified",
		Context:  3,
	}
	text, _ := difflib.GetUnifiedDiffString(diff)
	return text
}

func confirmChanges() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Do you want to apply these changes? (y/n): ")
	response, _ := reader.ReadString('\n')
	return strings.ToLower(strings.TrimSpace(response)) == "y"
}

func createBackup(dir string) string {
	backupDir := dir + "_backup"
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(backupDir, relPath)
		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}
		input, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		return ioutil.WriteFile(destPath, input, info.Mode())
	})
	if err != nil {
		log.Printf("Error creating backup: %v", err)
		return ""
	}
	return backupDir
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -dir <path_to_go_command_dir> -mod <modification_description>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
	}
}
