# depvis

A dependency visualization tool for Go projects that generates interactive dependency graphs.

## Features

- Analyzes Go project dependencies at the package level
- Generates interactive HTML visualization of dependencies
- Supports filtering by path prefixes
- Identifies circular dependencies and potential bottlenecks
- Displays dependency metrics (fan-in, fan-out)
- Provides focused views for specific packages
- Generates SVG/PNG exports of the dependency graph

## Installation

```
go install github.com/tmc/mkprog/tools/depvis@latest
```

## Usage

```
depvis [options] [path]
```

Options:
- `-o, --output <file>`: Output HTML file (default: "deps.html")
- `-f, --format <format>`: Output format: html, svg, png, dot (default: "html")
- `-d, --depth <n>`: Maximum dependency depth to analyze (default: unlimited)
- `-i, --include <prefix>`: Only include packages with this prefix (can be repeated)
- `-e, --exclude <prefix>`: Exclude packages with this prefix (can be repeated)
- `-s, --stdlib`: Include standard library dependencies
- `-v, --verbose`: Enable verbose output

## Examples

```bash
# Generate dependency graph for current project
depvis

# Generate SVG visualization
depvis -f svg -o deps.svg

# Focus on specific packages
depvis -i github.com/myorg/myproject

# Exclude standard library
depvis -e std

# Limit depth to direct dependencies
depvis -d 1
```

## License

MIT