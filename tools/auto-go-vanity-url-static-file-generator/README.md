# Auto Go Vanity URL Static File Generator

This tool automatically generates static HTML files to support vanity URLs for Go submodules. When provided with a Go source tree and an output directory, it identifies all submodules and creates the necessary HTML files to enable vanity URL support.

## Features

1. Recursively scan a Go source tree to identify all submodules
2. Generate appropriate HTML files for each submodule to support vanity URLs
3. Output generated files to a specified directory
4. Configurable base URL for vanity imports
5. Option to overwrite existing files or skip them
6. Verbose mode for detailed output during the generation process

## Installation

To install the tool, make sure you have Go 1.16 or later installed, then run:

```
go install github.com/example/auto-go-vanity-url-static-file-generator@latest
```

## Usage

```
auto-go-vanity-url-static-file-generator [flags]
```

### Flags

- `-s, --source string`: Path to the Go source tree (required)
- `-o, --output string`: Output directory for generated files (required)
- `-b, --base-url string`: Base URL for vanity imports (e.g., 'example.com/repo') (required)
- `-w, --overwrite`: Overwrite existing files
- `-v, --verbose`: Verbose mode

### Example

```
auto-go-vanity-url-static-file-generator -s /path/to/go/source -o /path/to/output -b example.com/repo -v
```

This command will scan the Go source tree at `/path/to/go/source`, generate static HTML files for each submodule, and output them to `/path/to/output`. The vanity URLs will use `example.com/repo` as the base URL. The `-v` flag enables verbose output.

## Error Handling

The tool handles various error scenarios:

- Permission issues when reading source files or writing output files
- Invalid input paths or URLs
- Cases where the output directory already contains files

Clear error messages are provided for each scenario.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

