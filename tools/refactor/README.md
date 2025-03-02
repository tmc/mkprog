# refactor

A tool that analyzes Go code and suggests or implements refactoring improvements.

## Features

- Analyzes Go code for potential improvements
- Identifies common anti-patterns and suggests fixes
- Implements refactoring changes with user confirmation
- Preserves code style and documentation
- Supports focused refactoring of specific functions or packages

## Installation

```
go install github.com/tmc/mkprog/tools/refactor@latest
```

## Usage

```
refactor [options] [path]
```

Options:
- `-f, --file <path>`: Specific file to analyze (defaults to all Go files in current directory)
- `-i, --interactive`: Interactive mode - ask for confirmation before making changes
- `-d, --dry-run`: Print suggested changes without modifying files
- `-v, --verbose`: Enable verbose output

## Examples

```bash
# Analyze all Go files in current directory
refactor

# Analyze a specific file
refactor -f main.go

# Analyze and interactively apply changes
refactor -i

# Show suggested changes without modifying files
refactor -d
```

## License

MIT