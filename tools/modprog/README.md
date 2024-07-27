# modprog

modprog is a Go program that modifies an existing Go program based on a high-level description of the desired modification. It uses the langchaingo library to generate modification suggestions and applies those changes to the code.

## Installation

1. Ensure you have Go 1.20 or later installed on your system.
2. Clone this repository or download the source code.
3. Run `go mod tidy` to download the required dependencies.

## Usage

```
modprog <path_to_program> <modification_description>
```

- `<path_to_program>`: The path to the existing Go program you want to modify.
- `<modification_description>`: A string describing the desired modification.

The program will analyze the existing code, generate modification suggestions, and show a diff of the proposed changes. You can then choose to apply the changes or cancel the operation.

## Example

```
modprog ./myprogram.go "Add error handling to the main function"
```

This command will modify the `myprogram.go` file by adding error handling to its main function.

## Features

- Analyzes existing Go code
- Generates modification suggestions using AI
- Shows a diff of proposed changes
- Allows user confirmation before applying changes
- Handles errors and provides logging

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

