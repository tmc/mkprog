# fixme

fixme is a command-line tool that runs a given command, analyzes its output, and suggests actions to add to a .actions file. It uses AI to generate actionable items based on the command output.

## Installation

1. Ensure you have Go 1.20 or later installed on your system.
2. Clone this repository or download the source code.
3. Navigate to the project directory and run:

```
go build
```

This will create an executable named `fixme` in the current directory.

## Usage

To use fixme, run it with the following syntax:

```
./fixme -- <command> [args...]
```

For example:

```
./fixme -- go build ./...
```

This will run the `go build ./...` command, analyze its output, and add suggested actions to the `.actions` file in the current directory.

## Configuration

Make sure to set the `ANTHROPIC_API_KEY` environment variable with your Anthropic API key before running the program.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

