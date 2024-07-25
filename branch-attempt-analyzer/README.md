# Branch Attempt Analyzer

This Go program attempts to perform a specified task on a Git branch multiple times, analyzes the results, and optionally performs multiple runs for meta-comparison.

## Installation

1. Ensure you have Go 1.20 or later installed on your system.
2. Clone this repository:
   ```
   git clone https://github.com/yourusername/branch-attempt-analyzer.git
   ```
3. Change to the project directory:
   ```
   cd branch-attempt-analyzer
   ```
4. Install dependencies:
   ```
   go mod download
   ```

## Usage

Run the program with the following command:

```
go run main.go [flags] <task command>
```

### Flags

- `-attempts`: Number of attempts per run (default 10)
- `-runs`: Number of meta-comparison runs (default 1)
- `-branch`: Name of the branch to use for attempts (default "attempt-branch")

### Example

```
go run main.go -attempts 5 -runs 2 -branch test-branch "echo 'Hello, World!'"
```

This command will:
1. Perform 5 attempts of the "echo 'Hello, World!'" command on a branch named "test-branch"
2. Analyze the results of these attempts
3. Repeat this process twice (2 runs)
4. Perform a meta-analysis of both runs

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

