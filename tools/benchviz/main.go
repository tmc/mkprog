package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/spf13/pflag"
	"golang.org/x/perf/benchstat"
	"gonum.org/v1/gonum/stat"
)

// BenchResult represents a single benchmark result
type BenchResult struct {
	Name       string  `json:"name"`
	Iterations int     `json:"iterations"`
	TimePerOp  float64 `json:"timePerOp"`     // ns/op
	BytesPerOp int64   `json:"bytesPerOp"`    // B/op
	AllocsPerOp int64  `json:"allocsPerOp"`   // allocs/op
	Metadata   map[string]string `json:"metadata"`
}

// BenchmarkRun represents a collection of benchmark results
type BenchmarkRun struct {
	Timestamp  time.Time              `json:"timestamp"`
	GOOS       string                 `json:"goos"`
	GOARCH     string                 `json:"goarch"`
	Package    string                 `json:"package"`
	CPU        string                 `json:"cpu"`
	Results    map[string]BenchResult `json:"results"` // keyed by benchmark name
	RawContent string                 `json:"rawContent"`
}

// ComparisonResult represents a comparison between two benchmark runs
type ComparisonResult struct {
	Name      string  `json:"name"`
	BaseTime  float64 `json:"baseTime"`
	NewTime   float64 `json:"newTime"`
	Delta     float64 `json:"delta"`
	DeltaPct  float64 `json:"deltaPct"`
	Speedup   float64 `json:"speedup"`
	BaseAlloc int64   `json:"baseAlloc"`
	NewAlloc  int64   `json:"newAlloc"`
	AllocDelta float64 `json:"allocDelta"`
	Significant bool  `json:"significant"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Command-line flags
	outputFile := pflag.StringP("output", "o", "benchmark.html", "Output file")
	format := pflag.StringP("format", "f", "html", "Output format (html, svg, json)")
	baselineFile := pflag.StringP("baseline", "b", "", "Baseline benchmark file for comparison")
	threshold := pflag.Float64P("threshold", "t", 20.0, "Highlight changes above this threshold (percent)")
	showProfiles := pflag.BoolP("profile", "p", false, "Include CPU/memory profile visualization")
	showStats := pflag.BoolP("stat", "s", false, "Show detailed statistical analysis")
	verbose := pflag.BoolP("verbose", "v", false, "Enable verbose output")

	pflag.Parse()
	args := pflag.Args()

	if len(args) == 0 {
		return fmt.Errorf("no benchmark files specified")
	}

	// Validate format
	switch *format {
	case "html", "svg", "json":
		// Valid formats
	default:
		return fmt.Errorf("invalid format: %s (must be html, svg, or json)", *format)
	}

	// Parse benchmark files
	benchmarks := make([]*BenchmarkRun, 0, len(args))
	
	for _, file := range args {
		if *verbose {
			fmt.Printf("Parsing benchmark file: %s\n", file)
		}
		
		benchmark, err := parseBenchmarkFile(file)
		if err != nil {
			return fmt.Errorf("failed to parse benchmark file %s: %w", file, err)
		}
		
		benchmarks = append(benchmarks, benchmark)
	}

	if len(benchmarks) == 0 {
		return fmt.Errorf("no valid benchmark data found")
	}

	// Handle baseline comparison if specified
	var baseline *BenchmarkRun
	if *baselineFile != "" {
		var err error
		baseline, err = parseBenchmarkFile(*baselineFile)
		if err != nil {
			return fmt.Errorf("failed to parse baseline benchmark file %s: %w", *baselineFile, err)
		}
		
		if *verbose {
			fmt.Printf("Using baseline: %s\n", *baselineFile)
		}
	}

	// Generate output based on format
	switch *format {
	case "html":
		return generateHTMLOutput(benchmarks, baseline, *outputFile, *threshold, *showProfiles, *showStats)
	case "json":
		return generateJSONOutput(benchmarks, baseline, *outputFile, *threshold)
	case "svg":
		return generateSVGOutput(benchmarks, baseline, *outputFile, *threshold)
	}

	return nil
}

// parseBenchmarkFile parses a Go benchmark output file
func parseBenchmarkFile(path string) (*BenchmarkRun, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	run := &BenchmarkRun{
		Timestamp: time.Now(),
		Results:   make(map[string]BenchResult),
		RawContent: string(content),
	}

	// Regular expressions for parsing benchmark output
	reGOOS := regexp.MustCompile(`^goos: (.+)$`)
	reGOARCH := regexp.MustCompile(`^goarch: (.+)$`)
	rePkg := regexp.MustCompile(`^pkg: (.+)$`)
	reCPU := regexp.MustCompile(`^cpu: (.+)$`)
	reBench := regexp.MustCompile(`^(Benchmark[^\s]+)\s+(\d+)\s+(\d+(?:\.\d+)?) ns/op(?:\s+(\d+) B/op)?(?:\s+(\d+) allocs/op)?`)

	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		
		// Parse metadata
		if matches := reGOOS.FindStringSubmatch(line); matches != nil {
			run.GOOS = matches[1]
			continue
		}
		
		if matches := reGOARCH.FindStringSubmatch(line); matches != nil {
			run.GOARCH = matches[1]
			continue
		}
		
		if matches := rePkg.FindStringSubmatch(line); matches != nil {
			run.Package = matches[1]
			continue
		}
		
		if matches := reCPU.FindStringSubmatch(line); matches != nil {
			run.CPU = matches[1]
			continue
		}
		
		// Parse benchmark results
		if matches := reBench.FindStringSubmatch(line); matches != nil {
			name := matches[1]
			iterations, _ := strconv.Atoi(matches[2])
			timePerOp, _ := strconv.ParseFloat(matches[3], 64)
			
			result := BenchResult{
				Name:       name,
				Iterations: iterations,
				TimePerOp:  timePerOp,
				Metadata:   make(map[string]string),
			}
			
			// Parse optional B/op
			if len(matches) > 4 && matches[4] != "" {
				bytesPerOp, _ := strconv.ParseInt(matches[4], 10, 64)
				result.BytesPerOp = bytesPerOp
			}
			
			// Parse optional allocs/op
			if len(matches) > 5 && matches[5] != "" {
				allocsPerOp, _ := strconv.ParseInt(matches[5], 10, 64)
				result.AllocsPerOp = allocsPerOp
			}
			
			run.Results[name] = result
		}
	}

	if len(run.Results) == 0 {
		return nil, fmt.Errorf("no benchmark results found in file")
	}

	return run, nil
}

// compareBenchmarks compares two benchmark runs and generates comparison results
func compareBenchmarks(baseline, current *BenchmarkRun, threshold float64) []ComparisonResult {
	var results []ComparisonResult
	
	// Get all unique benchmark names
	benchmarkNames := make(map[string]bool)
	for name := range baseline.Results {
		benchmarkNames[name] = true
	}
	for name := range current.Results {
		benchmarkNames[name] = true
	}
	
	// Sort benchmark names for consistent output
	names := make([]string, 0, len(benchmarkNames))
	for name := range benchmarkNames {
		names = append(names, name)
	}
	sort.Strings(names)
	
	// Compare each benchmark
	for _, name := range names {
		baseResult, baseExists := baseline.Results[name]
		newResult, newExists := current.Results[name]
		
		// Skip if either benchmark doesn't exist
		if !baseExists || !newExists {
			continue
		}
		
		delta := newResult.TimePerOp - baseResult.TimePerOp
		deltaPct := (delta / baseResult.TimePerOp) * 100
		speedup := baseResult.TimePerOp / newResult.TimePerOp
		
		allocDelta := 0.0
		if baseResult.BytesPerOp > 0 {
			allocDelta = float64(newResult.BytesPerOp - baseResult.BytesPerOp) / float64(baseResult.BytesPerOp) * 100
		}
		
		// Determine if the change is significant (exceeds threshold)
		significant := math.Abs(deltaPct) >= threshold
		
		results = append(results, ComparisonResult{
			Name:        name,
			BaseTime:    baseResult.TimePerOp,
			NewTime:     newResult.TimePerOp,
			Delta:       delta,
			DeltaPct:    deltaPct,
			Speedup:     speedup,
			BaseAlloc:   baseResult.BytesPerOp,
			NewAlloc:    newResult.BytesPerOp,
			AllocDelta:  allocDelta,
			Significant: significant,
		})
	}
	
	return results
}

// calculateStatistics calculates statistical metrics for benchmark results
func calculateStatistics(benchmarks []*BenchmarkRun) map[string]map[string]float64 {
	stats := make(map[string]map[string]float64)
	
	// Get all unique benchmark names
	benchmarkNames := make(map[string]bool)
	for _, run := range benchmarks {
		for name := range run.Results {
			benchmarkNames[name] = true
		}
	}
	
	// Calculate statistics for each benchmark
	for name := range benchmarkNames {
		// Collect times for this benchmark across all runs
		var times []float64
		for _, run := range benchmarks {
			if result, exists := run.Results[name]; exists {
				times = append(times, result.TimePerOp)
			}
		}
		
		// Skip if we don't have enough data
		if len(times) < 2 {
			continue
		}
		
		// Calculate statistics
		mean := stat.Mean(times, nil)
		stdDev := stat.StdDev(times, nil)
		cv := (stdDev / mean) * 100 // Coefficient of variation as percentage
		
		// Store statistics
		stats[name] = map[string]float64{
			"mean":   mean,
			"stddev": stdDev,
			"cv":     cv,
			"min":    sliceMin(times),
			"max":    sliceMax(times),
		}
	}
	
	return stats
}

// sliceMin returns the minimum value in a slice of float64
func sliceMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	
	return min
}

// sliceMax returns the maximum value in a slice of float64
func sliceMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	
	return max
}

// generateHTMLOutput generates an HTML visualization of benchmark results
func generateHTMLOutput(benchmarks []*BenchmarkRun, baseline *BenchmarkRun, outputFile string, threshold float64, showProfiles, showStats bool) error {
	// Create a new page
	page := components.NewPage()
	page.PageTitle = "Go Benchmark Visualization"
	page.Layout = components.PageFlexLayout
	page.Padding = 10
	
	// Configuration for charts
	toolbox := []opts.ToolBoxFeatureOpts{
		{
			Show:  true,
			Type:  "saveAsImage",
			Title: "Save as Image",
		},
		{
			Show:  true,
			Type:  "dataView",
			Title: "Data View",
		},
		{
			Show:  true,
			Type:  "dataZoom",
			Title: "Data Zoom",
		},
		{
			Show:  true,
			Type:  "restore",
			Title: "Restore",
		},
	}
	
	// Add header info
	header := components.NewDivWithConf(&components.DivConfiguration{
		Padding: 10,
		Width:   "100%",
		Height:  "auto",
	})
	benchmark := benchmarks[len(benchmarks)-1] // Use the most recent benchmark for header info
	headerContent := fmt.Sprintf(`
    <h1>Go Benchmark Visualization</h1>
    <div style="display: flex; flex-wrap: wrap;">
        <div style="margin-right: 30px;">
            <p><strong>Package:</strong> %s</p>
            <p><strong>GOOS:</strong> %s</p>
        </div>
        <div style="margin-right: 30px;">
            <p><strong>GOARCH:</strong> %s</p>
            <p><strong>CPU:</strong> %s</p>
        </div>
        <div>
            <p><strong>Generated:</strong> %s</p>
            <p><strong>Benchmarks:</strong> %d</p>
        </div>
    </div>
    `, benchmark.Package, benchmark.GOOS, benchmark.GOARCH, benchmark.CPU, 
       time.Now().Format("2006-01-02 15:04:05"), len(benchmark.Results))
	header.SetContent(headerContent)
	page.AddCustomCharts(header)
	
	// Performance Time Chart (ns/op)
	timeChart := charts.NewBar()
	timeChart.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Theme: types.ThemeWesteros,
		}),
		charts.WithTitleOpts(opts.Title{
			Title: "Benchmark Performance (ns/op)",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    true,
			Trigger: "axis",
		}),
		charts.WithToolboxOpts(opts.Toolbox{
			Show:    true,
			Feature: toolbox,
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: true,
		}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Start: 0,
			End:   100,
		}),
	)
	
	// Get benchmark names
	benchmarkNames := make([]string, 0, len(benchmark.Results))
	for name := range benchmark.Results {
		// Get shorter display names
		displayName := strings.TrimPrefix(name, "Benchmark")
		benchmarkNames = append(benchmarkNames, displayName)
	}
	sort.Strings(benchmarkNames)
	
	// Set x-axis
	timeChart.SetXAxis(benchmarkNames)
	
	// Create series for times
	timeSeries := opts.BarSeries{
		Name: "Time (ns/op)",
		Data: make([]opts.BarData, 0, len(benchmarkNames)),
	}
	
	for _, name := range benchmarkNames {
		fullName := "Benchmark" + name
		if result, exists := benchmark.Results[fullName]; exists {
			timeSeries.Data = append(timeSeries.Data, opts.BarData{
				Name:  name,
				Value: result.TimePerOp,
			})
		} else {
			timeSeries.Data = append(timeSeries.Data, opts.BarData{
				Name:  name,
				Value: 0,
			})
		}
	}
	
	timeChart.AddSeries("Time (ns/op)", timeSeries.Data)
	
	// Add the chart to the page
	page.AddCharts(timeChart)
	
	// Memory Usage Chart (B/op)
	memChart := charts.NewBar()
	memChart.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Theme: types.ThemeWesteros,
		}),
		charts.WithTitleOpts(opts.Title{
			Title: "Memory Usage (B/op)",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    true,
			Trigger: "axis",
		}),
		charts.WithToolboxOpts(opts.Toolbox{
			Show:    true,
			Feature: toolbox,
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: true,
		}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Start: 0,
			End:   100,
		}),
	)
	
	// Set x-axis
	memChart.SetXAxis(benchmarkNames)
	
	// Create series for memory
	memSeries := opts.BarSeries{
		Name: "Memory (B/op)",
		Data: make([]opts.BarData, 0, len(benchmarkNames)),
	}
	
	for _, name := range benchmarkNames {
		fullName := "Benchmark" + name
		if result, exists := benchmark.Results[fullName]; exists {
			memSeries.Data = append(memSeries.Data, opts.BarData{
				Name:  name,
				Value: result.BytesPerOp,
			})
		} else {
			memSeries.Data = append(memSeries.Data, opts.BarData{
				Name:  name,
				Value: 0,
			})
		}
	}
	
	memChart.AddSeries("Memory (B/op)", memSeries.Data)
	
	// Add the chart to the page
	page.AddCharts(memChart)
	
	// If we have a baseline, add comparison charts
	if baseline != nil {
		comparisons := compareBenchmarks(baseline, benchmark, threshold)
		
		// Performance Comparison Chart
		compareChart := charts.NewBar()
		compareChart.SetGlobalOptions(
			charts.WithInitializationOpts(opts.Initialization{
				Theme: types.ThemeWesteros,
			}),
			charts.WithTitleOpts(opts.Title{
				Title: "Performance Change (%)",
			}),
			charts.WithTooltipOpts(opts.Tooltip{
				Show:    true,
				Trigger: "axis",
			}),
			charts.WithToolboxOpts(opts.Toolbox{
				Show:    true,
				Feature: toolbox,
			}),
			charts.WithLegendOpts(opts.Legend{
				Show: true,
			}),
			charts.WithDataZoomOpts(opts.DataZoom{
				Start: 0,
				End:   100,
			}),
		)
		
		// Get comparison names
		compNames := make([]string, 0, len(comparisons))
		for _, comp := range comparisons {
			// Get shorter display names
			displayName := strings.TrimPrefix(comp.Name, "Benchmark")
			compNames = append(compNames, displayName)
		}
		
		// Set x-axis
		compareChart.SetXAxis(compNames)
		
		// Create series for percentage changes
		changeSeries := opts.BarSeries{
			Name: "Change (%)",
			Data: make([]opts.BarData, 0, len(compNames)),
		}
		
		for i, name := range compNames {
			// Negative percentage is an improvement (faster)
			// Use different colors for improvements vs. regressions
			var itemStyle *opts.ItemStyle
			if comparisons[i].DeltaPct < 0 {
				// Green for improvement
				itemStyle = &opts.ItemStyle{
					Color: "#91cc75",
				}
			} else {
				// Red for regression
				itemStyle = &opts.ItemStyle{
					Color: "#ee6666",
				}
			}
			
			changeSeries.Data = append(changeSeries.Data, opts.BarData{
				Name:      name,
				Value:     math.Round(comparisons[i].DeltaPct*100) / 100, // Round to 2 decimal places
				ItemStyle: itemStyle,
			})
		}
		
		compareChart.AddSeries("Change (%)", changeSeries.Data)
		
		// Add the chart to the page
		page.AddCharts(compareChart)
		
		// Comparison table
		compTable := components.NewDivWithConf(&components.DivConfiguration{
			Padding: 10,
			Width:   "100%",
			Height:  "auto",
		})
		
		tableHTML := `
        <h2>Benchmark Comparisons</h2>
        <table style="width: 100%; border-collapse: collapse; margin-top: 20px;">
            <thead>
                <tr style="background-color: #f2f2f2;">
                    <th style="padding: 8px; text-align: left; border: 1px solid #ddd;">Benchmark</th>
                    <th style="padding: 8px; text-align: right; border: 1px solid #ddd;">Baseline (ns/op)</th>
                    <th style="padding: 8px; text-align: right; border: 1px solid #ddd;">Current (ns/op)</th>
                    <th style="padding: 8px; text-align: right; border: 1px solid #ddd;">Delta (ns)</th>
                    <th style="padding: 8px; text-align: right; border: 1px solid #ddd;">Delta (%)</th>
                    <th style="padding: 8px; text-align: right; border: 1px solid #ddd;">Speedup</th>
                    <th style="padding: 8px; text-align: right; border: 1px solid #ddd;">Memory Change (%)</th>
                </tr>
            </thead>
            <tbody>
        `
		
		for _, comp := range comparisons {
			// Determine row style based on significance
			rowStyle := ""
			if comp.Significant {
				if comp.DeltaPct < 0 {
					// Green background for significant improvement
					rowStyle = "background-color: rgba(145, 204, 117, 0.2);"
				} else {
					// Red background for significant regression
					rowStyle = "background-color: rgba(238, 102, 102, 0.2);"
				}
			}
			
			// Format the comparison row
			tableHTML += fmt.Sprintf(`
            <tr style="%s">
                <td style="padding: 8px; text-align: left; border: 1px solid #ddd;">%s</td>
                <td style="padding: 8px; text-align: right; border: 1px solid #ddd;">%.2f</td>
                <td style="padding: 8px; text-align: right; border: 1px solid #ddd;">%.2f</td>
                <td style="padding: 8px; text-align: right; border: 1px solid #ddd;">%.2f</td>
                <td style="padding: 8px; text-align: right; border: 1px solid #ddd;">%.2f%%</td>
                <td style="padding: 8px; text-align: right; border: 1px solid #ddd;">%.2fx</td>
                <td style="padding: 8px; text-align: right; border: 1px solid #ddd;">%.2f%%</td>
            </tr>
            `, rowStyle, comp.Name, comp.BaseTime, comp.NewTime, comp.Delta, comp.DeltaPct, comp.Speedup, comp.AllocDelta)
		}
		
		tableHTML += `
            </tbody>
        </table>
        `
		
		compTable.SetContent(tableHTML)
		page.AddCustomCharts(compTable)
	}
	
	// If statistics are requested, calculate and show them
	if showStats && len(benchmarks) >= 2 {
		stats := calculateStatistics(benchmarks)
		
		statsTable := components.NewDivWithConf(&components.DivConfiguration{
			Padding: 10,
			Width:   "100%",
			Height:  "auto",
		})
		
		tableHTML := `
        <h2>Statistical Analysis</h2>
        <table style="width: 100%; border-collapse: collapse; margin-top: 20px;">
            <thead>
                <tr style="background-color: #f2f2f2;">
                    <th style="padding: 8px; text-align: left; border: 1px solid #ddd;">Benchmark</th>
                    <th style="padding: 8px; text-align: right; border: 1px solid #ddd;">Mean (ns/op)</th>
                    <th style="padding: 8px; text-align: right; border: 1px solid #ddd;">StdDev (ns/op)</th>
                    <th style="padding: 8px; text-align: right; border: 1px solid #ddd;">CV (%)</th>
                    <th style="padding: 8px; text-align: right; border: 1px solid #ddd;">Min (ns/op)</th>
                    <th style="padding: 8px; text-align: right; border: 1px solid #ddd;">Max (ns/op)</th>
                </tr>
            </thead>
            <tbody>
        `
		
		// Sort benchmark names for consistent output
		benchNames := make([]string, 0, len(stats))
		for name := range stats {
			benchNames = append(benchNames, name)
		}
		sort.Strings(benchNames)
		
		for _, name := range benchNames {
			s := stats[name]
			
			// Highlight rows with high variability
			rowStyle := ""
			if s["cv"] > 10 {
				// Yellow background for high variability
				rowStyle = "background-color: rgba(255, 205, 86, 0.2);"
			}
			
			tableHTML += fmt.Sprintf(`
            <tr style="%s">
                <td style="padding: 8px; text-align: left; border: 1px solid #ddd;">%s</td>
                <td style="padding: 8px; text-align: right; border: 1px solid #ddd;">%.2f</td>
                <td style="padding: 8px; text-align: right; border: 1px solid #ddd;">%.2f</td>
                <td style="padding: 8px; text-align: right; border: 1px solid #ddd;">%.2f%%</td>
                <td style="padding: 8px; text-align: right; border: 1px solid #ddd;">%.2f</td>
                <td style="padding: 8px; text-align: right; border: 1px solid #ddd;">%.2f</td>
            </tr>
            `, rowStyle, name, s["mean"], s["stddev"], s["cv"], s["min"], s["max"])
		}
		
		tableHTML += `
            </tbody>
        </table>
        `
		
		statsTable.SetContent(tableHTML)
		page.AddCustomCharts(statsTable)
	}
	
	// Write the HTML output
	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()
	
	if err := page.Render(io.MultiWriter(f)); err != nil {
		return fmt.Errorf("failed to render HTML: %w", err)
	}
	
	fmt.Printf("HTML visualization generated: %s\n", outputFile)
	return nil
}

// generateJSONOutput outputs benchmark data as JSON
func generateJSONOutput(benchmarks []*BenchmarkRun, baseline *BenchmarkRun, outputFile string, threshold float64) error {
	output := struct {
		Benchmarks  []*BenchmarkRun   `json:"benchmarks"`
		Baseline    *BenchmarkRun     `json:"baseline,omitempty"`
		Comparisons []ComparisonResult `json:"comparisons,omitempty"`
		Statistics  map[string]map[string]float64 `json:"statistics,omitempty"`
		Generated   string            `json:"generated"`
	}{
		Benchmarks: benchmarks,
		Baseline:   baseline,
		Generated:  time.Now().Format(time.RFC3339),
	}
	
	// Calculate comparisons if baseline is provided
	if baseline != nil && len(benchmarks) > 0 {
		output.Comparisons = compareBenchmarks(baseline, benchmarks[len(benchmarks)-1], threshold)
	}
	
	// Calculate statistics if we have multiple benchmarks
	if len(benchmarks) >= 2 {
		output.Statistics = calculateStatistics(benchmarks)
	}
	
	// Marshal to JSON
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}
	
	fmt.Printf("JSON output generated: %s\n", outputFile)
	return nil
}

// generateSVGOutput generates an SVG chart of benchmark results
func generateSVGOutput(benchmarks []*BenchmarkRun, baseline *BenchmarkRun, outputFile string, threshold float64) error {
	// Use the same charting logic as HTML but output only the SVG
	benchmark := benchmarks[len(benchmarks)-1] // Use the most recent benchmark
	
	// Create a chart
	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Theme:      types.ThemeWesteros,
			AssetsHost: "",
		}),
		charts.WithTitleOpts(opts.Title{
			Title: "Go Benchmark Results",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: true,
		}),
	)
	
	// Get benchmark names
	benchmarkNames := make([]string, 0, len(benchmark.Results))
	for name := range benchmark.Results {
		// Get shorter display names
		displayName := strings.TrimPrefix(name, "Benchmark")
		benchmarkNames = append(benchmarkNames, displayName)
	}
	sort.Strings(benchmarkNames)
	
	// Set x-axis
	bar.SetXAxis(benchmarkNames)
	
	// Create series for times
	var items []opts.BarData
	for _, name := range benchmarkNames {
		fullName := "Benchmark" + name
		if result, exists := benchmark.Results[fullName]; exists {
			items = append(items, opts.BarData{
				Name:  name,
				Value: result.TimePerOp,
			})
		} else {
			items = append(items, opts.BarData{
				Name:  name,
				Value: 0,
			})
		}
	}
	
	bar.AddSeries("Time (ns/op)", items)
	
	// Create file
	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()
	
	// Render SVG
	if err := bar.Render(io.MultiWriter(f)); err != nil {
		return fmt.Errorf("failed to render SVG: %w", err)
	}
	
	fmt.Printf("SVG chart generated: %s\n", outputFile)
	return nil
}