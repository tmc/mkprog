# mkprog

mkprog is a command-line tool that generates a complete Go project structure based on a user-provided description using AI language models.

## Features

- Generate project structure based on user description
- Support for multiple AI models (Anthropic, OpenAI, Cohere)
- Custom project templates (CLI tool, web server, library)
- Dry-run option for previewing generated content
- Concurrent file writing for improved performance
- Progress indicator during content generation
- Configuration file support for default values

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

- `-o, --output`: Output directory for the generated project
- `-k, --api-key`: API key for the AI service
- `-t, --template`: Custom template file
- `-d, --dry-run`: Perform a dry run without creating files
- `-m, --ai-model`: AI model to use (anthropic, openai, cohere)
- `-p, --project-type`: Project template (cli, web, library)
- `--temperature`: AI model temperature (default: 0.1)
- `--max-tokens`: Maximum number of tokens for AI response (default: 8192)
- `-v, --verbose`: Enable verbose logging

### Example

```
mkprog -o ./my-project -k your-api-key -p cli "Create a CLI tool for managing todo lists"
```

## Configuration

mkprog supports configuration files in YAML format. It looks for a configuration file named `mkprog.yaml` in the following locations:

1. `$HOME/.config/mkprog/mkprog.yaml`
2. Current directory

You can also use environment variables prefixed with `MKPROG_` to set configuration values.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

