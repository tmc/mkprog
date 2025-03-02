package review

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/fatih/color"
)

// Issue represents a code review issue
type Issue struct {
	// File is the path to the file containing the issue
	File string `json:"file"`

	// Line is the line number where the issue occurs
	Line int `json:"line"`

	// Severity is one of "info", "warning", "error"
	Severity string `json:"severity"`

	// Message describes the issue
	Message string `json:"message"`

	// Rule is the name of the rule that detected the issue
	Rule string `json:"rule"`

	// Snippet is an optional code snippet showing the issue
	Snippet string `json:"snippet,omitempty"`
}

// Results contains the results of a code review
type Results struct {
	// Issues is a list of issues found during the review
	Issues []*Issue `json:"issues"`
}

// ToJSON outputs the results in JSON format
func (r *Results) ToJSON(w io.Writer) error {
	return json.NewEncoder(w).Encode(r)
}

// ToText outputs the results in plain text format
func (r *Results) ToText(w io.Writer) error {
	// Sort issues by file and line
	sort.Slice(r.Issues, func(i, j int) bool {
		if r.Issues[i].File == r.Issues[j].File {
			return r.Issues[i].Line < r.Issues[j].Line
		}
		return r.Issues[i].File < r.Issues[j].File
	})

	// Set up colors
	errorColor := color.New(color.FgRed, color.Bold).SprintFunc()
	warningColor := color.New(color.FgYellow, color.Bold).SprintFunc()
	infoColor := color.New(color.FgBlue, color.Bold).SprintFunc()
	fileColor := color.New(color.FgCyan).SprintFunc()
	lineColor := color.New(color.FgGreen).SprintFunc()

	// Count severity levels
	errorCount := 0
	warningCount := 0
	infoCount := 0

	// Group by file
	fileIssues := make(map[string][]*Issue)
	for _, issue := range r.Issues {
		fileIssues[issue.File] = append(fileIssues[issue.File], issue)

		switch issue.Severity {
		case "error":
			errorCount++
		case "warning":
			warningCount++
		case "info":
			infoCount++
		}
	}

	// Print summary
	fmt.Fprintf(w, "Code Review Results: %s, %s, %s\n\n",
		errorColor(fmt.Sprintf("%d errors", errorCount)),
		warningColor(fmt.Sprintf("%d warnings", warningCount)),
		infoColor(fmt.Sprintf("%d info", infoCount)))

	// Print issues by file
	for file, issues := range fileIssues {
		fmt.Fprintf(w, "%s\n", fileColor(file))
		for _, issue := range issues {
			// Format severity
			var severityText string
			switch issue.Severity {
			case "error":
				severityText = errorColor("ERROR")
			case "warning":
				severityText = warningColor("WARNING")
			case "info":
				severityText = infoColor("INFO")
			default:
				severityText = issue.Severity
			}

			// Print issue
			fmt.Fprintf(w, "  %s:%s [%s] %s (%s)\n",
				lineColor(fmt.Sprintf("%d", issue.Line)),
				severityText,
				issue.Rule,
				issue.Message,
				file)

			// Print snippet if available
			if issue.Snippet != "" {
				lines := strings.Split(issue.Snippet, "\n")
				for _, line := range lines {
					fmt.Fprintf(w, "    | %s\n", line)
				}
				fmt.Fprintln(w)
			}
		}
		fmt.Fprintln(w)
	}

	return nil
}

// ToMarkdown outputs the results in markdown format
func (r *Results) ToMarkdown(w io.Writer) error {
	// Sort issues by file and line
	sort.Slice(r.Issues, func(i, j int) bool {
		if r.Issues[i].File == r.Issues[j].File {
			return r.Issues[i].Line < r.Issues[j].Line
		}
		return r.Issues[i].File < r.Issues[j].File
	})

	// Count severity levels
	errorCount := 0
	warningCount := 0
	infoCount := 0
	for _, issue := range r.Issues {
		switch issue.Severity {
		case "error":
			errorCount++
		case "warning":
			warningCount++
		case "info":
			infoCount++
		}
	}

	// Print header
	fmt.Fprintf(w, "# Code Review Results\n\n")
	fmt.Fprintf(w, "## Summary\n\n")
	fmt.Fprintf(w, "- **Errors:** %d\n", errorCount)
	fmt.Fprintf(w, "- **Warnings:** %d\n", warningCount)
	fmt.Fprintf(w, "- **Info:** %d\n\n", infoCount)

	// Group by file
	fileIssues := make(map[string][]*Issue)
	for _, issue := range r.Issues {
		fileIssues[issue.File] = append(fileIssues[issue.File], issue)
	}

	// Table of contents
	fmt.Fprintf(w, "## Files\n\n")
	for file := range fileIssues {
		fileID := strings.ReplaceAll(file, "/", "-")
		fileID = strings.ReplaceAll(fileID, ".", "-")
		fmt.Fprintf(w, "- [%s](#%s)\n", file, fileID)
	}
	fmt.Fprintln(w)

	// Print issues by file
	for file, issues := range fileIssues {
		fileID := strings.ReplaceAll(file, "/", "-")
		fileID = strings.ReplaceAll(fileID, ".", "-")
		fmt.Fprintf(w, "## %s {#%s}\n\n", file, fileID)

		fmt.Fprintf(w, "| Line | Severity | Rule | Message |\n")
		fmt.Fprintf(w, "|------|----------|------|--------|\n")

		for _, issue := range issues {
			// Format severity based on level
			var severityText string
			switch issue.Severity {
			case "error":
				severityText = "🔴 ERROR"
			case "warning":
				severityText = "🟡 WARNING"
			case "info":
				severityText = "🔵 INFO"
			default:
				severityText = issue.Severity
			}

			// Escape pipe characters in message
			message := strings.ReplaceAll(issue.Message, "|", "\\|")

			fmt.Fprintf(w, "| %d | %s | %s | %s |\n",
				issue.Line, severityText, issue.Rule, message)
		}

		fmt.Fprintln(w)
	}

	return nil
}

// FilterBySeverity returns a new Results with only issues of the given severity or higher
func (r *Results) FilterBySeverity(severity string) *Results {
	filtered := &Results{
		Issues: make([]*Issue, 0),
	}

	for _, issue := range r.Issues {
		if isAtLeastSeverity(issue.Severity, severity) {
			filtered.Issues = append(filtered.Issues, issue)
		}
	}

	return filtered
}