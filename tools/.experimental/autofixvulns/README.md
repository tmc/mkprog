# AutoFixVulns

AutoFixVulns is a Go program that automatically detects and fixes vulnerabilities in Go projects using `govulncheck`. It updates vulnerable dependencies, adjusts the Go version if necessary, and provides a detailed report of the changes made.

## Features

- Detects vulnerabilities using `govulncheck`
- Automatically updates vulnerable dependencies
- Updates the Go version for standard library vulnerabilities
- Runs `go mod tidy` after making changes
- Provides a summary of changes made
- Works on both the current project and Go projects in subdirectories
- Handles errors gracefully with informative messages
- Uses AI-assisted suggestions for complex vulnerabilities
- Includes logging and verbose output options
- Offers a dry-run mode to preview changes
- Generates a detailed vulnerability report

## Installation

1. Ensure you have Go 1.21 or later installed.
2. Install `govulncheck`:

```
go install golang.org/x/vuln/cmd/govulncheck@latest
```

3. Clone this repository:

```
git clone https://github.com/yourusername/autofixvulns.git
cd autofixvulns
```

4. Build the program:

```
go build
```

## Usage

Run AutoFixVulns in the root directory of your Go project:

```
./autofixvulns [flags] [directory]
```

Flags:
- `-v, --verbose`: Enable verbose output
- `--dry-run`: Show changes without making them

If no directory is specified, AutoFixVulns will run in the current directory.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

