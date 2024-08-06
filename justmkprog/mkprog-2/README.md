# mkprog

mkprog is a command-line tool that generates a complete Go project structure based on a user-provided description. It uses the Anthropic API to generate code and documentation for the project.

## Features

- Generates a complete Go project structure
- Uses AI to create code, documentation, and README content
- Supports custom templates and project types
- Implements concurrent file writing for improved performance
- Includes a dry-run option to preview generated content
- Configurable via command-line flags and configuration file

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
- `-k, --api-key`: API key for the AI model (required)
- `-t, --template`: Custom template file (optional)
- `-d, --dry-run`: Preview generated content without creating files
- `-m, --ai-model`: AI model to use (anthropic, openai, cohere) (default: anthropic)
- `-p, --project-type`: Project template (cli, web, library) (default: cli)

### Example

```
mkprog -o ./my-project -k your-api-key "Create a CLI tool that converts markdown to HTML"
```

## Configuration

mkprog supports configuration via a YAML file. Create a file named `mkprog.yaml` in either `$HOME/.config/mkprog/` or the current directory with the following structure:

```yaml
api-key: your-api-key
output: ./default-output-dir
ai-model: anthropic
project-type: cli
```

You can also use environment variables prefixed with `MKPROG_` to set configuration values.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

