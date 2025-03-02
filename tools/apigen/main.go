package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/tmc/mkprog/tools/apigen/generator"
	"github.com/tmc/mkprog/tools/apigen/openapi"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Define command-line flags
	specFile := pflag.StringP("spec", "s", "", "Path to OpenAPI/Swagger specification file (required)")
	outputDir := pflag.StringP("output", "o", "", "Output directory for generated client package (required)")
	packageName := pflag.StringP("package", "p", "", "Package name for generated code (default: derived from API name)")
	prefix := pflag.StringP("prefix", "x", "", "Prefix for type names (default: none)")
	features := pflag.StringP("features", "f", "all", "Comma-separated list of features to include")
	timeout := pflag.StringP("timeout", "t", "10s", "Default client timeout")
	dryRun := pflag.BoolP("dry-run", "d", false, "Print generated code without writing to disk")
	verbose := pflag.BoolP("verbose", "v", false, "Enable verbose output")

	pflag.Parse()

	// Validate required flags
	if *specFile == "" {
		return fmt.Errorf("spec file is required")
	}

	if *outputDir == "" && !*dryRun {
		return fmt.Errorf("output directory is required unless dry-run is enabled")
	}

	// Parse timeout duration
	clientTimeout, err := time.ParseDuration(*timeout)
	if err != nil {
		return fmt.Errorf("invalid timeout value: %w", err)
	}

	// Parse features
	featuresList := parseFeatures(*features)

	// Validate spec file existence
	if _, err := os.Stat(*specFile); os.IsNotExist(err) {
		return fmt.Errorf("spec file does not exist: %s", *specFile)
	}

	// Create output directory if it doesn't exist
	if *outputDir != "" && !*dryRun {
		if err := os.MkdirAll(*outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	if *verbose {
		fmt.Printf("Parsing OpenAPI specification: %s\n", *specFile)
	}

	// Parse the OpenAPI specification
	spec, err := openapi.ParseSpecification(*specFile)
	if err != nil {
		return fmt.Errorf("failed to parse specification: %w", err)
	}

	// Determine package name if not provided
	if *packageName == "" {
		*packageName = derivePackageName(spec.Info.Title)
	}

	// Create generator options
	options := generator.Options{
		PackageName:    *packageName,
		Prefix:         *prefix,
		Features:       featuresList,
		ClientTimeout:  clientTimeout,
		DryRun:         *dryRun,
		Verbose:        *verbose,
	}

	if *verbose {
		fmt.Printf("Generating client with options: %+v\n", options)
	}

	// Generate the client code
	ctx := context.Background()
	gen := generator.New(options)
	files, err := gen.GenerateClient(ctx, spec)
	if err != nil {
		return fmt.Errorf("failed to generate client: %w", err)
	}

	// Write generated files
	if *dryRun {
		// Print files to stdout in dry-run mode
		for name, content := range files {
			fmt.Printf("=== %s ===\n", name)
			fmt.Println(content)
			fmt.Println()
		}
	} else {
		// Write files to disk
		for name, content := range files {
			fullPath := filepath.Join(*outputDir, name)
			
			// Create parent directories if needed
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", name, err)
			}
			
			if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write file %s: %w", name, err)
			}
			
			if *verbose {
				fmt.Printf("Created file: %s\n", fullPath)
			}
		}
		
		if *verbose {
			fmt.Printf("Successfully generated API client in %s\n", *outputDir)
		}
	}

	return nil
}

// parseFeatures parses the comma-separated features list
func parseFeatures(features string) []string {
	if features == "all" {
		return []string{
			"models",
			"client",
			"operations",
			"auth",
			"ratelimit",
			"retry",
			"logging",
			"tests",
			"examples",
		}
	}
	
	return strings.Split(features, ",")
}

// derivePackageName derives a package name from the API title
func derivePackageName(title string) string {
	// Convert to lowercase and remove special characters
	name := strings.ToLower(title)
	name = strings.ReplaceAll(name, " ", "")
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, "_", "")
	name = strings.ReplaceAll(name, ".", "")
	name = strings.ReplaceAll(name, "api", "")
	
	// Ensure it's a valid Go package name
	if name == "" || strings.HasPrefix(name, "0") || strings.HasPrefix(name, "1") || 
		strings.HasPrefix(name, "2") || strings.HasPrefix(name, "3") || 
		strings.HasPrefix(name, "4") || strings.HasPrefix(name, "5") ||
		strings.HasPrefix(name, "6") || strings.HasPrefix(name, "7") ||
		strings.HasPrefix(name, "8") || strings.HasPrefix(name, "9") {
		name = "client"
	}
	
	// Add "client" suffix if it doesn't already end with "client"
	if !strings.HasSuffix(name, "client") {
		name += "client"
	}
	
	return name
}