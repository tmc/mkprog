# coderev - AI-assisted Code Review Tool

`coderev` is a tool that helps developers perform automated code reviews with AI assistance. It provides actionable feedback on code quality, potential issues, and improvement suggestions.

## Features

- Analyzes Go code for common anti-patterns and best practices
- Generates actionable, prioritized recommendations
- Supports integration with CI/CD pipelines
- Can analyze git diffs or entire directories
- Configurable rule sets for team-specific standards

## Installation

```bash
go install github.com/tmc/mkprog/tools/coderev@latest
```

## Usage

```bash
# Review all code in the current directory
coderev ./...

# Review changes in a git branch (compared to main)
coderev --diff main

# Review a specific file
coderev path/to/file.go

# Generate a JSON report
coderev --format=json ./... > report.json

# Use a custom ruleset
coderev --config=.coderev.yaml ./...
```

## Configuration

Create a `.coderev.yaml` file in your project root:

```yaml
# Example configuration
severity_threshold: warning
ignore:
  - "**/vendor/**"
  - "**/generated/**"
rules:
  max_function_length: 100
  require_comments: true
  enforce_error_handling: true
```

## License

MIT