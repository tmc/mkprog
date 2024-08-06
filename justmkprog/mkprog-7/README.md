# mkprog

mkprog is a Go program that generates structured content based on user input using the langchaingo library to interact with AI language models. It accepts a program name and description as input and generates a complete Go project including main.go, go.mod, README.md, and other necessary files.

## Features

- Uses the Anthropic language model via langchaingo library
- Implements error handling and follows Go best practices
- Supports custom temperature settings for AI generation
- Optional goimports execution on generated Go files
- Verbose logging option
- Flexible input options (stdin or file)
- Customizable output directory

## Installation

1. Ensure you have Go 1.21 or later installed on your system.
2. Clone this repository:
   ```
   git clone https://github.com/yourusername/mkprog.git
   ```
3. Change to the project directory:
   ```
   cd mkprog
   ```
4. Build the program:
   ```
   go build
   ```

## Usage

```
./mkprog [flags] "program-name program-description"
```

### Flags

- `-temperature float`: Temperature for AI generation (default 0.1)
- `-max-tokens int`: Maximum number of tokens for AI generation (default 8192)
- `-verbose`: Enable verbose logging
- `-f string`: Input file (use '-' for stdin) (default "-")
- `-o string`: Output directory for generated files (default "generated_program")
- `-goimports`: Run goimports on generated Go files

### Example

```
./mkprog -temperature 0.2 -max-tokens 4000 -o my_new_program "hello_world A simple Hello World program in Go"
```

This command will generate a new Go project for a Hello World program in the `my_new_program` directory.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

