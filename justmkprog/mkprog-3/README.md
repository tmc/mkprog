# mkprog

mkprog is a command-line tool that generates a complete Go project structure based on a user-provided description. It uses the Anthropic API to generate code and documentation for the project.

## Features

- Generate project structure based on description
- Support for different project templates (CLI, web server, library)
- Custom template support
- Dry-run option for previewing generated content
- Concurrent file writing for improved performance
- Progress indicator during content generation
- Caching system for reusing previously generated content
- Comprehensive error handling and logging

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

- `-o, --output`: Output directory for the generated project (required)
- `-k, --api-key`: API key for the AI model (can also be set via ANTHROPIC_API_KEY environment variable)
- `-t, --template`: Custom template file (optional)
- `-d, --dry-run`: Preview generated content without creating files
- `-m, --ai-model`: AI model to use (anthropic, openai, cohere) (default: anthropic)
- `-p, --project-type`: Project template (cli, web, library) (default: cli)

### Example

```
mkprog -o ./my-project -p web "Create a simple web server that serves a REST API for a todo list application"
```

## Configuration

You can create a configuration file named `mkprog.yaml` in the current directory or in `$HOME/.config/mkprog/` to set default values for flags. For example:

```yaml
api-key: your-api-key-here
project-type: web
ai-model: anthropic
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

