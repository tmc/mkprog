# mkprog

mkprog is a command-line tool that generates a complete Go project structure based on a user-provided description using AI-powered code generation.

## Features

- Generate Go project structure based on a description
- Support for multiple AI models (Anthropic, OpenAI, Cohere)
- Customizable project templates (CLI, web server, library)
- Dry-run option to preview generated content
- Configuration file support
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

mkprog supports configuration files in YAML format. Create a file named `mkprog.yaml` in one of the following locations:

- `$HOME/.config/mkprog/mkprog.yaml`
- `./mkprog.yaml` (current directory)

Example configuration file:

```yaml
api-key: your-api-key
model: anthropic
template: cli
temperature: 0.2
max-tokens: 4096
```

You can also use environment variables to set configuration values. Prefix the flag name with `MKPROG_` and use uppercase letters. For example:

```
export MKPROG_API_KEY=your-api-key
export MKPROG_MODEL=anthropic
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

