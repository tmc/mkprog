# mkprog

mkprog is a command-line tool that generates a complete Go project structure based on a user-provided description using AI-powered code generation.

## Features

- Generate a complete Go project structure
- Use AI (Anthropic API) to generate code and documentation
- Support for custom templates
- Dry-run option to preview generated content
- Concurrent file writing for improved performance
- Caching system to store and reuse previously generated content
- Configuration file support for default values

## Installation

To install mkprog, make sure you have Go installed on your system, then run:

```
go install github.com/yourusername/mkprog@latest
```

## Usage

```
mkprog [flags] [project description]
```

### Flags

- `-o, --output string`: Output directory for the generated project (required)
- `-k, --api-key string`: API key for the AI service (required)
- `-t, --template string`: Custom template file
- `-d, --dry-run`: Preview generated content without creating files
- `-m, --ai-model string`: AI model to use (anthropic, openai, cohere) (default "anthropic")
- `-p, --project-type string`: Project template (cli, web, library) (default "cli")

### Example

```
mkprog -o ./my-project -k your-api-key "Create a CLI tool that converts markdown to HTML"
```

## Configuration

mkprog supports configuration files to store default values for flags. It looks for a configuration file named `mkprog.yaml` in the following locations:

1. `$HOME/.config/mkprog/mkprog.yaml`
2. Current directory

You can also use environment variables prefixed with `MKPROG_` to set configuration values.

Example configuration file:

```yaml
api-key: your-default-api-key
ai-model: anthropic
project-type: cli
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

