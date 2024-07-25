# parallelprog

parallelprog is a Go program that takes a plan, starts Docker containers, and attempts improvements up to a specified number of times in parallel. It then compares the results and provides a conclusion using AI analysis.

## Features

- Parallel execution of improvement attempts using Docker containers
- Configurable number of attempts and parallelism
- AI-powered analysis of results using the Anthropic language model

## Installation

1. Ensure you have Go 1.20 or later installed on your system.
2. Clone this repository or download the source code.
3. Navigate to the project directory and run:

```
go build
```

This will create an executable named `parallelprog` in the current directory.

## Usage

To run parallelprog, use the following command:

```
./parallelprog -plan <path_to_plan_file> [-attempts <max_attempts>] [-parallel <num_parallel>]
```

Options:
- `-plan`: Path to the plan file (required)
- `-attempts`: Maximum number of improvement attempts (default: 10)
- `-parallel`: Number of parallel executions (default: 3)

Example:
```
./parallelprog -plan my_plan.txt -attempts 5 -parallel 2
```

This command will execute the plan specified in `my_plan.txt` up to 5 times with 2 parallel workers.

## Requirements

- Docker installed and running on your system
- Anthropic API key set in the `ANTHROPIC_API_KEY` environment variable

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
