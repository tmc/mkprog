package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/pflag"
	"golang.org/x/term"
)

type Node struct {
	name        string
	tokenCount  int64
	children    map[string]*Node
	isDir       bool
	lastUpdated time.Time
}

func NewNode(name string) *Node {
	return &Node{
		name:        name,
		tokenCount:  0,
		children:    make(map[string]*Node),
		isDir:       true,
		lastUpdated: time.Now(),
	}
}

type Tree struct {
	root         *Node
	mu           sync.RWMutex
	dirOnly      bool
	maxDepth     int
	minTokens    int64
	sortByWeight bool
	reverse      bool
}

func NewTree(dirOnly bool, maxDepth int, minTokens int64, sortByWeight bool, reverse bool) *Tree {
	return &Tree{
		root:         NewNode("."),
		dirOnly:      dirOnly,
		maxDepth:     maxDepth,
		minTokens:    minTokens,
		sortByWeight: sortByWeight,
		reverse:      reverse,
	}
}

func (t *Tree) Insert(path string, tokenCount int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	parts := strings.Split(filepath.Clean(path), string(os.PathSeparator))
	current := t.root

	for i, part := range parts {
		if part == "" {
			continue
		}

		if _, exists := current.children[part]; !exists {
			current.children[part] = NewNode(part)
		}
		current = current.children[part]

		if i == len(parts)-1 {
			current.isDir = false
		}
		current.tokenCount += tokenCount
		current.lastUpdated = time.Now()
	}
	t.root.tokenCount += tokenCount
	t.root.lastUpdated = time.Now()
}

func (t *Tree) Print(w io.Writer, running bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	highlight := ""
	if running {
		highlight = "\033[1;97m" // Bold and bright white
	}
	reset := "\033[0m"

	if !t.reverse {
		fmt.Fprintf(w, "%s%s (%d tokens)%s\n", highlight, t.root.name, t.root.tokenCount, reset)
		t.printNode(w, t.root, "", 0, running)
	} else {
		// For reverse mode, print all children first then the root
		t.printChildrenOnly(w, t.root, "", 0, running)
		fmt.Fprintf(w, "%s%s (%d tokens)%s\n", highlight, t.root.name, t.root.tokenCount, reset)
	}
}

func (t *Tree) PrintFinal(w io.Writer) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.reverse {
		fmt.Fprintf(w, "%s (%d tokens)\n", t.root.name, t.root.tokenCount)
		t.printNodeFinal(w, t.root, "", 0)
	} else {
		// For reverse mode, print all children first then the root
		t.printChildrenFinal(w, t.root, "", 0)
		fmt.Fprintf(w, "%s (%d tokens)\n", t.root.name, t.root.tokenCount)
	}
}

// printChildrenOnly prints only the direct children of the node
func (t *Tree) printChildrenOnly(w io.Writer, node *Node, prefix string, depth int, running bool) {
	childrenKeys := t.getSortedChildrenKeys(node)

	for i, key := range childrenKeys {
		child := node.children[key]
		childPrefix := ""

		if i == len(childrenKeys)-1 {
			childPrefix = "└── " // Use standard tree notation
			if t.reverse {
				childPrefix = "┌── " // Change notation for reverse mode
			}
		} else {
			childPrefix = "├── " // Use standard tree notation
			if t.reverse {
				childPrefix = "┬── " // Change notation for reverse mode
			}
		}

		if child.tokenCount >= t.minTokens {
			highlight := ""
			if running && time.Since(child.lastUpdated) < 500*time.Millisecond {
				highlight = "\033[1;97m" // Bold and bright white
			}
			reset := "\033[0m"

			if child.isDir {
				fmt.Fprintf(w, "%s%s%s/ (%d tokens)%s\n", childPrefix, highlight, child.name, child.tokenCount, reset)
			} else if !t.dirOnly {
				fmt.Fprintf(w, "%s%s%s (%d tokens)%s\n", childPrefix, highlight, child.name, child.tokenCount, reset)
			}
		}

		// Prepare prefix for child's children
		childrenPrefix := ""
		if i == len(childrenKeys)-1 {
			childrenPrefix = "    " // 4 spaces after the last item
		} else {
			if t.reverse {
				childrenPrefix = "│   " // vertical bar + 3 spaces for non-last items
			} else {
				childrenPrefix = "│   " // vertical bar + 3 spaces for non-last items
			}
		}

		// Process child's children
		t.printNode(w, child, childrenPrefix, 1, running)
	}
}

