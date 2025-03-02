# mkprog Tools Collection

This directory contains a suite of specialized tools that work together to form a self-assembling Unix-style pipeline for code generation, analysis, and improvement.

## Tool Organization

The tools are designed to work independently but compose well together, following Unix philosophy:

### Code Generation Tools
- **mkprog**: Generate Go programs from natural language descriptions
- **better-mkprog**: Enhanced code generation with more advanced patterns
- **mkprogctx**: Context-aware code generation 

### Code Improvement Tools
- **refactor**: AI-assisted Go code refactoring with pattern recognition
- **fixme**: Fix common issues in Go code
- **fixprog**: Comprehensive codebase issue resolution
- **improveprog**: General code improvement suggestions
- **try**: Try various code improvements and select the best one
- **modprog**: Modify code based on specific requirements

### Planning and Analysis Tools
- **planprog**: Create detailed implementation plans for complex features
- **askprog**: Ask questions about code and get detailed answers
- **attempt-analyzer**: Analyze code attempt effectiveness
- **depvis**: Go dependency visualization and analysis
- **token-tree**: Visualize token usage across a codebase
- **benchviz**: Parse and visualize Go benchmark results

### Git Integration Tools
- **git-commit-style**: Analysis and standardization of commit messages
- **auto-git-commit**: Automatic commit message generation
- **record-result**: Record and track code evolution results

### Meta-Tools
- **list-tools**: List all available tools with descriptions
- **try-analyze**: Analyze which approaches work best

## Usage Patterns

The tools can be combined in various ways:

```bash
# Generate a program and improve it
mkprog new-app "HTTP API to track bookmarks" | improveprog > improved-app/

# Visualize dependencies in an existing project
go test -bench=. ./... > benchmarks.txt
benchviz benchmarks.txt

# Identify and fix issues
fixme ./src/ | auto-git-commit

# Plan and implement a new feature
planprog "Add authentication to the API" > plan.md
mkprogctx -plan plan.md -o auth/
```

## Development

To add a new tool to the collection:

1. Create a new directory under `tools/`
2. Implement the tool following the standard pattern (see existing tools)
3. Include appropriate documentation in README.md
4. Ensure the tool can operate as part of a Unix pipeline

## License

All tools are available under the MIT License.