package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"github.com/spf13/pflag"
	"golang.org/x/tools/go/packages"
)

// Embed the HTML template
//go:embed templates
var templateFS embed.FS

// Node represents a package in the dependency graph
type Node struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Path     string   `json:"path"`
	FanIn    int      `json:"fanIn"`
	FanOut   int      `json:"fanOut"`
	Imports  []string `json:"imports"`
	Imported []string `json:"imported"`
	IsStdlib bool     `json:"isStdlib"`
}

// Edge represents a dependency between packages
type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// Graph represents the dependency graph
type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Command-line flags
	outputFile := pflag.StringP("output", "o", "deps.html", "Output file")
	format := pflag.StringP("format", "f", "html", "Output format (html, svg, png, dot)")
	depth := pflag.IntP("depth", "d", 0, "Maximum dependency depth (0 = unlimited)")
	includePatterns := pflag.StringArrayP("include", "i", nil, "Include only packages with this prefix")
	excludePatterns := pflag.StringArrayP("exclude", "e", nil, "Exclude packages with this prefix")
	includeStdlib := pflag.BoolP("stdlib", "s", false, "Include standard library dependencies")
	verbose := pflag.BoolP("verbose", "v", false, "Enable verbose output")

	pflag.Parse()
	args := pflag.Args()

	// Default to current directory if no path provided
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// Validate format
	switch *format {
	case "html", "svg", "png", "dot":
		// Valid formats
	default:
		return fmt.Errorf("invalid format: %s (must be html, svg, png, or dot)", *format)
	}

	// Configure package loading
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedDeps,
		Dir:  path,
	}

	// Load packages
	if *verbose {
		fmt.Println("Loading packages...")
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return fmt.Errorf("failed to load packages: %w", err)
	}

	if *verbose {
		fmt.Printf("Loaded %d packages\n", len(pkgs))
	}

	// Build the dependency graph
	graph, err := buildGraph(pkgs, *includeStdlib, *depth, includePatterns, excludePatterns, *verbose)
	if err != nil {
		return fmt.Errorf("failed to build dependency graph: %w", err)
	}

	// Generate output based on format
	switch *format {
	case "html":
		return generateHTML(graph, *outputFile)
	case "svg", "png", "dot":
		return generateGraphviz(graph, *outputFile, *format)
	}

	return nil
}

func buildGraph(pkgs []*packages.Package, includeStdlib bool, maxDepth int, includePatterns, excludePatterns []string, verbose bool) (*Graph, error) {
	graph := &Graph{
		Nodes: []Node{},
		Edges: []Edge{},
	}

	// Map to track unique nodes and edges
	nodeMap := make(map[string]*Node)
	edgeMap := make(map[string]bool)

	// Process packages
	for _, pkg := range pkgs {
		// Skip packages with errors
		if len(pkg.Errors) > 0 {
			continue
		}

		// Check if the package should be included based on patterns
		if !shouldIncludePackage(pkg.PkgPath, includePatterns, excludePatterns) {
			continue
		}

		// Check if it's a standard library package
		isStdlib := isStandardLibrary(pkg.PkgPath)
		if isStdlib && !includeStdlib {
			continue
		}

		// Create or get the node
		node, exists := nodeMap[pkg.PkgPath]
		if !exists {
			node = &Node{
				ID:       sanitizeID(pkg.PkgPath),
				Name:     filepath.Base(pkg.PkgPath),
				Path:     pkg.PkgPath,
				Imports:  []string{},
				Imported: []string{},
				IsStdlib: isStdlib,
			}
			nodeMap[pkg.PkgPath] = node
		}

		// Process imports
		for importPath := range pkg.Imports {
			importPkg := pkg.Imports[importPath]
			
			// Skip imports with errors
			if len(importPkg.Errors) > 0 {
				continue
			}

			// Check if the import should be included
			importIsStdlib := isStandardLibrary(importPath)
			if importIsStdlib && !includeStdlib {
				continue
			}

			if !shouldIncludePackage(importPath, includePatterns, excludePatterns) {
				continue
			}

			// Create the import node if it doesn't exist
			importNode, exists := nodeMap[importPath]
			if !exists {
				importNode = &Node{
					ID:       sanitizeID(importPath),
					Name:     filepath.Base(importPath),
					Path:     importPath,
					Imports:  []string{},
					Imported: []string{},
					IsStdlib: importIsStdlib,
				}
				nodeMap[importPath] = importNode
			}

			// Create the edge
			edgeKey := fmt.Sprintf("%s->%s", pkg.PkgPath, importPath)
			if !edgeMap[edgeKey] {
				edge := Edge{
					Source: node.ID,
					Target: importNode.ID,
				}
				graph.Edges = append(graph.Edges, edge)
				edgeMap[edgeKey] = true

				// Update node information
				node.Imports = append(node.Imports, importPath)
				node.FanOut++
				importNode.Imported = append(importNode.Imported, pkg.PkgPath)
				importNode.FanIn++
			}
		}
	}

	// Convert node map to slice
	for _, node := range nodeMap {
		graph.Nodes = append(graph.Nodes, *node)
	}

	// Sort nodes by path for consistent output
	sort.Slice(graph.Nodes, func(i, j int) bool {
		return graph.Nodes[i].Path < graph.Nodes[j].Path
	})

	if verbose {
		fmt.Printf("Built graph with %d nodes and %d edges\n", len(graph.Nodes), len(graph.Edges))
	}

	return graph, nil
}