// printChildrenFinal prints only the direct children of the node (for final output)
func (t *Tree) printChildrenFinal(w io.Writer, node *Node, prefix string, depth int) {
	childrenKeys := t.getSortedChildrenKeys(node)

	for i, key := range childrenKeys {
		child := node.children[key]
		childPrefix := ""

		if i == len(childrenKeys)-1 {
			childPrefix = "└── " // Use standard tree notation
			if t.reverse {
				childPrefix = "┌── " // Change notation for reverse mode
			}
		} else {
			childPrefix = "├── " // Use standard tree notation
			if t.reverse {
				childPrefix = "┬── " // Change notation for reverse mode
			}
		}

		if child.tokenCount >= t.minTokens {
			if child.isDir {
				fmt.Fprintf(w, "%s%s/ (%d tokens)\n", childPrefix, child.name, child.tokenCount)
			} else if !t.dirOnly {
				fmt.Fprintf(w, "%s%s (%d tokens)\n", childPrefix, child.name, child.tokenCount)
			}
		}

		// Prepare prefix for child's children
		childrenPrefix := ""
		if i == len(childrenKeys)-1 {
			childrenPrefix = "    " // 4 spaces after the last item
		} else {
			if t.reverse {
				childrenPrefix = "│   " // vertical bar + 3 spaces for non-last items
			} else {
				childrenPrefix = "│   " // vertical bar + 3 spaces for non-last items
			}
		}

		// Process child's children
		t.printNodeFinal(w, child, childrenPrefix, 1)
	}
}

func (t *Tree) printNode(w io.Writer, node *Node, prefix string, depth int, running bool) {
	if t.maxDepth > 0 && depth > t.maxDepth {
		return
	}

	childrenKeys := t.getSortedChildrenKeys(node)

	for i, key := range childrenKeys {
		child := node.children[key]
		newPrefix := prefix
		if i == len(childrenKeys)-1 {
			if t.reverse {
				newPrefix += "┌── " // Change notation for reverse mode
			} else {
				newPrefix += "└── " // Standard tree notation
			}
		} else {
			if t.reverse {
				newPrefix += "┬── " // Change notation for reverse mode
			} else {
				newPrefix += "├── " // Standard tree notation
			}
		}

		if child.tokenCount >= t.minTokens {
			highlight := ""
			if running && time.Since(child.lastUpdated) < 500*time.Millisecond {
				highlight = "\033[1;97m" // Bold and bright white
			}
			reset := "\033[0m"

			if child.isDir {
				fmt.Fprintf(w, "%s%s%s/ (%d tokens)%s\n", newPrefix, highlight, child.name, child.tokenCount, reset)
			} else if !t.dirOnly {
				fmt.Fprintf(w, "%s%s%s (%d tokens)%s\n", newPrefix, highlight, child.name, child.tokenCount, reset)
			}
		}

		childPrefix := prefix
		if i == len(childrenKeys)-1 {
			childPrefix += "    " // 4 spaces after the last item
		} else {
			if t.reverse {
				childPrefix += "│   " // vertical bar + 3 spaces for non-last items
			} else {
				childPrefix += "│   " // vertical bar + 3 spaces for non-last items
			}
		}

		t.printNode(w, child, childPrefix, depth+1, running)
	}
}

