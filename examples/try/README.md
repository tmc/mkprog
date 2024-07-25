# try

`try` is a command-line tool that allows developers to safely experiment with changes in a Git repository. It creates a temporary branch, executes a given command, commits the changes, and optionally deletes the temporary branch.

## Installation

To install `try`, make sure you have Go installed on your system, then run:

```
go install github.com/yourusername/try@latest
```

## Usage

```
try <command>
```

### Flags

- `-v, --verbose`: Provide more detailed output
- `-k, --keep`: Keep the temporary branch instead of deleting it
- `-h, --help`: Display help information

## Examples

1. Try a simple command:
   ```
   try echo "Hello, World!"
   ```

2. Run a script with arguments:
   ```
   try ./myscript.sh arg1 arg2
   ```

3. Keep the temporary branch and use verbose output:
   ```
   try -v -k npm test
   ```

## How it works

1. Saves the current branch name
2. Creates a new temporary branch with a unique name (e.g., 'try-<timestamp>')
3. Switches to the new branch
4. Executes the given command
5. Captures the command's output and exit status
6. Creates a Git commit with a message indicating the attempted command
7. Adds a Git note to the commit with the command's output and exit status
8. Switches back to the original branch
9. Deletes the temporary branch (unless `-k` flag is used)

## Error Handling

If any step fails, `try` will attempt to return to the original branch and provide informative error messages.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

