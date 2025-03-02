# benchviz

A tool for parsing and visualizing Go benchmark results with comparisons and trend analysis.

## Features

- Parses benchmark results from `go test -bench`
- Creates interactive visualizations of benchmark data
- Compares results across multiple benchmark runs
- Tracks performance trends over time
- Highlights significant performance changes
- Supports various output formats (HTML, SVG, JSON)
- Generates statistical analysis of benchmark variability
- Integrates with continuous integration workflows

## Installation

```
go install github.com/tmc/mkprog/tools/benchviz@latest
```

## Usage

```
benchviz [options] [benchmark_files...]
```

Options:
- `-o, --output <file>`: Output HTML file (default: "benchmark.html")
- `-f, --format <format>`: Output format: html, svg, json (default: "html")
- `-b, --baseline <file>`: Baseline benchmark file for comparison
- `-t, --threshold <percent>`: Highlight changes above this threshold (default: 20%)
- `-p, --profile`: Include CPU/memory profile visualization
- `-s, --stat`: Show detailed statistical analysis
- `-v, --verbose`: Enable verbose output

## Examples

```bash
# Visualize a single benchmark file
benchviz benchmark.txt

# Compare two benchmark runs
benchviz -b baseline.txt current.txt

# Visualize with detailed statistics
benchviz -s benchmark.txt

# Generate JSON output for CI integration
benchviz -f json -o results.json benchmark.txt

# Visualize with CPU profile data
benchviz -p benchmark.txt
```

## Input Format

The tool accepts standard Go benchmark output format:

```
goos: darwin
goarch: amd64
pkg: github.com/example/package
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkFunction-12           10000             12045 ns/op           1024 B/op          8 allocs/op
BenchmarkOtherFunction-12       5000             31254 ns/op           2048 B/op         16 allocs/op
```

## License

MIT