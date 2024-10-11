# token-tree 🌳

[![Go Report Card](https://goreportcard.com/badge/github.com/tmc/token-tree)](https://goreportcard.com/report/github.com/tmc/token-tree)
[![GoDoc](https://godoc.org/github.com/tmc/token-tree?status.svg)](https://godoc.org/github.com/tmc/token-tree)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

token-tree is a powerful Go-based CLI tool that generates a tree-like visualization of token counts for files and directories. It processes a stream of input lines representing token counts for files and displays the results in a hierarchical structure, similar to the Unix 'tree' command but with additional token count information.

## 🚀 Features

- 🌳 Tree-like visualization of token counts for files and directories
- 📁 Option to display only directories with the `-d` flag
- 🔢 Limit tree depth with the `-L` flag
- 🔄 Live updates as new input lines are received
- 🚀 Efficient processing of large file lists with parallel workers
- 🎨 Clear and intuitive output format
- 📊 Summary statistics including total files, directories, and tokens
- 🔍 Customizable minimum token count filter

## 🛠 Installation

1. Ensure you have Go 1.20 or later installed on your system.
2. Install token-tree using `go get`:

   ```
   go get -u github.com/tmc/token-tree
   ```

   Or clone the repository and build manually:

   ```
   git clone https://github.com/tmc/token-tree.git
   cd token-tree
   go build
   ```

## 📖 Usage

Basic usage:

```
some_token_counter | token-tree [flags]
```

### Flags

- `-d, --directories`: Show only directories
- `-L, --max-depth int`: Limit the depth of the tree
- `-P, --parallelism int`: Number of parallel workers (default 1)
- `-m, --min-tokens int`: Minimum token count to display (default 0)
- `-s, --sort string`: Sort order for nodes (options: "name", "tokens", "none"; default "name")
- `-c, --no-color`: Disable colored output

### Example

Display a tree-like structure of directories up to a depth of 2 levels, sorted by token count:

```
$ some_token_counter | token-tree -L 2 -d -s tokens
```

## 📥 Input Format

The input should be a stream of lines in the following format:

```
<token_count> <relative_path> <absolute_path>
```

Example:
```
100 src/main.go /home/user/project/src/main.go
50 src/utils/helper.go /home/user/project/src/utils/helper.go
```

## 📤 Output

The tool generates a tree-like structure with token counts:

```
src/ (150 tokens)
├── main.go (100 tokens)
└── utils/ (50 tokens)
    └── helper.go (50 tokens)

Summary:
  Total files: 2
  Total directories: 2
  Total tokens: 150
```

## 🐛 Error Handling

token-tree handles various edge cases and errors, including:

- Empty input
- Malformed input lines
- Duplicate file paths
- Very deep directory structures
- Files with unusually large token counts

If an error occurs, the program will display a warning message and continue processing.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
