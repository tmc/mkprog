# Attempt Analyzer

Attempt Analyzer is a Go program that analyzes and scores attempt repositories. It extracts commit history and Git notes, uses AI to score attempts, and generates a summary of all attempts.

## Installation

1. Ensure you have Go 1.20 or later installed on your system.
2. Clone this repository:
   ```
   git clone https://github.com/yourusername/attempt-analyzer.git
   ```
3. Change to the project directory:
   ```
   cd attempt-analyzer
   ```
4. Install dependencies:
   ```
   go mod download
   ```

## Usage

Run the program with the path to the Git repository you want to analyze:

```
go run main.go /path/to/repository
```

The program will analyze the repository, score the attempts, and generate a summary of all attempts.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

