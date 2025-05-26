# Self-Terminator

Self-Terminator is a Go program that demonstrates controlled self-termination after performing a specific task. It serves as an educational tool to showcase program lifecycle management and Go's concurrency features.

## Features

- Accepts a command-line argument specifying the runtime duration (default: 10 seconds)
- Displays a countdown timer showing the remaining time
- Generates and prints random numbers during its runtime
- Gracefully terminates itself when the specified time has elapsed
- Handles potential errors, such as invalid input for the runtime duration
- Uses Go's time package for precise timing and goroutines for concurrent operations

## Installation

1. Ensure you have Go 1.16 or later installed on your system.
2. Clone this repository:
   ```
   git clone https://github.com/example/self-terminator.git
   ```
3. Change to the project directory:
   ```
   cd self-terminator
   ```

## Usage

Run the program with the following command:

```
go run main.go [-duration <seconds>]
```

- `-duration`: Optional. Specifies the number of seconds for the program to run. Default is 10 seconds.

Example:

```
go run main.go -duration 15
```

This will run the program for 15 seconds.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

