# mkprog

mkprog is a Go program that generates a complete Go project structure based on a user-provided description. It uses the Anthropic API to generate code and documentation for the project.

## Features

- Generate a complete Go project structure
- Use AI to create code and documentation
- Support for different project templates (CLI, web server, library)
- Dry-run option to preview generated content
- Concurrent file writing for improved performance
- Progress indicator during content generation

## Installation

To install mkprog, make sure you have Go installed on your system, then run:

```
go install github.com/yourusername/mkprog@latest
```

## Usage

```
mkprog [flags] "project description"
```

### Flags

- `--api-key`: API key for the AI service (required)
- `--output`: Output directory for the generated project (default: current directory)
- `--template`: Custom template file (optional)
- `--dry-run`: Preview generated content without creating files
- `--ai-model`: AI model to use (anthropic, openai, cohere) (default: anthropic)
- `--project-type`: Project template (cli, web, library) (default: cli)
- `--max-tokens`: Maximum number of tokens for AI response (default: 8192)
- `--temperature`: Temperature for AI response (default: 0.1)

### Example

```
mkprog --api-key=your-api-key --output=./my-project --project-type=web "Create a simple web server that serves a REST API for a todo list application"
```

## Configuration

You can create a configuration file named `mkprog.yaml` in your home directory (`$HOME/.config/mkprog/mkprog.yaml`) or in the current directory to set default values for flags. For example:

```yaml
api-key: your-default-api-key
output: ./projects
ai-model: anthropic
project-type: cli
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

