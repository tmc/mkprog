# fixprog

fixprog is a tool that automatically fixes programming issues in a directory tree of source code based on a given description of the change. It can optionally run a test command to verify if the problem has been resolved.

## Installation

1. Ensure you have Go 1.20 or later installed on your system.
2. Clone this repository or download the source code.
3. Run `go build` in the project directory to compile the program.

## Usage

```
./fixprog -dir <source_directory> -desc "<change_description>" [-test "<test_command>"]
```

- `-dir`: Directory containing the source code (default: current directory)
- `-desc`: Description of the change to be made (required)
- `-test`: Command to run to check if the problem is fixed (optional)

## Example

```
./fixprog -dir ./myproject -desc "Fix the off-by-one error in the loop" -test "go test ./..."
```

This command will attempt to fix the off-by-one error in the loop within the ./myproject directory and run the test suite to verify the fix.

## Features

- Analyzes source code in various programming languages (Go, Python, JavaScript, Java, C, C++)
- Uses AI to suggest and apply changes based on the provided description
- Optionally runs a test command to verify the fix
- Supports Git repositories for reverting changes if needed
- Implements a retry mechanism with a maximum number of attempts

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

