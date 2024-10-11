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
}

func NewTree(dirOnly bool, maxDepth int, minTokens int64, sortByWeight bool) *Tree {
	return &Tree{
		root:         NewNode("."),
		dirOnly:      dirOnly,
		maxDepth:     maxDepth,
		minTokens:    minTokens,
		sortByWeight: sortByWeight,
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

	fmt.Fprintf(w, "%s%s (%d tokens)%s\n", highlight, t.root.name, t.root.tokenCount, reset)
	t.printNode(w, t.root, "", 0, running)
}

func (t *Tree) PrintFinal(w io.Writer) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	fmt.Fprintf(w, "%s (%d tokens)\n", t.root.name, t.root.tokenCount)
	t.printNodeFinal(w, t.root, "", 0)
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
			newPrefix += "└── "
		} else {
			newPrefix += "├── "
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

		t.printNode(w, child, newPrefix, depth+1, running)
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
			newPrefix += "└── "
		} else {
			newPrefix += "├── "
		}

		if child.tokenCount >= t.minTokens {
			if child.isDir {
				fmt.Fprintf(w, "%s%s/ (%d tokens)\n", newPrefix, child.name, child.tokenCount)
			} else if !t.dirOnly {
				fmt.Fprintf(w, "%s%s (%d tokens)\n", newPrefix, child.name, child.tokenCount)
			}
		}

		t.printNodeFinal(w, child, newPrefix, depth+1)
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
	pflag.Parse()

	if err := run(*dirOnly, *maxDepth, *parallelism, *minTokens, *sortByWeight, *noStream); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(dirOnly bool, maxDepth, parallelism int, minTokens int64, sortByWeight, noStream bool) error {
	tree := NewTree(dirOnly, maxDepth, minTokens, sortByWeight)
	inputChan := make(chan string)
	errChan := make(chan error, parallelism)
	doneChan := make(chan struct{})

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

	if noStream {
		<-doneChan
		tree.PrintFinal(os.Stdout)
		return nil
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case err := <-errChan:
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
		case <-ticker.C:
			fmt.Print("\033[2J\033[H") // Clear screen and move cursor to top-left
			tree.Print(os.Stdout, true)
		case <-doneChan:
			fmt.Print("\033[2J\033[H") // Clear screen and move cursor to top-left
			tree.PrintFinal(os.Stdout)
			return nil
		}
	}
}

func processLine(tree *Tree, line string) error {
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 3 {
		return fmt.Errorf("invalid input format: %s", line)
	}

	tokenCount, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid token count: %s", parts[0])
	}

	relativePath := strings.TrimSpace(parts[1])
	tree.Insert(relativePath, tokenCount)
	return nil
}