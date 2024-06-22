# mkprog

`mkprog` is a Go program that generates other Go programs using AI. It utilizes the `langchaingo` library to interact with an AI language model and create structured Go projects based on user input.

## Installation

To install `mkprog`, you need Go installed on your machine. If you don't have Go installed, you can download and install it from [https://go.dev/dl/](https://go.dev/dl/).

Once Go is installed, you can set up `mkprog` by running:

```bash
go install github.com/tmc/mkprog@latest
```

## Usage

To use `mkprog`, run the following command:

```shell
mkprog <output directory> "description of the program you want to create"
```

For example:

```shell
mkprog haikuify "Create a program that generates haiku poems based on user input"
```

The generated program will be designed to use the `langchaingo` library to interact with AI models and process user input according to the specified description.

## Options

- `-temp`: Set the temperature for AI generation (typically 0.0 to 1.0). Default is 0.1.

Example:

```shell
mkprog -temp 0.7 my_project "Create a program that generates jokes"
```

## Features

- Generates complete, functional Go programs
- Uses the `langchaingo` library for AI model interaction
- Follows Go best practices
- Provides clear usage instructions and documentation

## Requirements

- Go 1.22 or later
- An Anthropic API key set in the `ANTHROPIC_API_KEY` environment variable

## License

mkprog is open-source software licensed under the MIT license.

## Contributing

Contributions to mkprog are welcome! Please feel free to submit a Pull Request
