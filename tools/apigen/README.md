# apigen

A tool that generates idiomatic Go API client code from OpenAPI/Swagger specifications.

## Features

- Generate complete Go API client packages from OpenAPI 3.0 and Swagger 2.0 specifications
- Automatically create strongly-typed request and response models with proper JSON annotations
- Include comprehensive error handling with custom error types for different API error scenarios
- Generate client-side rate limiting and retry mechanisms with configurable parameters
- Add authentication handlers for common auth types (API key, OAuth, JWT)
- Create comprehensive godoc documentation from OpenAPI descriptions
- Apply customizable code formatting and style preferences
- Generate usage examples and test code with mocked responses

## Installation

```bash
go install github.com/tmc/mkprog/tools/apigen@latest
```

## Usage

```
apigen [flags]
```

Flags:
- `-s, --spec <file>`: Path to OpenAPI/Swagger specification file (required)
- `-o, --output <dir>`: Output directory for generated client package (required)
- `-p, --package <name>`: Package name for generated code (default: derived from API name)
- `-x, --prefix <prefix>`: Prefix for type names (default: none)
- `-f, --features <list>`: Comma-separated list of features to include (default: all)
- `-t, --timeout <duration>`: Default client timeout (default: 10s)
- `-d, --dry-run`: Print generated code without writing to disk
- `-v, --verbose`: Enable verbose output

## Examples

```bash
# Generate a client for a Petstore API
apigen -spec petstore.yaml -output ./petclient

# Generate with custom options
apigen -spec petstore.yaml -output ./petclient -package petapi -prefix Pet -timeout 30s

# Generate and pipe to improveprog
apigen -spec github-api.json -output ./githubclient | improveprog

# Generate with specific features only
apigen -spec github-api.json -output ./githubclient -features "ratelimit,retry,logging"
```

## License

MIT