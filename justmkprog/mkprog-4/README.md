# mkprog

mkprog is a command-line tool that generates a complete Go project structure based on a user-provided description using AI. It leverages the Anthropic API to create code, documentation, and project files.

## Features

- Generate project structure based on description
- Support for multiple AI models (Anthropic, OpenAI, Cohere)
- Custom project templates (CLI tool, web server, library)
- Dry-run option for previewing generated content
- Concurrent file writing for improved performance
- Progress indicator during content generation
- Configuration file support for default values

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

- `-o, --output`: Output directory for the generated project (required)
- `-k, --api-key`: API key for the AI service (required)
- `-t, --template`: Custom template file (optional)
- `-d, --dry-run`: Perform a dry run without creating files
- `-m, --ai-model`: AI model to use (anthropic, openai, cohere) (default: anthropic)
- `-p, --project-type`: Project template (cli, web, library) (default: cli)
- `--temperature`: AI model temperature (default: 0.1)
- `--max-tokens`: Maximum number of tokens for AI response (default: 8192)
- `-v, --verbose`: Enable verbose logging

### Example

```
mkprog -o ./my-project -k your-api-key "Create a CLI tool for managing todo lists"
```

## Configuration

mkprog supports configuration files in YAML format. Create a file named `mkprog.yaml` in one of the following locations:

- `$HOME/.config/mkprog/mkprog.yaml`
- `./mkprog.yaml` (current directory)

Example configuration:

```yaml
api-key: your-default-api-key
ai-model: anthropic
project-type: cli
temperature: 0.2
max-tokens: 4096
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

