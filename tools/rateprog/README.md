# rateprog

rateprog is a Go program that evaluates a given program based on repo-wide expectations. It compares a given program in a directory to a set of default rules plus user-supplied overrides in a `.rateprog-rules` file found in the current or any parent directory in the current repository.

## Installation

1. Ensure you have Go 1.20 or later installed on your system.
2. Clone this repository or download the source code.
3. Navigate to the project directory and run:

```
go build
```

This will create an executable named `rateprog` in the current directory.

## Usage

To evaluate a program, run:

```
./rateprog -program /path/to/your/program.go
```

The program will look for a `.rateprog-rules` file in the current directory or any parent directory. If found, it will use the rules specified in that file along with the default rules. If not found, it will use only the default rules.

## Custom Rules

To specify custom rules, create a `.rateprog-rules` file in your repository. Each rule should be on a new line. For example:

```
Use dependency injection for better testability
Implement proper logging
Ensure all exported functions are documented
```

These rules will be combined with the default rules when evaluating the program.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

