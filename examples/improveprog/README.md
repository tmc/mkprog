# improveprog

improveprog is a command-line tool that improves an existing Go program based on a given description of changes. It uses AI to analyze the current program and implement the requested modifications while maintaining the program's overall structure and functionality.

## Features

- Improves Go programs based on text descriptions
- Git-aware: only operates on clean working directories
- Optional validation and test commands
- Automatic rollback if validation or tests fail

## Installation

1. Ensure you have Go 1.20 or later installed on your system.
2. Clone this repository:
   ```
   git clone https://github.com/yourusername/improveprog.git
   ```
3. Change to the project directory:
   ```
   cd improveprog
   ```
4. Build the program:
   ```
   go build
   ```

## Usage

```
./improveprog [flags] <file> <change description>
```

Flags:
- `-validate string`: Command to validate the changes
- `-test string`: Command to test the changes

Example:
```
./improveprog -validate "go vet" -test "go test ./..." main.go "Add error handling to the processData function"
```

This command will improve the `main.go` file by adding error handling to the `processData` function. It will then run `go vet` to validate the changes and `go test ./...` to run the tests. If either the validation or tests fail, the changes will be rolled back.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

