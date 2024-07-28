# mkusage

mkusage is a Go utility that generates and prints the contents of a USAGE file for Go programs. This tool helps developers create standardized documentation for their Go applications by extracting and formatting program information, excluding flag details.

## Features

- Analyzes Go source code to extract program information
- Generates a USAGE file with program details, excluding flags
- Prints the contents of the USAGE file to stdout

## Installation

To install mkusage, make sure you have Go 1.16 or later installed, then run:

```
go install github.com/yourusername/mkusage@latest
```

## Usage

Run mkusage from the command line, providing the path to the Go program as an argument:

```
mkusage /path/to/your/go/program
```

The tool will output the USAGE content to stdout, which can be redirected to a file if needed:

```
mkusage /path/to/your/go/program > USAGE
```

## Requirements

- Go 1.16 or later
- Compatible with Go modules and GOPATH projects

## Error Handling

mkusage gracefully handles missing or invalid Go source files and provides clear error messages for parsing failures. It also supports programs with multiple packages or entry points.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

