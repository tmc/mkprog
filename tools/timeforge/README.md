# TimeForge

TimeForge is a Go program that attempts to solve a problem by improving previous commits if the current attempt fails. It uses the 'try' tool to execute commands and analyzes git history to find and improve related commits.

## Installation

1. Ensure you have Go 1.20 or later installed on your system.
2. Clone this repository:
   ```
   git clone https://github.com/yourusername/timeforge.git
   ```
3. Change to the project directory:
   ```
   cd timeforge
   ```
4. Build the program:
   ```
   go build
   ```

## Usage

```
timeforge [-attempts N] [-depth M] <command> [args...]
```

- `-attempts`: number of historical points to try improving (default: 3)
- `-depth`: how far back in history to look for improvement points (default: 10 commits)

Example:
```
timeforge -attempts 5 -depth 20 go test ./...
```

This command will attempt to run `go test ./...` and, if it fails, will try to improve up to 5 previous commits within the last 20 commits.

## Features

- Executes commands using the 'try' tool
- Analyzes git history to find related commits
- Creates new branches from selected previous commits
- Attempts to improve code or configuration at historical points
- Re-runs subsequent commits up to the current task
- Merges improved branches if successful
- Uses git notes to record improvement attempts and results

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

