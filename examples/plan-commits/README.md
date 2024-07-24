# plan-commits

plan-commits is a Go program that analyzes git changes and generates a structured commit plan. It uses the Anthropic AI model to understand the changes and suggest appropriate commit messages.

## Features

- Analyzes the current git diff to understand changes
- Generates a commit plan with multiple commits if necessary
- Supports conventional commit format as an option
- Allows specifying a maximum number of commits
- Outputs the commit plan in a clear, readable format
- Handles errors gracefully and provides informative error messages

## Installation

1. Ensure you have Go 1.20 or later installed on your system.
2. Clone this repository:
   ```
   git clone https://github.com/yourusername/plan-commits.git
   ```
3. Change to the project directory:
   ```
   cd plan-commits
   ```
4. Build the program:
   ```
   go build
   ```

## Usage

Run the program in a git repository with uncommitted changes:

```
./plan-commits [flags]
```

Available flags:
- `-max-commits int`: Maximum number of commits to generate (default 5)
- `-conventional`: Use conventional commit format

Example:
```
./plan-commits -max-commits 3 -conventional
```

This will analyze the current git diff and generate a commit plan with up to 3 commits using the conventional commit format.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

