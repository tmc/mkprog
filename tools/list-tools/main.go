package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/pflag"
)

// Tool represents a tool in the mkprog toolset
type Tool struct {
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Path          string    `json:"path"`
	Category      string    `json:"category"`
	FeaturesCount int       `json:"featuresCount"`
	LastModified  time.Time `json:"lastModified"`
}

// Config represents the configuration for the list-tools utility
type Config struct {
	ToolsDir     string   `json:"toolsDir"`
	CategoryTags map[string][]string `json:"categoryTags"`
}

// Define the known categories
var knownCategories = map[string]string{
	"code-generation": "Code Generation",
	"code-improvement": "Code Improvement",
	"planning-analysis": "Planning & Analysis",
	"git-integration": "Git Integration",
	"meta": "Meta Tools",
}

// Base directory for our tools
var rootDir = filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(os.Args[0]))), "tools")

// Cache file settings
var (
	cacheFile     = filepath.Join(os.TempDir(), "mkprog-tools-cache.json")
	cacheDuration = 1 * time.Hour
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Command-line flags
	category := pflag.StringP("category", "c", "", "Filter tools by category")
	format := pflag.StringP("format", "f", "text", "Output format (text, json, markdown)")
	search := pflag.StringP("search", "s", "", "Search term for filtering tools")
	refresh := pflag.BoolP("refresh", "r", false, "Refresh cache and get fresh tool data")
	verbose := pflag.BoolP("verbose", "v", false, "Show verbose output")
	pflag.Parse()

	// Determine the tools directory
	toolsDir := rootDir
	if path := pflag.Arg(0); path != "" {
		toolsDir = path
	}

	// Try to load from cache first, unless refresh is specified
	var tools []Tool
	var err error
	if !*refresh {
		tools, err = loadCache()
		if err == nil && *verbose {
			fmt.Println("Using cached tool list")
		}
	}

	// Scan the tools directory if cache failed or refresh is requested
	if tools == nil || len(tools) == 0 || *refresh {
		if *verbose {
			fmt.Println("Scanning tools directory:", toolsDir)
		}
		tools, err = scanTools(toolsDir)
		if err != nil {
			return fmt.Errorf("failed to scan tools: %w", err)
		}

		// Save to cache for future use
		if err := saveCache(tools); err != nil && *verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to save cache: %v\n", err)
		}
	}

	// Apply category filter if specified
	if *category != "" {
		tools = filterByCategory(tools, *category)
	}

	// Apply search filter if specified
	if *search != "" {
		tools = filterBySearch(tools, *search)
	}

	// Sort tools by category and then by name
	sort.Slice(tools, func(i, j int) bool {
		if tools[i].Category != tools[j].Category {
			return tools[i].Category < tools[j].Category
		}
		return tools[i].Name < tools[j].Name
	})

	// Output the tools list
	switch *format {
	case "json":
		return outputJSON(tools)
	case "markdown":
		return outputMarkdown(tools)
	default:
		return outputText(tools, *verbose)
	}
}

// scanTools scans the tools directory and returns a list of tools
func scanTools(toolsDir string) ([]Tool, error) {
	var tools []Tool

	entries, err := os.ReadDir(toolsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		toolName := entry.Name()
		toolPath := filepath.Join(toolsDir, toolName)

		// Skip hidden directories and non-tool directories
		if strings.HasPrefix(toolName, ".") || toolName == "vendor" {
			continue
		}

		// Check if this is a proper tool directory
		readmePath := filepath.Join(toolPath, "README.md")
		if _, err := os.Stat(readmePath); os.IsNotExist(err) {
			continue
		}

		// Parse the README for description and features
		description, features := parseReadme(readmePath)
		if description == "" {
			description = toolName
		}

		// Get the category from the tool name or directory structure
		category := categorizeTools(toolName, description)

		// Get the last modified time
		info, err := os.Stat(toolPath)
		var lastModified time.Time
		if err == nil {
			lastModified = info.ModTime()
		}

		tools = append(tools, Tool{
			Name:          toolName,
			Description:   description,
			Path:          toolPath,
			Category:      category,
			FeaturesCount: len(features),
			LastModified:  lastModified,
		})
	}

	return tools, nil
}

// parseReadme parses a README.md file to extract description and features
func parseReadme(path string) (string, []string) {
	file, err := os.Open(path)
	if err != nil {
		return "", nil
	}
	defer file.Close()

	var description string
	var features []string
	inFeatures := false

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Grab the first paragraph after the title as the description
		if description == "" && !strings.HasPrefix(line, "#") && strings.TrimSpace(line) != "" {
			description = strings.TrimSpace(line)
			continue
		}

		// Look for the features section
		if strings.HasPrefix(line, "## Features") || strings.HasPrefix(line, "## Capabilities") {
			inFeatures = true
			continue
		} else if inFeatures && strings.HasPrefix(line, "##") {
			inFeatures = false
			continue
		}

		// Collect feature bullet points
		if inFeatures && strings.HasPrefix(strings.TrimSpace(line), "-") {
			feature := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
			if feature != "" {
				features = append(features, feature)
			}
		}
	}

	return description, features
}

