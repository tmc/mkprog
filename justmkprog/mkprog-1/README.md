# mkprog

mkprog is a Go program that generates a complete Go project structure based on a user-provided description. It uses the Anthropic API to generate code and documentation for the project.

## Features

- Accepts command-line arguments for project configuration
- Implements a configuration file system to store default values
- Creates a complete project structure with main package, additional packages, test files, README.md, and go.mod
- Uses the Anthropic API to generate code, documentation, and README content
- Implements concurrent file writing for improved performance
- Includes a progress indicator during content generation
- Supports a dry-run option to preview generated content
- Handles errors gracefully and provides informative error messages
- Implements proper logging for debugging and monitoring

## Installation

1. Ensure you have Go 1.21 or later installed on your system.
2. Clone this repository:
   ```
   git clone https://github.com/yourusername/mkprog.git
   ```
3. Change to the project directory:
   ```
   cd mkprog
   ```
4. Build the program:
   ```
   go build
   ```

## Usage

```
mkprog [flags] [project description]
```

### Flags

- `-o, --output string`: Output directory for the generated project (required)
- `-k, --api-key string`: API key for the AI service (required)
- `-t, --template string`: Custom template file (optional)
- `-d, --dry-run`: Preview generated content without creating files
- `-m, --ai-model string`: AI model to use (anthropic, openai, cohere) (default "anthropic")
- `-p, --project-type string`: Project template (cli, web, library) (default "cli")

### Example

```
mkprog -o ./my-project -k your-api-key -p web "Create a simple web server that serves a REST API for a todo list application"
```

## Configuration

You can create a configuration file named `mkprog.yaml` in either `$HOME/.config/mkprog/` or the current directory. The configuration file can store default values for the command-line flags.

Example `mkprog.yaml`:

```yaml
api-key: your-default-api-key
ai-model: anthropic
project-type: cli
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

