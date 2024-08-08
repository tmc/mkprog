# mkprog

mkprog is a command-line tool that generates a complete Go project structure based on a user-provided description using AI language models.

## Features

- Generate Go project structure based on a description
- Support for multiple AI models (Anthropic, OpenAI, Cohere)
- Custom project templates (CLI tool, web server, library)
- Dry-run option to preview generated content
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

- `-o, --output string`: Output directory for the generated project (required)
- `-k, --api-key string`: API key for the AI service (required)
- `-t, --template string`: Custom template file
- `-d, --dry-run`: Perform a dry run without creating files
- `-m, --ai-model string`: AI model to use (anthropic, openai, cohere) (default "anthropic")
- `-p, --project-type string`: Project template (cli, web, library) (default "cli")
- `--temperature float`: AI model temperature (default 0.1)
- `--max-tokens int`: Maximum number of tokens for AI response (default 8192)
- `-v, --verbose`: Enable verbose logging

### Example

```
mkprog -o ./my-project -k your-api-key "Create a CLI tool that converts markdown to HTML"
```

## Configuration

mkprog supports configuration files in YAML format. It looks for a configuration file named `mkprog.yaml` in the following locations:

1. `$HOME/.config/mkprog/mkprog.yaml`
2. Current directory

You can set default values for flags in the configuration file. For example:

```yaml
api-key: your-default-api-key
ai-model: anthropic
project-type: cli
temperature: 0.1
max-tokens: 8192
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

