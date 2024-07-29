# list-tools

list-tools is a Go program that lists and describes available tools, including standard Unix/Linux utilities and custom tools defined in the current toolchain.

## Features

- Scans the current repo for defined programs and scripts
- Scans PATH entries that are within the user's home directory
- Provides a simple command-line interface with the following features:
  - Lists all tools with brief descriptions when run without arguments
  - Accepts a search term to filter tools by name or description
  - Displays detailed information about a specific tool when its name is provided as an argument
- Shows the following information for each tool:
  - Name
  - Brief description (1-2 sentences)
  - Location (full path)
  - Type (standard system tool or custom toolchain tool)
  - Flags and if stdin is used/supported
- Implements a simple caching mechanism to improve performance for repeated queries
- Handles errors gracefully
- Uses Go's built-in 'flag' package for parsing command-line arguments
- Utilizes Go's concurrency features (goroutines) to speed up the scanning process
- Implements a config file (in JSON) to allow users to specify additional directories to scan for tools

## Installation

1. Ensure you have Go installed on your system (version 1.20 or later).
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

```
$ ./list-tools
$ ./list-tools search <term>
$ ./list-tools info <tool-name>
```

## Configuration

You can add additional directories to scan by editing the `config.json` file. Add the full paths of the directories you want to include in the `additional_dirs` array.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

