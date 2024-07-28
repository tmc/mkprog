package main

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

//go:embed system-prompt.txt
var systemPrompt string

type Vulnerability struct {
	Module  string
	Version string
	Fixed   string
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "autofixvulns [flags] [directory]",
	Short: "Automatically detect and fix vulnerabilities in Go projects",
	Long:  `A tool to automatically detect vulnerabilities using govulncheck and fix them by updating dependencies and Go version.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  run,
}

var (
	verbose bool
	dryRun  bool
)

func init() {
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show changes without making them")
}

func run(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	if err := checkGovulncheck(); err != nil {
		return err
	}

	projects, err := findGoProjects(dir)
	if err != nil {
		return fmt.Errorf("error finding Go projects: %w", err)
	}

	for _, project := range projects {
		if err := processProject(project); err != nil {
			log.Printf("Error processing project %s: %v", project, err)
		}
	}

	return nil
}

func checkGovulncheck() error {
	_, err := exec.LookPath("govulncheck")
	if err != nil {
		return fmt.Errorf("govulncheck not found. Please install it with: go install golang.org/x/vuln/cmd/govulncheck@latest")
	}
	return nil
}

func findGoProjects(root string) ([]string, error) {
	var projects []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "vendor" || info.Name() == ".git" {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
				projects = append(projects, path)
			}
		}
		return nil
	})

	return projects, err
}

func processProject(dir string) error {
	log.Printf("Processing project: %s", dir)

	vulns, err := runGovulncheck(dir)
	if err != nil {
		return fmt.Errorf("error running govulncheck: %w", err)
	}

	if len(vulns) == 0 {
		log.Println("No vulnerabilities found.")
		return nil
	}

	log.Printf("Found %d vulnerabilities", len(vulns))

	if err := backupFiles(dir); err != nil {
		return fmt.Errorf("error creating backup: %w", err)
	}

	if err := updateDependencies(dir, vulns); err != nil {
		return fmt.Errorf("error updating dependencies: %w", err)
	}

	if err := updateGoVersion(dir, vulns); err != nil {
		return fmt.Errorf("error updating Go version: %w", err)
	}

	if err := runGoModTidy(dir); err != nil {
		return fmt.Errorf("error running go mod tidy: %w", err)
	}

	if err := generateReport(dir, vulns); err != nil {
		return fmt.Errorf("error generating report: %w", err)
	}

	return nil
}

func runGovulncheck(dir string) ([]Vulnerability, error) {
	cmd := exec.Command("govulncheck", "-json", "./...")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var vulns []Vulnerability
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		var result map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &result); err != nil {
			continue
		}

		if vuln, ok := result["vulnerability"].(map[string]interface{}); ok {
			module := vuln["module"].(string)
			version := vuln["version"].(string)
			fixed := vuln["fixed"].(string)
			vulns = append(vulns, Vulnerability{Module: module, Version: version, Fixed: fixed})
		}
	}

	return vulns, nil
}

func backupFiles(dir string) error {
	files := []string{"go.mod", "go.sum"}
	for _, file := range files {
		src := filepath.Join(dir, file)
		dst := filepath.Join(dir, file+".bak")
		if err := copyFile(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func updateDependencies(dir string, vulns []Vulnerability) error {
	for _, vuln := range vulns {
		if vuln.Module == "stdlib" {
			continue
		}
		if dryRun {
			log.Printf("Would update %s to %s", vuln.Module, vuln.Fixed)
		} else {
			cmd := exec.Command("go", "get", vuln.Module+"@"+vuln.Fixed)
			cmd.Dir = dir
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("error updating %s: %w", vuln.Module, err)
			}
			log.Printf("Updated %s to %s", vuln.Module, vuln.Fixed)
		}
	}
	return nil
}

func updateGoVersion(dir string, vulns []Vulnerability) error {
	needsUpdate := false
	for _, vuln := range vulns {
		if vuln.Module == "stdlib" {
			needsUpdate = true
			break
		}
	}

	if !needsUpdate {
		return nil
	}

	goModPath := filepath.Join(dir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return err
	}

	re := regexp.MustCompile(`go (\d+\.\d+)`)
	matches := re.FindStringSubmatch(string(content))
	if len(matches) < 2 {
		return fmt.Errorf("could not find Go version in go.mod")
	}

	currentVersion := matches[1]
	latestVersion, err := getLatestGoVersion()
	if err != nil {
		return err
	}

	if currentVersion != latestVersion {
		newContent := re.ReplaceAllString(string(content), fmt.Sprintf("go %s", latestVersion))
		if dryRun {
			log.Printf("Would update Go version from %s to %s", currentVersion, latestVersion)
		} else {
			if err := os.WriteFile(goModPath, []byte(newContent), 0644); err != nil {
				return err
			}
			log.Printf("Updated Go version from %s to %s", currentVersion, latestVersion)
		}
	}

	return nil
}

func getLatestGoVersion() (string, error) {
	cmd := exec.Command("go", "version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(`go version go(\d+\.\d+)`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) < 2 {
		return "", fmt.Errorf("could not parse Go version")
	}

	return matches[1], nil
}

func runGoModTidy(dir string) error {
	if dryRun {
		log.Println("Would run go mod tidy")
		return nil
	}

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	return cmd.Run()
}

func generateReport(dir string, vulns []Vulnerability) error {
	reportPath := filepath.Join(dir, "vulnerability_report.txt")
	file, err := os.Create(reportPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	fmt.Fprintf(writer, "Vulnerability Report for %s\n\n", dir)
	fmt.Fprintf(writer, "Total vulnerabilities found: %d\n\n", len(vulns))

	for _, vuln := range vulns {
		fmt.Fprintf(writer, "Module: %s\n", vuln.Module)
		fmt.Fprintf(writer, "Vulnerable version: %s\n", vuln.Version)
		fmt.Fprintf(writer, "Fixed version: %s\n", vuln.Fixed)
		fmt.Fprintf(writer, "Action taken: ")
		if vuln.Module == "stdlib" {
			fmt.Fprintf(writer, "Updated Go version\n")
		} else {
			fmt.Fprintf(writer, "Updated dependency\n")
		}
		fmt.Fprintf(writer, "\n")
	}

	log.Printf("Generated vulnerability report: %s", reportPath)
	return nil
}

func handleComplexVulnerability(vuln Vulnerability) error {
	client, err := openai.New()
	if err != nil {
		return fmt.Errorf("error creating OpenAI client: %w", err)
	}

	ctx := context.Background()
	prompt := fmt.Sprintf("How to fix vulnerability in %s version %s?", vuln.Module, vuln.Version)
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(500))
	if err != nil {
		return fmt.Errorf("error generating AI response: %w", err)
	}

	fmt.Printf("AI suggestion for fixing %s:\n%s\n", vuln.Module, resp.Choices[0].Content)
	return nil
}
