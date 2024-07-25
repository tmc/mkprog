# Better-mkprog

Better-mkprog is an improved version of the original mkprog program. It generates multiple versions of a program based on a given name and description, runs tests on each version, and selects the best-performing one.

## Features

- Generates multiple program versions
- Runs predefined tests on each version
- Assigns scores based on test results
- Selects the best-performing version
- Provides analysis of test results using AI
- Supports concurrent processing
- Allows custom test definitions via a configuration file

## Installation

1. Ensure you have Go 1.20 or later installed on your system.
2. Clone this repository:
   ```
   git clone https://github.com/yourusername/better-mkprog.git
   ```
3. Change to the project directory:
   ```
   cd better-mkprog
   ```
4. Build the program:
   ```
   go build
   ```

## Usage

Run better-mkprog with the following command-line arguments:

```
./better-mkprog -name <program-name> -desc "<program-description>" [-n <iterations>] [-v] [-config <config-file>]
```

- `-name`: Name of the program to generate (required)
- `-desc`: Description of the program to generate (required)
- `-n`: Number of iterations to run tests (default: 5)
- `-v`: Enable verbose output
- `-config`: Path to the configuration file (default: config.json)

Example:

```
./better-mkprog -name myprogram -desc "A program that sorts numbers" -n 10 -v
```

## Configuration

You can customize the tests run on each program version by modifying the `config.json` file. The file should contain a JSON object with a "tests" array, where each test has a name and a command to run.

Example `config.json`:

```json
{
  "tests": [
    {
      "name": "Compilation",
      "command": "go build -o testprogram testprogram.go"
    },
    {
      "name": "Basic Functionality",
      "command": "./testprogram --test basic"
    },
    {
      "name": "Edge Cases",
      "command": "./testprogram --test edge"
    },
    {
      "name": "Performance",
      "command": "time ./testprogram --test performance"
    }
  ]
}
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

