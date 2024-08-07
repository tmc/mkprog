# mkprog

mkprog is a CLI tool that generates a complete Go project structure based on a user-provided description using AI-powered code generation.

## Features

- Generate a complete Go project structure
- Use AI-powered code generation (Anthropic API)
- Support for custom templates
- Dry-run option to preview generated content
- Configurable AI model and project template
- Concurrent file writing for improved performance

## Installation

To install mkprog, use the following command:

```
go install github.com/yourusername/mkprog@latest
```

## Usage

```
mkprog [flags] [project description]
```

### Flags

- `--api-key`: API key for the AI service (required)
- `--output-dir`: Output directory for the generated project (required)
- `--custom-template`: Path to a custom template file (optional)
- `--dry-run`: Preview generated content without creating files
- `--ai-model`: AI model to use (anthropic, openai, cohere) (default: anthropic)
- `--project-template`: Project template (cli, web, library) (default: cli)
- `--max-tokens`: Maximum number of tokens for AI generation (default: 8192)
- `--temperature`: Temperature for AI generation (default: 0.1)

### Example

```
mkprog --api-key YOUR_API_KEY --output-dir ./my-project "Create a CLI tool that converts markdown to HTML"
```

## Configuration

You can create a configuration file named `mkprog.yaml` in either `$HOME/.config/mkprog/` or the current directory to set default values for flags. For example:

```yaml
api-key: YOUR_API_KEY
output-dir: ./projects
ai-model: anthropic
project-template: cli
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

