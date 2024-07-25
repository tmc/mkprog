# Backport Changes

Backport Changes is a Go program that analyzes git history, identifies non-machine-made changes, and attempts to apply these changes throughout the repository's history while re-running code generation.

## Features

- Analyze git history to identify non-machine-made changes
- Create a new branch for backported changes
- Apply identified changes throughout the git history
- Re-run code generation steps
- Resolve conflicts with optional AI assistance
- Update commit messages to reflect backported changes

## Installation

1. Ensure you have Go 1.20 or later installed.
2. Clone this repository:
   ```
   git clone https://github.com/yourusername/backport-changes.git
   ```
3. Change to the project directory:
   ```
   cd backport-changes
   ```
4. Build the program:
   ```
   go build
   ```

## Usage

Run the program from within a git repository:

```
./backport-changes [flags]
```

### Flags

- `--start-commit`: Specify the earliest commit to start backporting (default: first commit)
- `--end-commit`: Specify the latest commit to backport to (default: HEAD)
- `--dry-run`: Show what would be done without making actual changes
- `--ai-assist`: Use AI to help resolve conflicts (requires OpenAI API key)
- `--verbose`: Provide detailed output of the backporting process

### Examples

1. Backport changes with default options:
   ```
   ./backport-changes
   ```

2. Backport changes between specific commits:
   ```
   ./backport-changes --start-commit abc123 --end-commit def456
   ```

3. Perform a dry run with AI assistance:
   ```
   ./backport-changes --dry-run --ai-assist
   ```

## Configuration

To use AI assistance for conflict resolution, set your OpenAI API key as an environment variable:

```
export OPENAI_API_KEY=your_api_key_here
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