// categorizeTools determines the category of a tool based on its name and description
func categorizeTools(name, description string) string {
	name = strings.ToLower(name)
	description = strings.ToLower(description)

	// Define category detection patterns
	patterns := map[string][]string{
		"Code Generation": {
			"mkprog", "generate", "generat", "scaffold", "create", "build", "make",
		},
		"Code Improvement": {
			"refactor", "fix", "improve", "mod", "modify", "change", "transform", "update",
		},
		"Planning & Analysis": {
			"plan", "analyze", "analyse", "ask", "visual", "stat", "bench", "profile", "token", "dep",
		},
		"Git Integration": {
			"git", "commit", "pull", "push", "merge", "branch", "version",
		},
		"Meta Tools": {
			"list", "meta", "help", "info", "try", "utility",
		},
	}

	// Try to match the tool name and description to a category
	for category, keywords := range patterns {
		for _, keyword := range keywords {
			if strings.Contains(name, keyword) || strings.Contains(description, keyword) {
				return category
			}
		}
	}

	return "Uncategorized"
}

// loadCache loads the tool list from cache
func loadCache() ([]Tool, error) {
	file, err := os.Open(cacheFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Check if cache is still valid
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if time.Since(info.ModTime()) > cacheDuration {
		return nil, fmt.Errorf("cache expired")
	}

	// Read and parse the cache
	var tools []Tool
	if err := json.NewDecoder(file).Decode(&tools); err != nil {
		return nil, err
	}

	return tools, nil
}

// saveCache saves the tool list to cache
func saveCache(tools []Tool) error {
	file, err := os.Create(cacheFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(tools)
}

// filterByCategory filters tools by category
func filterByCategory(tools []Tool, category string) []Tool {
	category = strings.ToLower(category)
	var filtered []Tool

	// Get the proper cased category if it's a known alias
	for k, v := range knownCategories {
		if strings.HasPrefix(strings.ToLower(k), category) || strings.HasPrefix(strings.ToLower(v), category) {
			category = v
			break
		}
	}

	for _, tool := range tools {
		if strings.HasPrefix(strings.ToLower(tool.Category), category) {
			filtered = append(filtered, tool)
		}
	}

	return filtered
}

// filterBySearch filters tools by search term
func filterBySearch(tools []Tool, term string) []Tool {
	term = strings.ToLower(term)
	var filtered []Tool

	for _, tool := range tools {
		if strings.Contains(strings.ToLower(tool.Name), term) ||
			strings.Contains(strings.ToLower(tool.Description), term) ||
			strings.Contains(strings.ToLower(tool.Category), term) {
			filtered = append(filtered, tool)
		}
	}

	return filtered
}

// outputText outputs the tool list in text format
func outputText(tools []Tool, verbose bool) error {
	if len(tools) == 0 {
		fmt.Println("No tools found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Group tools by category
	categories := make(map[string][]Tool)
	for _, tool := range tools {
		categories[tool.Category] = append(categories[tool.Category], tool)
	}

	// Sort categories
	categoryNames := make([]string, 0, len(categories))
	for category := range categories {
		categoryNames = append(categoryNames, category)
	}
	sort.Strings(categoryNames)

	// Print the tools by category
	fmt.Fprintln(w, "CATEGORY\tTOOL\tDESCRIPTION")
	fmt.Fprintln(w, "--------\t----\t-----------")

	for _, category := range categoryNames {
		categoryTools := categories[category]
		// Sort tools by name
		sort.Slice(categoryTools, func(i, j int) bool {
			return categoryTools[i].Name < categoryTools[j].Name
		})

		for _, tool := range categoryTools {
			fmt.Fprintf(w, "%s\t%s\t%s\n", category, tool.Name, tool.Description)
		}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "\nTotal: %d tools in %d categories\n", len(tools), len(categories))
	}

	return nil
}

// outputJSON outputs the tool list in JSON format
func outputJSON(tools []Tool) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(tools)
}

// outputMarkdown outputs the tool list in Markdown format
func outputMarkdown(tools []Tool) error {
	if len(tools) == 0 {
		fmt.Println("No tools found")
		return nil
	}

	fmt.Println("# mkprog Tools")
	fmt.Println()

	// Group tools by category
	categories := make(map[string][]Tool)
	for _, tool := range tools {
		categories[tool.Category] = append(categories[tool.Category], tool)
	}

	// Sort categories
	categoryNames := make([]string, 0, len(categories))
	for category := range categories {
		categoryNames = append(categoryNames, category)
	}
	sort.Strings(categoryNames)

	// Print the tools by category
	for _, category := range categoryNames {
		fmt.Printf("## %s\n\n", category)
		
		categoryTools := categories[category]
		// Sort tools by name
		sort.Slice(categoryTools, func(i, j int) bool {
			return categoryTools[i].Name < categoryTools[j].Name
		})

		for _, tool := range categoryTools {
			fmt.Printf("### %s\n\n", tool.Name)
			fmt.Printf("%s\n\n", tool.Description)
			fmt.Printf("**Path:** `%s`\n\n", tool.Path)
		}
	}

	return nil
}
