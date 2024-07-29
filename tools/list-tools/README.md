# list-tools

list-tools is a Go program that lists and describes available tools, including standard Unix/Linux utilities and custom tools defined in the current toolchain.

## Features

- Lists tools relevant to the current toolchain (the current repository)
- Secure: does not run arbitrary binaries
- Fast: caches the results of the search
- Flexible: allows users to specify additional directories to search for tools
- User-friendly: provides a simple command-line interface
- Concurrent: uses goroutines to speed up the scanning process
- Configurable: allows users to specify additional directories to scan for tools via a config file

## Installation

1. Ensure you have Go 1.16 or later installed on your system.
2. Clone this repository:
   ```
   git clone https://github.com/yourusername/list-tools.git
   ```
3. Change to the project directory:
   ```
   cd list-tools
   ```
4. Build the program:
   ```
   go build
   ```

## Usage

Run the program without arguments to list all tools:
```
./list-tools
```

Search for tools by name or description:
```
./list-tools -search <term>
```

Display detailed information about a specific tool:
```
./list-tools -info <tool-name>
```

## Configuration

You can specify additional directories to scan for tools by creating a JSON configuration file at `~/.list-tools.json`:

```json
{
  "additionalDirectories": [
    "/path/to/custom/tools",
    "/another/path/to/tools"
  ]
}
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

