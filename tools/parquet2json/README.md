# parquet2json

parquet2json is a command-line tool that converts Parquet files to JSON format. The program takes a Parquet file as input and generates a corresponding JSON file as output.

## Features

- Support for reading Parquet files of various sizes and schemas
- Efficient conversion process optimized for large datasets
- Option to pretty-print the JSON output for better readability
- Ability to handle nested and complex Parquet schemas
- Support for specifying custom output file names and locations
- Error handling for invalid Parquet files or conversion issues

## Installation

1. Ensure you have Go installed on your system (version 1.16 or later).
2. Clone this repository or download the source code.
3. Navigate to the project directory and run:

```
go build
```

This will create an executable named `parquet2json` in the current directory.

## Usage

```
parquet2json [options] <input_parquet_file> [output_json_file]
```

Options:
- `--pretty`: Enable pretty-printing of the JSON output
- `--max-rows`: Specify the maximum number of rows to convert (default: all)

If no output file is specified, the program will create a JSON file with the same name as the input file, but with a `.json` extension.

## Examples

1. Convert a Parquet file to JSON:
```
./parquet2json input.parquet
```

2. Convert a Parquet file to pretty-printed JSON:
```
./parquet2json --pretty input.parquet output.json
```

3. Convert the first 1000 rows of a Parquet file to JSON:
```
./parquet2json --max-rows 1000 input.parquet
```

## Error Handling

The program provides clear error messages for various scenarios, such as file not found, invalid Parquet format, or insufficient permissions. If an error occurs, the program will exit with a non-zero status code and display an error message on stderr.

## Dependencies

- github.com/xitongsys/parquet-go: For Parquet file parsing
- encoding/json: For JSON encoding (built-in Go package)

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

