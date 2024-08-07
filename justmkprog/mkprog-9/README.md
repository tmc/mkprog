# mkprog

mkprog is a command-line tool that generates a complete Go project structure based on a user-provided description using AI-powered code generation.

## Features

- Generate a complete Go project structure
- Use AI models (Anthropic, OpenAI, Cohere) for code generation
- Support for custom templates
- Dry-run option to preview generated content
- Configuration file support
- Progress indicator during generation
- Concurrent file writing for improved performance

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

- `--api-key`: API key for the AI service (required)
- `--output`: Output directory for the generated project (default: current directory)
- `--template`: Custom template file (optional)
- `--dry-run`: Preview generated content without creating files
- `--ai-model`: AI model to use (anthropic, openai, cohere) (default: anthropic)
- `--project-type`: Project template (cli, web, library) (default: cli)

### Example

```
mkprog --api-key=your-api-key --output=./my-project "Create a CLI tool that fetches weather data from an API and displays it in a formatted table"
```

## Configuration

mkprog supports configuration files in YAML format. Create a file named `mkprog.yaml` in your home directory (`~/.config/mkprog/mkprog.yaml`) or in the current directory to set default values for flags.

Example configuration file:

```yaml
api-key: your-default-api-key
ai-model: anthropic
project-type: cli
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

