# token-tree Tool Documentation

## Overview
`token-tree` is a command-line tool that visualizes token counts in a hierarchical tree structure. It's designed to process input data about file token counts and display the results in a tree format, making it easy to understand token distribution across directories and files.

## Usage
```bash
# Basic usage
cat token-counts.txt | token-tree

# Limit depth of the tree (shows only 2 levels)
cat token-counts.txt | token-tree -L2

# Show only directories
cat token-counts.txt | token-tree -d

# Sort by token weight (highest count first)
cat token-counts.txt | token-tree -s

# Reverse mode (root at bottom) - good for streaming
cat token-counts.txt | token-tree -r

# Filter out files with fewer than 100 tokens
cat token-counts.txt | token-tree -m 100
```

## Input Format
The tool expects input in the format of:
```
<token_count> <file_path>
```

Example:
```
1234 /path/to/file.txt
56 /another/path/to/file.md
```

## Common Commands
```bash
# Build and install the tool
cd /Users/tmc/go/src/github.com/tmc/mkprog/tools/token-tree && go install

# Count tokens in files and display as tree
~/code-to-gpt.sh --count-tokens | token-tree

# Limit tree depth and show only directories
~/code-to-gpt.sh --count-tokens | token-tree -L2 -d

# Reverse mode with limited depth
~/code-to-gpt.sh --count-tokens | token-tree -L2 -r
```

## Display Modes

### TTY Mode (Interactive Terminal)
When output is to a terminal:
- Updates the display in real-time as files are processed
- Clears screen between updates
- Highlights recently updated nodes

### Non-TTY Mode (Pipes and Redirects)
When output is piped to other commands or files:
- Outputs the final tree only once at completion
- No ANSI escape sequences or clearing
- Clean output suitable for further processing

## Options
- `-d, --directories`: Show only directories
- `-L, --max-depth`: Limit the depth of the tree
- `-P, --parallelism`: Number of parallel workers (default: 1)
- `-m, --min-tokens`: Minimum token count to display
- `-s, --sort-weight`: Sort by token weight (sum of tokens)
- `-n, --no-stream`: Disable streaming output
- `-r, --reverse`: Print root token count at the end (good for streaming)