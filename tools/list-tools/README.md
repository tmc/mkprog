# list-tools

list-tools is a Go program that lists and describes available tools, including standard Unix/Linux utilities and custom tools defined in the current toolchain.

## Features

- Lists standard system tools from PATH
- Searches for custom tools in ~/.toolchain/
- Supports listing tools defined in the current repository
- Provides a simple command-line interface for listing, searching, and displaying tool information
- Implements caching for improved performance
- Supports concurrent scanning of directories
- Allows configuration of additional directories to scan via a config file

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

- List all tools:
  ```
  ./list-tools
  ```

- Search for tools by name or description:
  ```
  ./list-tools search -term <search-term>
  ```

- Display detailed information about a specific tool:
  ```
  ./list-tools info -tool <tool-name>
  ```

## Configuration

You can specify additional directories to scan for tools by creating a `~/.list-tools.json` file with the following structure:

```json
{
  "additionalDirectories": [
    "/path/to/additional/directory1",
    "/path/to/additional/directory2"
  ]
}
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

