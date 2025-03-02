# Tool Specification

- **Generated:** 2025-03-02 05:05:05
- **Description:** A tool that generates Go API clients from OpenAPI specifications

## 1. Tool Overview
- Name: apigen
- One-sentence description: A tool that generates idiomatic Go API client code from OpenAPI/Swagger specifications.
- Primary category: Code Generation
- Core functionality and purpose: Convert OpenAPI specifications into complete, idiomatic Go client libraries with proper error handling, authentication, rate limiting, and documentation.
- Key differentiators: Unlike generic OpenAPI generators, apigen produces highly idiomatic Go code that follows best practices, uses standard Go patterns, and integrates seamlessly with the Go ecosystem.

## 2. Core Features
- Generate complete Go API client packages from OpenAPI 3.0 and Swagger 2.0 specifications
- Automatically create strongly-typed request and response models with proper JSON annotations
- Include comprehensive error handling with custom error types for different API error scenarios
- Generate client-side rate limiting and retry mechanisms with configurable parameters
- Add authentication handlers for common auth types (API key, OAuth, JWT)
- Create comprehensive godoc documentation from OpenAPI descriptions
- Apply customizable code formatting and style preferences
- Generate usage examples and test code with mocked responses

## 3. Implementation Plan
- Core components:
  - OpenAPI parser that handles both 3.0 and 2.0 formats
  - Go code generator with templating for different API components
  - Type converter for mapping OpenAPI types to Go types
  - Documentation generator for godoc
  - Test/example code generator

- Key functions/methods:
  - `ParseSpecification(specPath string)`: Parse OpenAPI spec file
  - `GenerateClient(spec *Specification, options Options)`: Generate client code
  - `GenerateTypes(spec *Specification)`: Generate request/response structs
  - `GenerateOperations(spec *Specification)`: Generate API operation methods
  - `GenerateAuthentication(spec *Specification)`: Generate auth components
  - `GenerateTests(spec *Specification)`: Generate test code

- Data structures:
  - `Specification`: Parsed OpenAPI specification
  - `Operation`: API operation with path, method, params, responses
  - `TypeDefinition`: Model object definition
  - `Options`: Code generation options and customizations

- External dependencies:
  - `github.com/getkin/kin-openapi`: For parsing OpenAPI specifications
  - `github.com/dave/jennifer`: For programmatic Go code generation
  - Standard Go template library for code templates
  - `golang.org/x/tools`: For AST manipulation and formatting

- Error handling approach:
  - Custom error types for different failure scenarios
  - Detailed error messages with context
  - Option to generate retry logic for transient errors

## 4. Integration Points
- Integration with the main `mkprog` tool to generate complete client packages
- Output can be piped to `improveprog` for further code enhancements
- Can be used in pipeline with `refactor` to customize generated code
- Works as input to `benchviz` for API client performance analysis
- Use with `fixme` to clean up and standardize existing API clients
- Use `list-tools` to discover related API tools in the toolchain

Input/output specs:
- Input: OpenAPI spec file (JSON or YAML)
- Output: Complete Go package with client code

## 5. Example Use Cases

Example 1: Generate a basic API client
```bash
# Generate a client for a Petstore API
apigen -spec petstore.yaml -output ./petclient

# Generate with custom options
apigen -spec petstore.yaml -output ./petclient -package petapi -prefix Pet -timeout 30s
```

Example 2: Generate and improve an API client
```bash
# Generate and then improve the code
apigen -spec github-api.json -output ./githubclient | improveprog

# Generate and customize with specific features
apigen -spec github-api.json -output ./githubclient -features "ratelimit,retry,logging"
```

## 6. Project Structure
- `main.go`: Entry point with command-line handling
- `openapi/`: Package for parsing OpenAPI specs
  - `parser.go`: OpenAPI spec parser
  - `converter.go`: Type converter for OpenAPI to Go
- `generator/`: Code generation package
  - `client.go`: Client code generator
  - `types.go`: Request/response type generator
  - `operations.go`: API operations generator
  - `auth.go`: Authentication code generator
  - `templates/`: Go templates for code generation
- `templates/`: Template files for generated code
  - `client.tmpl`: Base client template
  - `model.tmpl`: Model struct template
  - `operation.tmpl`: API method template
  - `auth.tmpl`: Authentication template
  - `examples.tmpl`: Example usage template