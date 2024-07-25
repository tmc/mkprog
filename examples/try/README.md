# try

`try` is a command-line tool that allows you to experiment with commands in a Git repository without affecting your current working state. It creates a temporary branch, executes the command, captures the output, and stores the result in a Git commit with a note.

## Installation

1. Ensure you have Go installed on your system.
2. Clone this repository:
   ```
   git clone https://github.com/yourusername/try.git
   ```
3. Navigate to the project directory:
   ```
   cd try
   ```
4. Build the program:
   ```
   go build
   ```
5. (Optional) Move the built binary to a directory in your PATH:
   ```
   sudo mv try /usr/local/bin/
   ```

## Usage

```
try <command>
```

Replace `<command>` with the command you want to try.

Example:
```
try echo "Hello, World!"
```

This will:
1. Create a temporary branch
2. Execute the command
3. Capture the output and exit status
4. Create a Git commit with the result
5. Add a Git note with details
6. Switch back to your original branch
7. Delete the temporary branch

The command's output will be displayed, along with a summary of the operation, including the commit hash where the result is stored.

## Error Handling

If any step in the process fails, the program will attempt to return to the original branch and clean up the temporary branch. Error messages will be printed to stderr.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

