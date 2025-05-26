# mkprog-mkprog

A tool that generates system prompts for mkprog. This meta-tool helps you create specialized system prompts that can be used with the mkprog tool to generate Go programs with specific functionality.

## Installation

```
go install github.com/tmc/mkprog/tools/mkprog-mkprog@latest
```

Or build from source:

```
git clone https://github.com/tmc/mkprog.git
cd mkprog/tools/mkprog-mkprog
go install
```

## Usage

```
mkprog-mkprog [flags] <description of the mkprog tool to create>
```

### Flags

- `-o <file>`: Specify output file for the generated system prompt (use `-` for stdout, default)
- `-temp <float>`: Set the temperature for AI generation (0.0 to 1.0, default 0.1)
- `-verbose`: Enable verbose output

### Examples

Generate a system prompt for a tool that creates test files:

```
mkprog-mkprog "a tool that generates test files for Go functions" > system-prompt.txt
```

Generate a system prompt and save it to a file:

```
mkprog-mkprog -o my-prompt.txt "a CLI tool that analyzes Go dependencies"
```

Use the generated system prompt with mkprog:

```
mkprog -system my-prompt.txt my-new-tool "description of the tool"
```

## How It Works

mkprog-mkprog uses an AI model to generate specialized system prompts for various types of Go programs. These system prompts can then be used with the mkprog tool to generate complete, functional Go programs.

The generated system prompts include detailed instructions for:
- Program structure and organization
- File requirements
- Code organization
- AI interaction patterns
- Program logic implementation
- Output formatting

By creating specialized system prompts, you can guide mkprog to generate more specific and tailored Go programs.

## License

MIT