# fixme

fixme is a command-line tool designed to help debug and fix issues with Go modules and other shell commands. It uses AI-powered suggestions to provide potential fixes for failed commands.

## Features

- Executes shell commands and analyzes their output
- Provides AI-generated suggestions for fixing failed commands
- Focuses on Go module-related issues, but can handle general shell commands
- Supports command history and custom descriptions for better context

## Installation

To install fixme, make sure you have Go installed on your system, then run:

```
go install github.com/tmc/mkprog/tools/fixme@latest
```

Replace `tmc` with the appropriate GitHub username or organization.

## Usage

To use fixme, run it with the following syntax:

```
fixme [flags] -- <command> [args...]
```

For example:

```
fixme -- go mod tidy
```

### Flags

- `-hist`: Use command history for better suggestions
- `-desc <description>`: Provide a custom description of the issue

## Examples

1. Basic usage:
   ```
   fixme -- go build ./...
   ```

2. With history and description:
   ```
   fixme -hist -desc "Trying to update dependencies" -- go get -u ./...
   ```

## How it works

1. fixme executes the provided command
2. If the command fails, it analyzes the output and error message
3. It then generates a suggestion using AI, considering the command context, fix history, and project structure
4. The user can choose to apply the suggestion or exit

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
