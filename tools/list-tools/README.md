# list-tools

A utility for discovering and exploring tools in the mkprog toolchain.

## Features

- Lists all available tools in the mkprog collection
- Organizes tools by category for easier discovery
- Provides detailed descriptions and feature counts
- Supports filtering by category or search term
- Outputs in text, JSON, or Markdown formats
- Caches results for faster repeat use
- Extracts information directly from README files

## Installation

```bash
go install github.com/tmc/mkprog/tools/list-tools@latest
```

## Usage

```
list-tools [options] [path]
```

Options:
- `-c, --category <category>`: Filter tools by category
- `-f, --format <format>`: Output format (text, json, markdown)
- `-s, --search <term>`: Search term for filtering tools
- `-r, --refresh`: Refresh cache and get fresh tool data
- `-v, --verbose`: Show verbose output

## Examples

```bash
# List all available tools
list-tools

# Filter tools by category
list-tools -c "Code Generation"

# Search for tools with specific capabilities
list-tools -s "visualization"

# Generate documentation in Markdown format
list-tools -f markdown > tools-docs.md

# Show detailed information with stats
list-tools -v
```

## License

MIT

