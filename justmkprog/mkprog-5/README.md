# mkprog

mkprog is a Go program that generates a complete Go project structure based on a user-provided description. It uses the Anthropic API to generate code and documentation for the project.

## Features

- Accepts command-line arguments for project configuration
- Implements a configuration file system to store default values
- Creates a complete project structure with main package, additional packages, test files, README.md, and go.mod file
- Uses the Anthropic API to generate code, documentation, and README content
- Implements concurrent file writing using goroutines for improved performance
- Adds a progress indicator during content generation
- Implements a caching system to store and reuse previously generated content
- Includes a dry-run option to preview generated content without creating files
- Handles errors gracefully and provides informative error messages
- Implements proper logging for debugging and monitoring

## Installation

To install mkprog, make sure you have Go installed on your system, then run:

```
go get github.com/yourusername/mkprog
```

## Usage

```
mkprog [flags] [project description]
```

### Flags

- `-o, --output string`: Output directory for the generated project (required)
- `-k, --api-key string`: API key for the AI model (required)
- `-t, --template string`: Custom template file (optional)
- `-d, --dry-run`: Preview generated content without creating files
- `-m, --ai-model string`: AI model to use (anthropic, openai, cohere) (default "anthropic")
- `-p, --project-type string`: Project template (cli, web, library) (default "cli")

### Example

```
mkprog -o ./my-project -k your-api-key "Create a CLI tool for managing todo lists"
```

## Configuration

mkprog uses a configuration file to store default values for flags. The configuration file is searched for in the following locations:

1. `$HOME/.config/mkprog/mkprog.yaml`
2. Current directory: `./mkprog.yaml`

You can also use environment variables prefixed with `MKPROG_` to set configuration values.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

