# mkprog

mkprog is a command-line tool that generates a complete Go project structure based on a user-provided description using AI. It leverages the Anthropic API to create code, documentation, and project files.

## Features

- Generate a complete Go project structure from a description
- Use AI to create code, documentation, and README content
- Support for multiple AI models (Anthropic, OpenAI, Cohere)
- Customizable project templates (CLI tool, web server, library)
- Dry-run option to preview generated content
- Configuration file support for default values
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
- `--model`: AI model to use (anthropic, openai, cohere) (default: anthropic)
- `--template`: Project template (cli, web, library) (default: cli)
- `--custom-template`: Path to a custom template file
- `--dry-run`: Preview generated content without creating files
- `--temperature`: Temperature for AI generation (default: 0.1)
- `--max-tokens`: Maximum number of tokens for AI generation (default: 8192)

### Example

```
mkprog --api-key=your-api-key --output=./my-project --template=web "Create a simple web server that serves a REST API for a todo list application"
```

## Configuration

You can create a configuration file named `mkprog.yaml` in either `$HOME/.config/mkprog/` or the current directory to set default values for flags. For example:

```yaml
api-key: your-default-api-key
model: anthropic
template: cli
temperature: 0.2
max-tokens: 4096
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

