# mkprog

mkprog is a command-line tool that generates a complete Go project structure based on a user-provided description using AI-powered code generation.

## Features

- Generate project structure and files based on a description
- Support for different project templates (CLI, web server, library)
- Customizable output directory
- Dry-run option to preview generated content
- Configurable AI model selection
- Progress indicator during generation
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

- `--api-key`: API key for the AI service (required)
- `-o, --output`: Output directory for the generated project (required)
- `--template`: Custom template file (optional)
- `--dry-run`: Preview generated content without creating files
- `--ai-model`: AI model to use (anthropic, openai, cohere) (default: anthropic)
- `--project-type`: Project template (cli, web, library) (default: cli)
- `--max-tokens`: Maximum number of tokens for AI generation (default: 8192)
- `--temperature`: Temperature for AI generation (default: 0.1)

### Example

```
mkprog --api-key=your-api-key --output=./my-project "Create a CLI tool that converts markdown to HTML"
```

## Configuration

mkprog supports configuration files in YAML format. It looks for a configuration file named `mkprog.yaml` in the following locations:

1. `$HOME/.config/mkprog/mkprog.yaml`
2. Current directory

You can set default values for flags in the configuration file. For example:

```yaml
api-key: your-default-api-key
output: ./default-output-dir
ai-model: anthropic
project-type: cli
max-tokens: 8192
temperature: 0.1
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

