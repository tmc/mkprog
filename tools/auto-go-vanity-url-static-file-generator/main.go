package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	sourcePath   string
	outputPath   string
	baseURL      string
	overwrite    bool
	verbose      bool
	htmlTemplate *template.Template
)

const templateString = `<!DOCTYPE html>
<html>
<head>
    <meta name="go-import" content="{{ .BaseURL }} git {{ .RepoURL }}">
    <meta name="go-source" content="{{ .BaseURL }} {{ .RepoURL }} {{ .RepoURL }}/tree/main{/dir} {{ .RepoURL }}/blob/main{/dir}/{file}#L{line}">
    <meta http-equiv="refresh" content="0; url={{ .GoDocURL }}">
</head>
<body>
    Redirecting to <a href="{{ .GoDocURL }}">{{ .GoDocURL }}</a>...
</body>
</html>
`

func init() {
	rootCmd.Flags().StringVarP(&sourcePath, "source", "s", "", "Path to the Go source tree")
	rootCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output directory for generated files")
	rootCmd.Flags().StringVarP(&baseURL, "base-url", "b", "", "Base URL for vanity imports (e.g., 'example.com/repo')")
	rootCmd.Flags().BoolVarP(&overwrite, "overwrite", "w", false, "Overwrite existing files")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose mode")

	rootCmd.MarkFlagRequired("source")
	rootCmd.MarkFlagRequired("output")
	rootCmd.MarkFlagRequired("base-url")

	var err error
	htmlTemplate, err = template.New("vanity").Parse(templateString)
	if err != nil {
		fmt.Printf("Error parsing HTML template: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "auto-go-vanity-url-static-file-generator",
	Short: "Generate static files for Go vanity URLs",
	Long:  `A tool that generates static HTML files to support vanity URLs for Go submodules.`,
	Run:   run,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	if err := validateInputs(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	submodules, err := findSubmodules(sourcePath)
	if err != nil {
		fmt.Printf("Error finding submodules: %v\n", err)
		os.Exit(1)
	}

	if err := generateFiles(submodules); err != nil {
		fmt.Printf("Error generating files: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Static files generated successfully.")
}

func validateInputs() error {
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source path does not exist: %s", sourcePath)
	}

	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	if !strings.Contains(baseURL, "://") {
		baseURL = "https://" + baseURL
	}

	return nil
}

func findSubmodules(root string) ([]string, error) {
	var submodules []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "vendor" {
			return filepath.SkipDir
		}
		if d.Name() == "go.mod" {
			relPath, err := filepath.Rel(root, filepath.Dir(path))
			if err != nil {
				return err
			}
			if relPath != "." {
				submodules = append(submodules, relPath)
			}
		}
		return nil
	})
	return submodules, err
}

func generateFiles(submodules []string) error {
	var g errgroup.Group
	var mu sync.Mutex

	for _, submodule := range submodules {
		submodule := submodule // https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func() error {
			return generateFile(submodule, &mu)
		})
	}

	return g.Wait()
}

func generateFile(submodule string, mu *sync.Mutex) error {
	outputFile := filepath.Join(outputPath, submodule, "index.html")

	mu.Lock()
	if _, err := os.Stat(outputFile); err == nil && !overwrite {
		mu.Unlock()
		if verbose {
			fmt.Printf("Skipping existing file: %s\n", outputFile)
		}
		return nil
	}
	mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %v", submodule, err)
	}

	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create file for %s: %v", submodule, err)
	}
	defer f.Close()

	data := struct {
		BaseURL  string
		RepoURL  string
		GoDocURL string
	}{
		BaseURL:  fmt.Sprintf("%s/%s", baseURL, submodule),
		RepoURL:  strings.TrimSuffix(baseURL, "/"),
		GoDocURL: fmt.Sprintf("https://pkg.go.dev/%s/%s", baseURL, submodule),
	}

	if err := htmlTemplate.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute template for %s: %v", submodule, err)
	}

	if verbose {
		fmt.Printf("Generated file: %s\n", outputFile)
	}

	return nil
}
