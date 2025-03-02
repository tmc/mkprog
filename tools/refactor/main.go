package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"golang.org/x/tools/go/ast/astutil"
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
	// Define command line flags
	filePath := pflag.StringP("file", "f", "", "Specific file to analyze")
	interactive := pflag.BoolP("interactive", "i", false, "Interactive mode")
	dryRun := pflag.BoolP("dry-run", "d", false, "Print suggested changes without modifying files")
	verbose := pflag.BoolP("verbose", "v", false, "Enable verbose output")
	temperature := pflag.Float64P("temp", "t", 0.1, "Temperature for AI generation")
	
	pflag.Parse()
	args := pflag.Args()
	
	// Default to current directory if no path provided
	path := "."
	if len(args) > 0 {
		path = args[0]
	}
	
	// Initialize the language model
	llm, err := anthropic.New(
		anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize language model: %w", err)
	}

	// Collect files to analyze
	var files []string
	if *filePath != "" {
		// Single file mode
		files = append(files, *filePath)
	} else {
		// Directory mode
		err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to walk directory: %w", err)
		}
	}

	if len(files) == 0 {
		return fmt.Errorf("no Go files found to analyze")
	}

	if *verbose {
		fmt.Printf("Found %d file(s) to analyze\n", len(files))
	}

	// Process each file
	for _, file := range files {
		if *verbose {
			fmt.Printf("Analyzing %s...\n", file)
		}

		// Parse the file
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", file, err)
		}

		// Read file content
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		// Get package imports
		imports := extractImports(node)
		packageInfo := fmt.Sprintf("Package: %s\nImports: %s\n", node.Name.Name, strings.Join(imports, ", "))
		
		// Analyze file with AI
		analysis, err := analyzeWithAI(llm, file, string(content), packageInfo, *temperature)
		if err != nil {
			return fmt.Errorf("failed to analyze %s: %w", file, err)
		}

		if *dryRun || *interactive {
			fmt.Printf("\n--- Suggested refactoring for %s ---\n\n", file)
			fmt.Println(analysis)
			
			if *interactive {
				fmt.Print("Apply these changes? (y/n): ")
				var input string
				fmt.Scanln(&input)
				if strings.ToLower(input) != "y" {
					continue
				}
			} else {
				continue // Skip application for dry run
			}
		}

		// Apply refactoring suggestions (in non-dry-run mode)
		if !*dryRun {
			err = applyRefactoring(analysis, file)
			if err != nil {
				return fmt.Errorf("failed to apply refactoring to %s: %w", file, err)
			}
			
			if *verbose {
				fmt.Printf("Successfully refactored %s\n", file)
			}
		}
	}

	return nil
}

// extractImports gets all imports from the AST
func extractImports(node *ast.File) []string {
	var imports []string
	for _, imp := range node.Imports {
		imports = append(imports, imp.Path.Value)
	}
	return imports
}

// analyzeWithAI sends the code to the AI for analysis
func analyzeWithAI(llm llms.Model, fileName, content, packageInfo string, temperature float64) (string, error) {
	ctx := context.Background()
	
	prompt := fmt.Sprintf("Analyze the following Go code and suggest refactoring improvements:\n\nFile: %s\n%s\n\n```go\n%s\n```", 
		fileName, packageInfo, content)
	
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}
	
	resp, err := llm.GenerateContent(ctx, messages, 
		llms.WithTemperature(temperature),
		llms.WithMaxTokens(4000))
	if err != nil {
		return "", fmt.Errorf("AI analysis failed: %w", err)
	}
	
	return llms.MessageContentToString(resp), nil
}

// applyRefactoring analyzes the AI suggestions and applies them to the file
// In a real implementation, this would parse the AI suggestions and apply changes
// Here we're demonstrating a simplified approach
func applyRefactoring(analysis, filePath string) error {
	// This is a simplified implementation
	// A real implementation would:
	// 1. Parse the AI suggestions into structured changes
	// 2. Apply each change to the AST
	// 3. Format and write the result
	
	// For now, we'll just show a basic file update with a marker
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	
	// Apply basic formatting for demonstration
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}
	
	// Add a comment indicating refactoring
	node = astutil.AddComment(node, 0, "// Code refactored by tmc/mkprog refactor tool")
	
	// Format the AST
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return fmt.Errorf("failed to format file: %w", err)
	}
	
	// Write back to file
	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}