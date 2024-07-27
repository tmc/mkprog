# Attempt

Attempt is a Go program that takes a goal and a set of tools as input, and attempts to achieve the goal using the provided tools. It uses git-based isolation and state files to store multiple attempts and their outputs for analysis.

## Installation

1. Ensure you have Go 1.20 or later installed on your system.
2. Clone this repository or download the source code.
3. Run `go mod tidy` to download and install the required dependencies.

## Usage

```
go run main.go <goal> <tools> <num_attempts>
```

- `<goal>`: The objective you want to achieve.
- `<tools>`: A comma-separated list of tools available for use.
- `<num_attempts>`: The number of attempts to make.

Example:

```
go run main.go "Build a simple web server" "Go,HTTP package,text editor" 3
```

This command will make 3 attempts to build a simple web server using Go, the HTTP package, and a text editor.

## Output

The program creates a temporary directory to store the attempts. Each attempt is saved as a JSON file and committed to a local git repository. The output includes:

1. JSON files for each attempt, containing the goal, tools, and AI-generated output.
2. A git repository with commits for each attempt.

The location of the output directory is printed at the end of the execution.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

