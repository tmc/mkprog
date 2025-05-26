# mkprog-quine

A quine-like tool that generates the source code of the original mkprog program.

## What's a quine?

In computing, a quine is a computer program which takes no input and produces a copy of its own source code as its only output. For this tool, we extend the concept to generate the source code of another program - specifically, the original mkprog program.

## Installation

```
go install github.com/tmc/mkprog/tools/mkprog-quine@latest
```

Or build from source:

```
git clone https://github.com/tmc/mkprog.git
cd mkprog/tools/mkprog-quine
go install
```

## Usage

```
mkprog-quine [flags]
```

### Flags

- `-o <directory>`: Specify output directory for the generated mkprog program (default: "mkprog-generated")
- `-temp <float>`: Set the temperature for AI generation (0.0 to 1.0, default 0.1)
- `-verbose`: Enable verbose output

### Example

Generate the original mkprog program in a custom directory:

```
mkprog-quine -o my-mkprog -verbose
```

After generation, you can build and use the recreated mkprog program:

```
cd my-mkprog
go mod tidy
go build
./mkprog my-new-tool "a CLI tool that does X"
```

## How It Works

mkprog-quine uses an AI model to recreate the original source code of the mkprog program. The AI is given specific instructions to generate all necessary files with their exact contents, including:

- main.go
- system-prompt.txt
- go.mod
- README.md
- LICENSE

This creates a fully functional recreation of the original mkprog program.

## License

MIT