# mkprog

mkprog is a CLI tool that generates a complete Go project structure based on a user-provided description. It uses AI models to generate code and documentation for the project.

## Features

- Generate a complete Go project structure
- Use AI models (Anthropic, OpenAI, Cohere) to generate code and documentation
- Support for custom templates
- Dry-run option to preview generated content
- Concurrent file writing for improved performance
- Configuration file system for default values
- Progress indicator during content generation
- Caching system for previously generated content
- Comprehensive unit tests
- Ability to update existing Go projects

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

- `-d, --description string`: Project description
- `-o, --output string`: Output directory
- `-k, --api-key string`: API key for AI model
- `-t, --template string`: Custom template file
- `--dry-run`: Dry run (preview generated content)
- `-m, --ai-model string`: AI model to use (anthropic, openai, cohere) (default "anthropic")
- `-p, --project-type string`: Project template (cli, web, library) (default "cli")
- `-v, --verbose`: Verbose output
- `--temperature float`: AI model temperature (default 0.1)
- `--max-tokens int`: Maximum number of tokens for AI response (default 8192)
- `--config string`: Config file (default is $HOME/.mkprog.yaml)

## Configuration

mkprog uses a configuration file to store default values for flags. The default configuration file is located at `$HOME/.mkprog.yaml`. You can specify a custom configuration file using the `--config` flag.

Example configuration file:

```yaml
api-key: your-api-key-here
ai-model: anthropic
project-type: cli
temperature: 0.1
max-tokens: 8192
```

## Examples

Generate a CLI project:

```
mkprog -d "A CLI tool for managing tasks" -o ./my-cli-project -k your-api-key
```

Generate a web server project with a custom template:

```
mkprog -d "A RESTful API server for user management" -o ./my-web-project -k your-api-key -p web -t ./custom-template.json
```

Preview generated content without creating files:

```
mkprog -d "A library for data processing" -o ./my-library -k your-api-key -p library --dry-run
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

