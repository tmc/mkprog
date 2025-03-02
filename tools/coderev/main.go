package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tmc/mkprog/tools/coderev/review"
)

var (
	diffFlag      string
	formatFlag    string
	configFlag    string
	verboseFlag   bool
	severityFlag  string
	outputFlag    string
	filterFlag    []string
	includeFlag   []string
	excludeFlag   []string
	ignoreFlag    []string
	maxErrorsFlag int
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "coderev [flags] [path...]",
		Short: "AI-assisted code review tool for Go",
		Long: `coderev is a tool that helps developers perform automated code reviews with AI assistance.
It provides actionable feedback on code quality, potential issues, and improvement suggestions.`,
		RunE: runCodeReview,
	}

	rootCmd.Flags().StringVar(&diffFlag, "diff", "", "Review changes compared to specified branch")
	rootCmd.Flags().StringVar(&formatFlag, "format", "text", "Output format (text, json, markdown)")
	rootCmd.Flags().StringVar(&configFlag, "config", ".coderev.yaml", "Path to configuration file")
	rootCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().StringVar(&severityFlag, "severity", "warning", "Minimum severity level (info, warning, error)")
	rootCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Output file (default is stdout)")
	rootCmd.Flags().StringArrayVar(&filterFlag, "filter", nil, "Filter by rule categories")
	rootCmd.Flags().StringArrayVar(&includeFlag, "include", nil, "Include only specific directories")
	rootCmd.Flags().StringArrayVar(&excludeFlag, "exclude", nil, "Exclude specific directories")
	rootCmd.Flags().StringArrayVar(&ignoreFlag, "ignore", nil, "Ignore specific issues")
	rootCmd.Flags().IntVar(&maxErrorsFlag, "max-errors", 0, "Maximum number of errors to report (0 = no limit)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runCodeReview(cmd *cobra.Command, args []string) error {
	// No paths provided, use current directory
	if len(args) == 0 {
		args = []string{"./..."}
	}

	config, err := review.LoadConfig(configFlag)
	if err != nil && configFlag != ".coderev.yaml" {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override config with command line flags
	if verboseFlag {
		config.Verbose = true
	}
	if severityFlag != "warning" {
		config.SeverityThreshold = severityFlag
	}
	if len(filterFlag) > 0 {
		config.Filters = filterFlag
	}
	if len(includeFlag) > 0 {
		config.Include = includeFlag
	}
	if len(excludeFlag) > 0 {
		config.Exclude = excludeFlag
	}
	if len(ignoreFlag) > 0 {
		config.Ignore = ignoreFlag
	}
	if maxErrorsFlag > 0 {
		config.MaxErrors = maxErrorsFlag
	}

	// Set up the reviewer
	reviewer := review.NewReviewer(config)

	// Run the review
	var results *review.Results
	if diffFlag != "" {
		results, err = reviewer.ReviewDiff(diffFlag, args)
	} else {
		results, err = reviewer.ReviewPaths(args)
	}

	if err != nil {
		return fmt.Errorf("review failed: %w", err)
	}

	// Output the results
	output := os.Stdout
	if outputFlag != "" {
		output, err = os.Create(outputFlag)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer output.Close()
	}

	switch formatFlag {
	case "json":
		return results.ToJSON(output)
	case "markdown":
		return results.ToMarkdown(output)
	default:
		return results.ToText(output)
	}
}