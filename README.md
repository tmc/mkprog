# mkprog: AI-Powered Go Programming Toolkit

`mkprog` is a comprehensive toolkit for AI-assisted Go software development, combining the power of large language models with Unix-style composability. It provides a growing collection of specialized tools that work together to generate, improve, analyze, and visualize Go code.

## Features

- **Code Generation**: Create complete, functional Go programs from natural language descriptions
- **Code Analysis**: Visualize dependencies, benchmark performance, and analyze code quality
- **Code Improvement**: Refactor code, fix common issues, and implement improvements
- **Unix Philosophy**: Small, focused tools that compose well in pipelines
- **AI Integration**: Leverages the langchaingo library to interact with advanced AI models
- **Extensible Toolchain**: Growing set of specialized tools for different aspects of Go development

## Core Tools

| Category | Tools |
|----------|-------|
| **Code Generation** | `mkprog`, `better-mkprog`, `mkprogctx` |
| **Code Improvement** | `refactor`, `fixme`, `fixprog`, `improveprog`, `modprog` |
| **Planning & Analysis** | `planprog`, `depvis`, `benchviz`, `token-tree`, `askprog` |
| **Git Integration** | `auto-git-commit`, `git-commit-style` |
| **Meta Tools** | `list-tools`, `try-analyze` |

For a complete list of tools with descriptions, run:
```
tools/list-tools/list-tools
```

## Installation

1. Install Go 1.22 or later
2. Clone this repository:
   ```
   git clone https://github.com/tmc/mkprog.git
   cd mkprog
   ```
3. Build the main program:
   ```
   go build
   ```
4. Build additional tools:
   ```
   for dir in tools/*; do 
     if [ -f "$dir/go.mod" ]; then
       (cd "$dir" && go build)
     fi
   done
   ```

## Using the Main Program

```
./mkprog [flags] <output_directory> <program_description>
```

### Flags

- `-temp <temperature>`: Set the temperature for AI generation (0.0 to 1.0, default 0.1)
- `-embed-prompt`: Embed the generation prompt in a PROMPT.md file (default: true)
- `-embed-metadata`: Embed program metadata in a .mkprog.json file (default: true)
- `-list-tools`: List available tools in the mkprog ecosystem
- `-version`: Show version information

### Self-aware Programs

Generated programs now include self-introspection flags:
- `--show-prompt`: Output the prompt used to generate the program
- `--show-source`: Output the source code of a specific file or all files
- `--mkprog-info`: Show information about the mkprog ecosystem

This makes programs self-documenting and able to share their own source code and creation history.

### Examples

Generate a web server:
```
./mkprog myapp "A web server that provides an API for managing bookmarks"
```

Generate with a higher temperature for more creativity:
```
./mkprog -temp 0.7 creative-app "A creative writing assistant with CLI interface"
```

Generate without embedding the prompt:
```
./mkprog -embed-prompt=false myapp "A simple key-value store with persistence"
```

## Tool Composition Examples

The true power of mkprog comes from combining tools in pipelines:

```bash
# Generate a program and visualize its dependencies
mkprog myapp "A REST API for task management" && depvis myapp

# Generate a program, fix common issues, and automatically commit
mkprog myapp "A file conversion utility" && fixme myapp && auto-git-commit

# Plan a feature implementation, then generate it
planprog "Add authentication to the API" > auth-plan.md
mkprogctx -plan auth-plan.md -o auth/

# Run benchmarks and visualize the results
go test -bench=. ./... > bench.txt
benchviz bench.txt

# Self-introspection examples
# --------------------------

# View available tools in your mkprog ecosystem
mkprog -list-tools

# Output a program's source code for sharing
cd myapp && go build
./myapp --show-source main.go | gist -p

# Share your program's generation prompt
./myapp --show-prompt > PROMPT_USED.md

# Generate a program and see related tools
mkprog data-analyzer "A tool that analyzes CSV files"
cd data-analyzer && go build
./data-analyzer --mkprog-info
```

## Architecture

The mkprog system follows a "Self-Assembling Unix Pipeline Agentic System" architecture:

1. **Core Generator**: The main mkprog tool for generating complete Go programs
2. **Tool Registry**: A collection of specialized tools for different tasks
3. **Composition Patterns**: Conventions for how tools work together
4. **Extension Mechanisms**: Ways to add new capabilities to the system

See [ARCHITECTURE.md](ARCHITECTURE.md) for more details.

## Contributing

Contributions are welcome! To add a new tool to the collection:

1. Create a new directory under `tools/`
2. Implement the tool following the standard pattern
3. Include appropriate documentation in README.md
4. Ensure the tool can operate as part of a Unix pipeline

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.