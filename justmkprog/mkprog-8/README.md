# mkprog

mkprog is a CLI tool that generates a complete Go project structure based on a user-provided description. It uses AI models to generate code and documentation for the project.

## Features

- Generate a complete Go project structure
- Use AI models (Anthropic, OpenAI, Cohere) to generate code and documentation
- Support for custom templates
- Dry-run option to preview generated content
- Concurrent file writing for improved performance
- Progress indicator during content generation
- Caching system for previously generated content
- Comprehensive unit tests
- Support for updating existing Go projects

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

- `-d, --description string`: Project description (required)
- `-o, --output string`: Output directory (required)
- `-k, --api-key string`: API key for AI model (required)
- `-t, --template string`: Custom template file
- `--dry-run`: Dry run (preview generated content)
- `-m, --ai-model string`: AI model to use (anthropic, openai, cohere) (default "anthropic")
- `-p, --project-type string`: Project template (cli, web, library) (default "cli")
- `-v, --verbose`: Verbose output
- `--temperature float`: AI model temperature (default 0.1)
- `--max-tokens int`: Maximum number of tokens for AI response (default 8192)

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
verbose: true
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

