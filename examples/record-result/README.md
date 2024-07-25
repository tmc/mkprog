# Record Result

Record Result is a Go program that runs a given command, creates a git commit with the command that generated it, and saves it in the git metadata/note.

## Installation

1. Ensure you have Go installed on your system.
2. Clone this repository:
   ```
   git clone https://github.com/yourusername/record-result.git
   ```
3. Change to the project directory:
   ```
   cd record-result
   ```
4. Build the program:
   ```
   go build
   ```

## Usage

Run the program with the following syntax:

```
./record-result -- <command> [args...]
```

The program will:
1. Run the specified command
2. Create a git commit with all changes
3. Add a git note with the command used

Make sure you're in a git repository when running the program.

Example:
```
./record-result -- echo "Hello, World!"
```

This will run the `echo "Hello, World!"` command, create a git commit with any changes, and add a git note with the command.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

