# try-analyze

try-analyze is a Go program that analyzes your Git repository, focusing on recent commit history, 'try' attempts recorded as Git notes, and current branch status. It uses AI to provide insights and suggestions based on the analysis.

## Installation

1. Ensure you have Go installed on your system (version 1.20 or later).
2. Clone this repository:
   ```
   git clone https://github.com/yourusername/try-analyze.git
   ```
3. Navigate to the project directory:
   ```
   cd try-analyze
   ```
4. Build the program:
   ```
   go build
   ```

## Usage

Run try-analyze from within a Git repository:

```
./try-analyze [flags]
```

### Flags

- `--commits N`: Number of recent commits to analyze (default 10)
- `--branch`: Analyze a specific branch instead of the current one
- `--verbose`: Provide more detailed output

### Examples

1. Analyze the last 10 commits of the current branch:
   ```
   ./try-analyze
   ```

2. Analyze the last 20 commits of a specific branch:
   ```
   ./try-analyze --commits 20 --branch feature-branch
   ```

3. Analyze with verbose output:
   ```
   ./try-analyze --verbose
   ```

## Output

The program will provide:

1. A summary of the analyzed data
2. AI-generated analysis of patterns in 'try' attempts
3. Suggested edits or improvements based on the analysis

## Error Handling

try-analyze includes error handling for:

- Git command errors
- API errors when interacting with the AI model
- General program execution errors

If an error occurs, the program will display an informative error message.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

