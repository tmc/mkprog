# mkprog

mkprog is a CLI tool that generates a complete Go project structure based on a user-provided description using AI-powered code generation.

## Features

- Generate project structure based on a description
- Support for multiple AI models (Anthropic, OpenAI, Cohere)
- Custom project templates (CLI tool, web server, library)
- Dry-run option to preview generated content
- Concurrent file writing for improved performance
- Configuration file support
- Progress indicator during content generation

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
- `--custom-template`: Path to a custom template file
- `--dry-run`: Preview generated content without creating files
- `--ai-model`: AI model to use (anthropic, openai, cohere)
- `--project-template`: Project template (cli, web, library)
- `--max-tokens`: Maximum number of tokens for AI generation
- `--temperature`: Temperature for AI generation

### Example

```
mkprog --api-key YOUR_API_KEY --output-dir ./my-project "Create a CLI tool for managing todo lists"
```

## Configuration

You can create a configuration file named `mkprog.yaml` in either `$HOME/.config/mkprog/` or the current directory. The configuration file can include default values for the flags.

Example `mkprog.yaml`:

```yaml
api-key: YOUR_API_KEY
output-dir: ./projects
ai-model: anthropic
project-template: cli
max-tokens: 8000
temperature: 0.1
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