func shouldIncludePackage(pkgPath string, includePatterns, excludePatterns []string) bool {
	// If include patterns are specified, package must match at least one
	if len(includePatterns) > 0 {
		included := false
		for _, pattern := range includePatterns {
			if strings.HasPrefix(pkgPath, pattern) {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}

	// Package must not match any exclude patterns
	for _, pattern := range excludePatterns {
		if strings.HasPrefix(pkgPath, pattern) {
			return false
		}
	}

	return true
}

func isStandardLibrary(pkgPath string) bool {
	return !strings.Contains(pkgPath, ".")
}

func sanitizeID(id string) string {
	return strings.ReplaceAll(id, "/", "_")
}

func generateHTML(graph *Graph, outputFile string) error {
	// Read the template
	tmplContent, err := templateFS.ReadFile("templates/graph.html")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	// Parse the template
	tmpl, err := template.New("graph").Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create the output file
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Convert graph to JSON for the template
	graphJSON, err := json.Marshal(graph)
	if err != nil {
		return fmt.Errorf("failed to marshal graph to JSON: %w", err)
	}

	// Execute the template
	err = tmpl.Execute(file, struct {
		GraphJSON template.JS
	}{
		GraphJSON: template.JS(graphJSON),
	})
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	fmt.Printf("Dependency visualization generated: %s\n", outputFile)
	return nil
}

func generateGraphviz(graph *Graph, outputFile, format string) error {
	// Initialize GraphViz
	g := graphviz.New()
	defer g.Close()

	// Create a new graph
	digraph, err := g.Graph(graphviz.Directed)
	if err != nil {
		return fmt.Errorf("failed to create graph: %w", err)
	}
	defer digraph.Close()

	// Set graph attributes
	digraph.SetLabel("Dependency Graph")
	digraph.SetFontSize(14)

	// Create nodes
	nodes := make(map[string]*cgraph.Node)
	for _, node := range graph.Nodes {
		n, err := digraph.CreateNode(node.ID)
		if err != nil {
			return fmt.Errorf("failed to create node %s: %w", node.ID, err)
		}
		n.SetLabel(node.Name)
		n.SetShape(cgraph.ShapeBox)
		
		// Style nodes differently based on whether they're stdlib or not
		if node.IsStdlib {
			n.SetStyle(cgraph.StyleDotted)
			n.SetFillColor("lightgrey")
		} else {
			n.SetStyle(cgraph.StyleFilled)
			n.SetFillColor("lightblue")
		}
		
		// Add tooltip with full path
		n.SetTooltip(node.Path)
		
		nodes[node.ID] = n
	}

	// Create edges
	for _, edge := range graph.Edges {
		srcNode, srcExists := nodes[edge.Source]
		dstNode, dstExists := nodes[edge.Target]
		if !srcExists || !dstExists {
			continue
		}
		e, err := digraph.CreateEdge("", srcNode, dstNode)
		if err != nil {
			return fmt.Errorf("failed to create edge %s -> %s: %w", edge.Source, edge.Target, err)
		}
		e.SetArrowhead(cgraph.ArrowTypeVee)
	}

	// Render the graph
	var buf bytes.Buffer
	if format == "dot" {
		// For DOT format, just write the DOT representation
		if err := g.Render(digraph, graphviz.XDOT, &buf); err != nil {
			return fmt.Errorf("failed to render graph to DOT: %w", err)
		}
		return os.WriteFile(outputFile, buf.Bytes(), 0644)
	}

	// For other formats, render to the appropriate format
	var gFormat graphviz.Format
	switch format {
	case "svg":
		gFormat = graphviz.SVG
	case "png":
		gFormat = graphviz.PNG
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	// Render the graph
	if err := g.Render(digraph, gFormat, &buf); err != nil {
		return fmt.Errorf("failed to render graph to %s: %w", format, err)
	}

	// Write to file
	if err := os.WriteFile(outputFile, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("Dependency visualization generated: %s\n", outputFile)
	return nil
}