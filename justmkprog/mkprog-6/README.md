# mkprog

mkprog is a CLI tool that generates a complete Go project structure based on a user-provided description. It uses the Anthropic API (or other selected AI models) to generate code and documentation for the project.

## Features

- Generate a complete Go project structure
- Use AI to create code and documentation
- Support for multiple AI models (Anthropic, OpenAI, Cohere)
- Custom project templates (CLI tool, web server, library)
- Concurrent file generation for improved performance
- Dry-run option to preview generated content
- Configuration file support for default values
- Progress indicator during content generation
- Caching system for previously generated content
- Comprehensive error handling and logging

## Installation

To install mkprog, use the following command:

```
go install github.com/yourusername/mkprog@latest
```

## Usage

```
mkprog [flags] <project description>
```

### Flags

- `-o, --output`: Output directory for the generated project
- `-k, --api-key`: API key for the selected AI model
- `-t, --template`: Custom template file
- `-d, --dry-run`: Preview generated content without creating files
- `-m, --ai-model`: AI model to use (anthropic, openai, cohere)
- `-p, --project-type`: Project template (cli, web, library)
- `-v, --verbose`: Enable verbose output
- `--temperature`: AI model temperature (0.0 - 1.0)
- `--max-tokens`: Maximum number of tokens for AI response

### Example

```
mkprog -o ./my-project -k your-api-key -m anthropic -p cli "Create a CLI tool that converts markdown to HTML"
```

## Configuration

You can create a configuration file named `.mkprog.yaml` in your home directory to set default values for flags. For example:

```yaml
output: ./projects
api-key: your-default-api-key
ai-model: anthropic
project-type: cli
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

