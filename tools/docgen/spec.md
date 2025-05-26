# Tool Specification

- **Generated:** 2025-03-02 05:05:05
- **Description:** A tool that generates Go code documentation with AI assistance

## 1. Tool Overview
- Name: docgen
- One-sentence description: A tool that analyzes Go code and generates comprehensive documentation with AI assistance.
- Primary category: Planning & Analysis
- Core functionality and purpose: Scan Go code to extract types, functions, and interfaces, then use AI to generate high-quality documentation that explains the code's purpose, usage patterns, and design decisions.
- Key differentiators: Unlike traditional documentation generators, docgen uses AI to create meaningful explanations that focus on the "why" and "how" rather than just describing what the code does. It adds value by explaining design patterns, suggesting usage examples, and highlighting relationships between components.

## 2. Core Features
- Scan Go code to identify types, interfaces, functions, and methods that need documentation
- Generate or enhance godoc-compatible documentation for the code
- Provide usage examples for complex functions and types
- Explain design patterns and architectural decisions embedded in the code
- Generate markdown documentation for project READMEs and wikis
- Detect relationships between components and explain their interactions
- Support both interactive and batch documentation modes
- Integrate with existing documentation tools and workflows

## 3. Implementation Plan
- Core components:
  - Go code parser using go/ast and go/parser packages
  - Documentation generator with customizable templates
  - AI content generator using langchaingo
  - Documentation formatter for different output formats

- Key functions/methods:
  - `ParsePackage(path string)`: Parse a Go package to extract code structure
  - `GenerateDocumentation(parsedCode *ast.Package)`: Generate documentation for a parsed package
  - `EnhanceWithAI(docs *Documentation)`: Enhance documentation using AI
  - `FormatOutput(docs *Documentation, format string)`: Format documentation for different outputs

- Data structures:
  - `CodeElement`: Represents a code element (function, type, interface)
  - `Documentation`: Contains documentation for a package
  - `DocOptions`: Configuration options for documentation generation

- External dependencies:
  - Standard Go AST packages (go/ast, go/parser, go/token)
  - github.com/tmc/langchaingo for AI integration
  - go/doc for existing documentation processing

- Error handling approach:
  - Clear error messages for parsing failures
  - Graceful degradation when AI enhancement fails
  - User-friendly reporting of documentation gaps

## 4. Integration Points
- Works with existing documentation tools like godoc and pkgsite
- Can be used in CI/CD pipelines to ensure documentation coverage
- Pairs well with `fixme` to identify and fix documentation issues
- Outputs can be used by `mkprog` when generating new code
- Use with `list-tools` to generate documentation for all tools in the toolchain

Input/output specs:
- Input: Go source code files or directories
- Output: Enhanced documentation in various formats (godoc comments, markdown files, HTML)

## 5. Example Use Cases

Example 1: Generate package documentation
```bash
# Generate enhanced godoc comments for a package
docgen -package ./mypackage

# Generate documentation and update source files in place
docgen -package ./mypackage -write
```

Example 2: Generate project documentation
```bash
# Create a markdown documentation file for the entire project
docgen -project ./myproject -format markdown -output README.md

# Generate HTML documentation for a project and its subpackages
docgen -project ./myproject -format html -output docs/ -recursive
```

## 6. Project Structure
- `main.go`: Entry point with command-line handling
- `parser/`: Package for parsing Go source code
  - `parser.go`: Go code parser
  - `extractor.go`: Documentation extractor
- `generator/`: Documentation generation package
  - `generator.go`: Documentation generator
  - `enhancer.go`: AI enhancement for documentation
  - `formatter.go`: Output formatter
- `templates/`: Template files for different output formats
  - `godoc.tmpl`: Template for Go doc comments
  - `markdown.tmpl`: Template for markdown output
  - `html.tmpl`: Template for HTML output