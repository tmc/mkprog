# mkprog

mkprog is a command-line tool that generates a complete Go project structure based on a user-provided description using AI-powered code generation.

## Features

- Generate project structure based on user description
- Support for multiple AI models (Anthropic, OpenAI, Cohere)
- Custom project templates (CLI tool, web server, library)
- Dry-run option for previewing generated content
- Configuration file support
- Concurrent file writing for improved performance
- Progress indicator during content generation

## Installation

To install mkprog, use the following command:

```
go install github.com/yourusername/mkprog@latest
```

## Usage

```
mkprog [flags] "project description"
```

### Flags

- `--api-key`: API key for the AI service
- `--output-dir`: Output directory for the generated project
- `--custom-template`: Path to a custom template file
- `--dry-run`: Preview generated content without creating files
- `--ai-model`: AI model to use (anthropic, openai, cohere)
- `--project-template`: Project template (cli, web, library)
- `--temperature`: Temperature for AI generation (default: 0.1)
- `--max-tokens`: Maximum number of tokens for AI generation (default: 8000)

### Example

```
mkprog --api-key YOUR_API_KEY --output-dir ./my-project --project-template cli "Create a CLI tool for managing todo lists"
```

## Configuration

mkprog supports configuration files in YAML format. The configuration file can be placed in the following locations:

- `$HOME/.config/mkprog/mkprog.yaml`
- `./mkprog.yaml` (current directory)

You can also use environment variables prefixed with `MKPROG_` to set configuration values.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

