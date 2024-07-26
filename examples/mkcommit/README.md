# mkcommit

mkcommit is a Go program that generates suitable Git commit messages and commands based on the current working tree and repository context. It analyzes changed files, recent commit history, and project-specific conventions to create meaningful and consistent commit messages.

## Features

- Git integration using go-git
- Natural language processing for commit message analysis and generation
- Configurable commit message templates and style preferences
- Support for conventional commits and semantic versioning
- Command-line interface with optional flags for customization
- Interactive mode for user confirmation or modification of the generated commit

## Installation

1. Ensure you have Go 1.16 or later installed on your system.
2. Clone this repository:
   ```
   git clone https://github.com/yourusername/mkcommit.git
   ```
3. Change to the project directory:
   ```
   cd mkcommit
   ```
4. Build the program:
   ```
   go build
   ```

## Usage

Run the program in your Git repository:

```
./mkcommit [flags]
```

Available flags:
- `-t, --type`: Specify the commit type (e.g., feat, fix, docs)
- `-s, --scope`: Specify the commit scope
- `-i, --interactive`: Enable interactive mode for user confirmation or modification

Example usage:
```
./mkcommit -t feat -s auth -i
```

This command will generate a commit message for a new feature in the "auth" scope and prompt for user confirmation or modification.

## Dependencies

- github.com/go-git/go-git/v5
- github.com/spf13/cobra
- github.com/tmc/langchaingo
- golang.org/x/exp/slices

## Error Handling

- Graceful handling of non-Git repositories
- Proper error messages for merge conflicts or uncommitted changes
- Fallback options for insufficient context or history

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

