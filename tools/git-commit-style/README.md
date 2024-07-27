# git-commit-style

git-commit-style is a tool that analyzes the git history of a repository and compiles guidance based on commit messages, diffstats, and additional context from a `.git-commit-style` directory.

## Features

- Analyzes git commit history
- Incorporates context from `.git-commit-style` directories
- Generates tailored guidance for repository maintenance and commit style

## Installation

1. Ensure you have Go 1.20 or later installed on your system.
2. Clone this repository:
   ```
   git clone https://github.com/yourusername/git-commit-style.git
   ```
3. Navigate to the project directory:
   ```
   cd git-commit-style
   ```
4. Build the project:
   ```
   go build
   ```

## Usage

1. Navigate to the root directory of the git repository you want to analyze.
2. Run the git-commit-style tool:
   ```
   /path/to/git-commit-style
   ```
3. The tool will analyze the repository's commit history and any context files found in `.git-commit-style` directories, then output guidance for the repository.

## Context Files

You can provide additional context for the tool by creating a `.git-commit-style` directory in the repository or any parent directory. Any files in this directory will be read and incorporated into the analysis.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

