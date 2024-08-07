# mkprog

mkprog is a CLI tool that generates a complete Go project structure based on a user-provided description using AI-powered code generation.

## Features

- Generate a complete Go project structure
- Use AI-powered code generation (Anthropic API)
- Support for custom templates
- Dry-run option for previewing generated content
- Configurable project types (CLI, web server, library)
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
mkprog [flags] [project description]
```

### Flags

- `-o, --output string`: Output directory for the generated project
- `-k, --api-key string`: API key for the AI service
- `-t, --template string`: Custom template file
- `-d, --dry-run`: Perform a dry run without creating files
- `-m, --ai-model string`: AI model to use (anthropic, openai, cohere) (default "anthropic")
- `-p, --project-type string`: Project template (cli, web, library) (default "cli")

### Example

```
mkprog -o ./my-project -k your-api-key "Create a CLI tool that fetches weather data from an API and displays it in a formatted table"
```

## Configuration

mkprog supports configuration files in YAML format. The configuration file can be placed in the following locations:

- `$HOME/.config/mkprog/mkprog.yaml`
- `./mkprog.yaml` (current directory)

Example configuration file:

```yaml
api-key: your-default-api-key
output: ./default-output-directory
ai-model: anthropic
project-type: cli
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

