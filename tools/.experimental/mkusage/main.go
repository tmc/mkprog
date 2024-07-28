package main

import (
	"context"
	_ "embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

//go:embed system-prompt.txt
var systemPrompt string

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: %s <path_to_go_program>", os.Args[0])
	}

	programPath := os.Args[1]
	programInfo, err := extractProgramInfo(programPath)
	if err != nil {
		return fmt.Errorf("failed to extract program info: %w", err)
	}

	usageContent, err := generateUsageContent(programInfo)
	if err != nil {
		return fmt.Errorf("failed to generate usage content: %w", err)
	}

	fmt.Println(usageContent)
	return nil
}

type ProgramInfo struct {
	Name        string
	Description string
	Package     string
}

func extractProgramInfo(path string) (ProgramInfo, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, path, nil, parser.ParseComments)
	if err != nil {
		return ProgramInfo{}, err
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			if hasMainFunction(file) {
				return extractInfoFromFile(file, filepath.Base(path))
			}
		}
	}

	return ProgramInfo{}, fmt.Errorf("no main package or function found")
}

func hasMainFunction(file *ast.File) bool {
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "main" {
			return true
		}
	}
	return false
}

func extractInfoFromFile(file *ast.File, defaultName string) (ProgramInfo, error) {
	info := ProgramInfo{
		Name:    defaultName,
		Package: file.Name.Name,
	}

	if file.Doc != nil {
		info.Description = file.Doc.Text()
	}

	return info, nil
}

func generateUsageContent(info ProgramInfo) (string, error) {
	ctx := context.Background()
	client, err := anthropic.New()
	if err != nil {
		return "", fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	prompt := fmt.Sprintf("Generate a USAGE file content for a Go program with the following information:\n"+
		"Name: %s\n"+
		"Package: %s\n"+
		"Description: %s\n\n"+
		"Please format the USAGE content in a clear and concise manner, excluding any flag details. "+
		"Focus on providing a brief overview of the program, its purpose, and basic usage instructions.",
		info.Name, info.Package, info.Description)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(1000))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	return strings.TrimSpace(resp.Choices[0].Content), nil
}
