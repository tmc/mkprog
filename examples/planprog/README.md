# PlanProg

PlanProg is a Go program that helps users define and refine their program descriptions by interacting with an AI assistant. It uses the Anthropic language model to provide feedback and ask relevant questions, ensuring that the program description is sufficiently detailed and well-defined.

## Installation

1. Make sure you have Go installed on your system (version 1.20 or later).
2. Clone this repository or download the source code.
3. Navigate to the project directory.
4. Run `go mod tidy` to download the required dependencies.

## Usage

1. Set up your Anthropic API key as an environment variable:

   ```
   export ANTHROPIC_API_KEY=your_api_key_here
   ```

2. Run the program:

   ```
   go run main.go
   ```

3. Follow the prompts to provide your initial program description and respond to the AI's feedback.

4. Continue the conversation until you feel your program description is sufficiently defined.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

