# toolchain-planner

A specialized planning tool for generating designs and specifications for Unix-style command-line tools in the mkprog toolchain.

## Features

- Generates comprehensive specifications for new tools in the mkprog toolchain
- Designs tools following Unix philosophy principles
- Creates detailed implementation plans with modular components
- Suggests ways for tools to compose with existing toolchain members
- Produces structured Markdown output suitable for implementation
- Ensures compatibility with the overall mkprog architecture
- Optimizes for tool composability and pipeline integration
- Considers both user experience and developer experience

## Installation

```bash
go install github.com/tmc/mkprog/tools/toolchain-planner@latest
```

## Usage

```
toolchain-planner [flags] [tool description]
```

Flags:
- `-o, --output <file>`: Output file for the plan (defaults to stdout)
- `-c, --category <category>`: Tool category (generation, analysis, improvement, etc.)
- `-i, --interactive`: Run in interactive mode for plan refinement
- `-e, --examples <number>`: Number of example use cases to generate
- `-x, --existing`: List existing tools that could be referenced in the plan
- `-v, --verbose`: Enable verbose output

## Examples

```bash
# Generate a plan for a simple code complexity analyzer
toolchain-planner "A tool that analyzes Go code complexity metrics"

# Create a plan for a specialized tool with examples
toolchain-planner -e 3 "A tool that generates mock implementations for Go interfaces"

# Generate a plan and save to a file
toolchain-planner -o complexity-tool-plan.md "A tool that measures cyclomatic complexity"

# Interactive planning session for a new code generator
toolchain-planner -i "A tool that creates error handler boilerplate for Go functions"
```

## License

MIT