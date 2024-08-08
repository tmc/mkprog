# mkprog

mkprog is a command-line tool that generates a complete Go project structure based on a user-provided description using AI. It leverages the power of language models to create a project scaffold, including code, documentation, and necessary configuration files.

## Features

- Generate project structure based on a description
- Support for multiple AI models (Anthropic, OpenAI, Cohere)
- Custom project templates
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
temperature: 0.2
max-tokens: 4096
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

