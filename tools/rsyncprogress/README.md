# rsyncprogress

rsyncprogress is a Go program that processes rsync output and displays a tasteful progress bar in the terminal. It reads rsync output from stdin, parses relevant information such as file transfer progress and overall sync status, and renders a visually appealing progress bar.

## Features

- Real-time parsing of rsync output
- Calculation of overall progress percentage
- Display of current file being transferred
- Estimation of remaining time
- Transfer speed indicator
- Customizable progress bar appearance

## Installation

1. Ensure you have Go 1.16 or later installed on your system.
2. Clone this repository:
   ```
   git clone https://github.com/yourusername/rsyncprogress.git
   ```
3. Change to the project directory:
   ```
   cd rsyncprogress
   ```
4. Build the program:
   ```
   go build
   ```

## Usage

To use rsyncprogress, pipe the output of rsync into the program:

```
rsync [your rsync options] | ./rsyncprogress
```

### Command-line flags

- `-v`: Enable verbose output
- `-char string`: Custom character for progress bar (default "█")
- `-color string`: Color of the progress bar (black, red, green, yellow, blue, magenta, cyan, white) (default "cyan")
- `-refresh int`: Refresh rate in milliseconds (default 100)
- `-width int`: Width of the progress bar (default 50)

Example with custom options:

```
rsync [your rsync options] | ./rsyncprogress -char "#" -color green -width 60 -refresh 200
```

## Requirements

- Go 1.16 or later
- github.com/nsf/termbox-go
- github.com/mattn/go-runewidth

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

