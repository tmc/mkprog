# Auto Git Commit

Auto Git Commit is an intelligent Git commit message generator designed to streamline the software development workflow. The program analyzes changes made to a Git repository and automatically generates meaningful, descriptive commit messages based on the modifications.

## Features

1. Automatic change analysis: Scans modified files and detects the nature of changes (e.g., bug fixes, new features, refactoring).
2. Smart commit message generation: Creates concise yet informative commit messages based on the detected changes.
3. Customizable templates: Allows users to define and use custom commit message templates.
4. Integration options: Works as a command-line tool and offers plugins for popular IDEs (e.g., VS Code, IntelliJ).
5. Conventional Commits support: Follows the Conventional Commits specification for structured commit messages.

## Installation

1. Ensure you have Go 1.20 or later installed on your system.
2. Clone this repository:
   ```
   git clone https://github.com/yourusername/auto-git-commit.git
   ```
3. Change to the project directory:
   ```
   cd auto-git-commit
   ```
4. Build the project:
   ```
   go build
   ```

## Usage

1. Navigate to your Git repository:
   ```
   cd /path/to/your/repo
   ```
2. Run the Auto Git Commit tool:
   ```
   /path/to/auto-git-commit
   ```
3. The tool will analyze the changes in your repository and generate a commit message.
4. Review the generated message and confirm if you want to proceed with the commit.

## Error Handling

- The program gracefully handles cases where no changes are detected.
- Clear error messages are provided for issues like unreadable files or Git conflicts.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

