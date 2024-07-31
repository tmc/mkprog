# askprog

askprog is a simple command-line tool that allows you to ask questions about a codebase. It collects the source code of the current directory and uses it as context to answer your questions.

## Installation

1. Make sure you have Go installed on your system.
2. Clone this repository or download the source code.
3. Navigate to the project directory and run:

```
go build
```

This will create an executable named `askprog` in the current directory.

## Usage

To use askprog, simply run the executable followed by your question:

```
./askprog "What does the main function do?"
```

The program will collect the source code from the current directory and use it as context to answer your question.

## Supported File Types

askprog currently supports the following file extensions:
- .go
- .js
- .py
- .java
- .c
- .cpp
- .h

## Note

This tool requires an Anthropic API key to be set in your environment variables. Make sure to set the `ANTHROPIC_API_KEY` environment variable before running the program.

## License

This project is licensed under the MIT License. See the LICENSE file for details.

