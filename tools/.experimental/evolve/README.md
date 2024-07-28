# evolve

evolve is a self-improving Go program that can implement changes, test itself, and recursively enhance its own capabilities.

## Features

- Implement changes based on descriptions
- Run tests after implementing changes
- Evaluate and improve changes
- Recursively evolve to perform new tasks
- Use Git to commit changes after successful improvements

## Usage

```
evolve [-test] [-attempts N] [-evaluate] [-improve] [-max-recursive-attempts M] 'description of change'
evolve -- evolve [-max-recursive-attempts M] 'description of task'
```

### Options

- `-test`: Run tests after implementing changes
- `-attempts N`: Number of improvement attempts (default: 1)
- `-evaluate`: Evaluate the changes
- `-improve`: Attempt to improve the changes
- `-max-recursive-attempts M`: Maximum number of recursive self-improvement attempts (default: 10)

## Examples

1. Implement a simple change:
   ```
   evolve 'Add a function to calculate the factorial of a number'
   ```

2. Implement a change with testing and evaluation:
   ```
   evolve -test -evaluate 'Implement a quicksort algorithm'
   ```

3. Recursively evolve to perform a new task:
   ```
   evolve -- evolve 'Implement a web server that serves "Hello, World!"'
   ```

## Installation

1. Ensure you have Go 1.20 or later installed.
2. Clone this repository:
   ```
   git clone https://github.com/yourusername/evolve.git
   ```
3. Change to the project directory:
   ```
   cd evolve
   ```
4. Build the program:
   ```
   go build
   ```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

