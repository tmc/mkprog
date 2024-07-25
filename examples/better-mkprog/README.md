# Better MkProg

Better MkProg is an improved version of the original mkprog program. It generates Go programs based on user input, runs multiple iterations of tests, and provides analysis of the results.

## Features

- Generates Go programs based on user-provided name and description
- Runs multiple iterations of program generation and testing
- Compiles and tests each generated program
- Saves test results as Git notes
- Assigns scores to generated programs based on test results
- Compares scores of all generated versions
- Presents the best-performing version to the user
- Provides a summary of all versions and their scores
- Uses concurrent processing for running iterations
- Offers a verbose output option for detailed logging
- Analyzes test results using AI to provide insights

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
   go build -o better-mkprog
   ```

## Usage

Run the program with the following command-line arguments:

```
./better-mkprog -name <program-name> -desc "<program-description>" [-n <iterations>] [-v]
```

- `-name`: Name of the program to generate (required)
- `-desc`: Description of the program to generate (required)
- `-n`: Number of iterations to run tests (default: 5)
- `-v`: Enable verbose output

Example:

```
./better-mkprog -name my-awesome-program -desc "A program that does amazing things" -n 10 -v
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

