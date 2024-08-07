# mkprog

mkprog is a command-line tool that generates a complete Go project structure based on a user-provided description using AI-powered code generation.

## Features

- Generate Go project structure based on a description
- Support for different project types (CLI, web server, library)
- AI-powered code generation using Anthropic API
- Custom template support
- Dry-run option for previewing generated content
- Interactive mode for step-by-step project configuration
- Git repository initialization
- Docker file generation

## Installation

To install mkprog, make sure you have Go installed on your system, then run:

```
go install github.com/yourusername/mkprog@latest
```

## Usage

```
mkprog [flags] [description]
```

### Flags

- `-o, --output string`: Output directory for the generated project
- `-k, --api-key string`: API key for the AI service
- `-t, --template string`: Custom template file
- `-d, --dry-run`: Perform a dry run without creating files
- `-m, --ai-model string`: AI model to use (anthropic, openai, cohere) (default "anthropic")
- `-p, --project-type string`: Project template (cli, web, library) (default "cli")
- `--temperature float`: AI model temperature (default 0.1)
- `--max-tokens int`: Maximum number of tokens for AI response (default 8192)
- `-v, --verbose`: Enable verbose logging
- `-i, --interactive`: Enable interactive mode
- `--init-git`: Initialize Git repository
- `--generate-docker`: Generate Dockerfile and docker-compose.yml

### Examples

Generate a CLI project:

```
mkprog -o my-cli-project -k your-api-key "Create a CLI tool for managing todo lists"
```

Generate a web server project with Docker files:

```
mkprog -o my-web-project -k your-api-key -p web --generate-docker "Create a RESTful API server for a blog application"
```

Use interactive mode:

```
mkprog -i
```

## Configuration

mkprog supports configuration files in YAML format. Create a file named `mkprog.yaml` in one of the following locations:

- `$HOME/.config/mkprog/mkprog.yaml`
- `./mkprog.yaml` (current directory)

Example configuration:

```yaml
api-key: your-default-api-key
ai-model: anthropic
project-type: cli
temperature: 0.1
max-tokens: 8192
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

