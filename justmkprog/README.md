# mkprog

mkprog is a Go program that generates structured content based on user input using the langchaingo library to interact with AI language models.

## Features

- Generates complete, functional Go programs based on user descriptions
- Uses the langchaingo library to interact with AI language models
- Implements error handling and follows Go best practices
- Generates all necessary files for a runnable Go project
- Supports custom system prompts and temperature settings
- Optionally runs goimports on the generated code

## Installation

1. Ensure you have Go 1.22 or later installed on your system.
2. Clone this repository:
   ```
   git clone https://github.com/tmc/mkprog.git
   cd mkprog
   ```
3. Build the program:
   ```
   go build
   ```

## Usage

```
./mkprog [-temp <temperature>] <output_directory> <program_description>
```

- `-temp`: Set the temperature for AI generation (0.0 to 1.0, default: 0.1)
- `<output_directory>`: The directory where the generated program will be created
- `<program_description>`: A description of the program you want to generate

Example:
```
./mkprog -temp 0.2 my_program "A CLI tool that converts markdown to HTML"
```

## Output

The program will generate a complete Go project in the specified output directory, including:

- main.go
- go.mod
- README.md
- LICENSE (MIT)
- Any other necessary files

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.