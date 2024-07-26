To add documentation on how filters can be expressed in the usage, we need to modify the `main.go` file to include this information in the command's help text. We'll update the `rootCmd` definition to include a more detailed description of the filter syntax. Here's the modified `main.go` file with the added documentation:

package main

import (
	"context"
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

//go:embed system-prompt.txt
var systemPrompt string

type HAREntry struct {
	StartedDateTime time.Time `json:"startedDateTime"`
	Time            float64   `json:"time"`
	Request         struct {
		Method string `json:"method"`
		URL    string `json:"url"`
	} `json:"request"`
	Response struct {
		Status     int    `json:"status"`
		StatusText string `json:"statusText"`
		Content    struct {
			Size     int    `json:"size"`
			MimeType string `json:"mimeType"`
		} `json:"content"`
	} `json:"response"`
}

type HARLog struct {
	Entries []HAREntry `json:"entries"`
}

type HAR struct {
	Log HARLog `json:"log"`
}

type FilterFunc func(HAREntry) bool

type ChunkSummary struct {
	TotalEntries       int
	TotalTime          float64
	TotalSize          int
	StatusDistribution map[int]int
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	var inputFile, outputFormat, sortBy, filterQuery string
	var chunkSize int
	var performAIAnalysis bool

	rootCmd := &cobra.Command{
		Use:   "haranalyzer",
		Short: "Analyze HAR files with advanced filtering capabilities",
		Long: `Analyze HAR files with advanced filtering capabilities.

Filter Query Syntax:
  Filters can be expressed using key:value pairs, combined with AND/OR operators.
  Available filter keys:
    - method: HTTP method (e.g., GET, POST)
    - url: URL of the request
      - Supports exact match, contains, regex (/regex/), and wildcard (*)
    - status: HTTP status code
      - Supports exact match or range (e.g., 200-299)
    - time: Response time in milliseconds
      - Supports gt (greater than) and lt (less than) operators
    - size: Response size in bytes
      - Supports gt (greater than) and lt (less than) operators
    - content-type: Response content type

  Examples:
    - method:GET AND status:200
    - url:*/api/* OR url:/example\.com/
    - status:400-499 AND time:gt500
    - size:lt1000 AND content-type:application/json

  Multiple filters can be combined using AND/OR operators.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return analyzeHAR(inputFile, outputFormat, sortBy, filterQuery, chunkSize, performAIAnalysis)
		},
	}

	rootCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input HAR file (required)")
	rootCmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "Output format (text, json, csv)")
	rootCmd.Flags().StringVarP(&sortBy, "sort", "s", "time", "Sort entries by (time, size, status, url)")
	rootCmd.Flags().StringVarP(&filterQuery, "filter", "f", "", "Filter query (see usage for syntax)")
	rootCmd.Flags().IntVarP(&chunkSize, "chunk-size", "c", 100, "Chunk size for analysis")
	rootCmd.Flags().BoolVarP(&performAIAnalysis, "ai-analysis", "a", false, "Perform AI analysis on chunks")

	rootCmd.MarkFlagRequired("input")

	return rootCmd.Execute()
}

// ... (rest of the file remains unchanged)