func (t *Tree) printNodeFinal(w io.Writer, node *Node, prefix string, depth int) {
	if t.maxDepth > 0 && depth > t.maxDepth {
		return
	}

	childrenKeys := t.getSortedChildrenKeys(node)

	for i, key := range childrenKeys {
		child := node.children[key]
		newPrefix := prefix
		if i == len(childrenKeys)-1 {
			if t.reverse {
				newPrefix += "┌── " // Change notation for reverse mode
			} else {
				newPrefix += "└── " // Standard tree notation
			}
		} else {
			if t.reverse {
				newPrefix += "┬── " // Change notation for reverse mode
			} else {
				newPrefix += "├── " // Standard tree notation
			}
		}

		if child.tokenCount >= t.minTokens {
			if child.isDir {
				fmt.Fprintf(w, "%s%s/ (%d tokens)\n", newPrefix, child.name, child.tokenCount)
			} else if !t.dirOnly {
				fmt.Fprintf(w, "%s%s (%d tokens)\n", newPrefix, child.name, child.tokenCount)
			}
		}

		childPrefix := prefix
		if i == len(childrenKeys)-1 {
			childPrefix += "    " // 4 spaces after the last item
		} else {
			if t.reverse {
				childPrefix += "│   " // vertical bar + 3 spaces
			} else {
				childPrefix += "│   " // vertical bar + 3 spaces
			}
		}

		t.printNodeFinal(w, child, childPrefix, depth+1)
	}
}

func (t *Tree) getSortedChildrenKeys(node *Node) []string {
	childrenKeys := make([]string, 0, len(node.children))
	for k := range node.children {
		childrenKeys = append(childrenKeys, k)
	}

	if t.sortByWeight {
		sort.Slice(childrenKeys, func(i, j int) bool {
			return node.children[childrenKeys[i]].tokenCount > node.children[childrenKeys[j]].tokenCount
		})
	} else {
		sort.Strings(childrenKeys)
	}

	return childrenKeys
}

func main() {
	dirOnly := pflag.BoolP("directories", "d", false, "Show only directories")
	maxDepth := pflag.IntP("max-depth", "L", 0, "Limit the depth of the tree")
	parallelism := pflag.IntP("parallelism", "P", 1, "Number of parallel workers")
	minTokens := pflag.Int64P("min-tokens", "m", 0, "Minimum token count to display")
	sortByWeight := pflag.BoolP("sort-weight", "s", false, "Sort by token weight (sum of tokens)")
	noStream := pflag.BoolP("no-stream", "n", false, "Disable streaming output")
	reverse := pflag.BoolP("reverse", "r", false, "Print root token count at the end (good for streaming)")
	pflag.Parse()

	if err := run(*dirOnly, *maxDepth, *parallelism, *minTokens, *sortByWeight, *noStream, *reverse); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(dirOnly bool, maxDepth, parallelism int, minTokens int64, sortByWeight, noStream, reverse bool) error {
	tree := NewTree(dirOnly, maxDepth, minTokens, sortByWeight, reverse)
	inputChan := make(chan string)
	errChan := make(chan error, parallelism)
	doneChan := make(chan struct{})

	// Check if stdout is a terminal
	isStdoutTTY := term.IsTerminal(int(os.Stdout.Fd()))

	var wg sync.WaitGroup
	for i := 0; i < parallelism; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for line := range inputChan {
				if err := processLine(tree, line); err != nil {
					errChan <- err
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(errChan)
		close(doneChan)
	}()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			inputChan <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			errChan <- err
		}
		close(inputChan)
	}()

	// Handle different output modes based on TTY status and stream flag
	if noStream {
		// No streaming mode - wait for completion and print final result
		<-doneChan
		tree.PrintFinal(os.Stdout)
		return nil
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	if isStdoutTTY {
		// Interactive TTY mode with frequent updates
		for {
			select {
			case err := <-errChan:
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
				}
			case <-ticker.C:
				// Clear screen and redraw
				fmt.Print("\033[2J\033[H") // Clear screen and move cursor to top-left
				tree.Print(os.Stdout, true)
			case <-doneChan:
				fmt.Print("\033[2J\033[H") // Clear screen and move cursor to top-left
				tree.PrintFinal(os.Stdout)
				return nil
			}
		}
	} else {
		// Non-TTY mode - print only once at the end
		<-doneChan
		tree.PrintFinal(os.Stdout)
		return nil
	}
}

// Expects a line in the format: "(optional whitespace)1234 /path/to/file"
func processLine(tree *Tree, line string) error {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return fmt.Errorf("invalid input format: %s", line)
	}

	tokenCount, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid token count: %s", parts[0])
	}

	relativePath := strings.Join(parts[1:], " ")
	tree.Insert(relativePath, tokenCount)
	return nil
}