# mkprog

mkprog is a command-line tool that generates a complete Go project structure based on a user-provided description. It uses AI to generate code and documentation for the project.

## Features

- Generate a complete Go project structure
- Use AI to create code and documentation
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
mkprog [flags]
```

### Flags

- `-d, --description`: Project description (required)
- `-o, --output`: Output directory (required)
- `-k, --api-key`: API key (required)
- `-t, --template`: Custom template file (optional)
- `--dry-run`: Dry run (preview generated content)
- `-m, --ai-model`: AI model to use (anthropic, openai, cohere)
- `-p, --project-type`: Project template (cli, web, library)
- `-v, --verbose`: Verbose output
- `--temperature`: AI model temperature (default: 0.1)
- `--max-tokens`: Maximum number of tokens for AI response (default: 8192)

### Example

```
mkprog -d "A CLI tool for managing todo lists" -o ./my-todo-app -k your-api-key -p cli
```

## Configuration

You can create a configuration file named `.mkprog.yaml` in your home directory to set default values for flags. For example:

```yaml
api-key: your-default-api-key
ai-model: anthropic
project-type: cli
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

