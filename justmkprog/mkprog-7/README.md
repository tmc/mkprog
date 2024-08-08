# mkprog

mkprog is a command-line tool that generates a complete Go project structure based on a user-provided description using AI-powered code generation. It leverages the Anthropic API to create code and documentation for the project.

## Features

- Generate a complete Go project structure
- Use AI to create code and documentation
- Support for different project templates (CLI, web server, library)
- Dry-run option to preview generated content
- Configurable AI model selection
- Custom template support
- Progress indicator during generation
- Concurrent file writing for improved performance

## Installation

To install mkprog, make sure you have Go installed on your system, then run:

```
go install github.com/yourusername/mkprog@latest
```

## Usage

```
mkprog [flags] <project description>
```

### Flags

- `-o, --output`: Output directory for the generated project (required)
- `-k, --api-key`: API key for the AI service (required)
- `-t, --template`: Custom template file (optional)
- `-d, --dry-run`: Perform a dry run without creating files
- `-m, --ai-model`: AI model to use (anthropic, openai, cohere) (default: anthropic)
- `-p, --project-type`: Project template (cli, web, library) (default: cli)
- `--temperature`: Temperature for AI generation (default: 0.1)
- `--max-tokens`: Maximum tokens for AI generation (default: 8192)

### Example

```
mkprog -o ./my-project -k your-api-key -p cli "Create a CLI tool that converts markdown to HTML"
```

## Configuration

You can create a configuration file named `mkprog.yaml` in either `$HOME/.config/mkprog/` or the current directory to set default values for flags. For example:

```yaml
api-key: your-default-api-key
ai-model: anthropic
project-type: cli
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

