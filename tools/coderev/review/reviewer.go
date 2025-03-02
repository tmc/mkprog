package review

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Reviewer is responsible for reviewing code
type Reviewer struct {
	config *Config
	fset   *token.FileSet
	issues []*Issue
}

// NewReviewer creates a new code reviewer
func NewReviewer(config *Config) *Reviewer {
	if config == nil {
		config = DefaultConfig()
	}
	return &Reviewer{
		config: config,
		fset:   token.NewFileSet(),
		issues: make([]*Issue, 0),
	}
}

// ReviewPaths reviews the code in the specified paths
func (r *Reviewer) ReviewPaths(paths []string) (*Results, error) {
	// Expand ./... patterns
	expandedPaths, err := r.expandPaths(paths)
	if err != nil {
		return nil, err
	}

	// Parse Go files
	for _, path := range expandedPaths {
		if r.shouldIgnore(path) {
			continue
		}

		if strings.HasSuffix(path, ".go") {
			if err := r.reviewGoFile(path); err != nil {
				return nil, err
			}
		}
	}

	// Return results
	return &Results{
		Issues: r.issues,
	}, nil
}

// ReviewDiff reviews the changes between the current branch and the specified branch
func (r *Reviewer) ReviewDiff(branch string, paths []string) (*Results, error) {
	// Get changed files from git
	cmd := exec.Command("git", "diff", "--name-only", branch)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	// Process changed files
	changedFiles := strings.Split(strings.TrimSpace(string(output)), "\n")
	filesToReview := make([]string, 0)

	for _, file := range changedFiles {
		if strings.HasSuffix(file, ".go") && !r.shouldIgnore(file) {
			filesToReview = append(filesToReview, file)
		}
	}

	// Review each file
	for _, path := range filesToReview {
		if err := r.reviewGoFile(path); err != nil {
			return nil, err
		}
	}

	// Return results
	return &Results{
		Issues: r.issues,
	}, nil
}

// expandPaths expands directory patterns like ./...
func (r *Reviewer) expandPaths(paths []string) ([]string, error) {
	result := make([]string, 0)

	for _, path := range paths {
		if strings.HasSuffix(path, "/...") {
			// Load Go packages
			baseDir := strings.TrimSuffix(path, "/...")
			cfg := &packages.Config{
				Mode:  packages.NeedName | packages.NeedFiles,
				Tests: false,
				Dir:   baseDir,
			}

			pkgs, err := packages.Load(cfg, path)
			if err != nil {
				return nil, fmt.Errorf("failed to load packages: %w", err)
			}

			// Add all Go files
			for _, pkg := range pkgs {
				for _, file := range pkg.GoFiles {
					result = append(result, file)
				}
			}
		} else {
			// Check if it's a directory
			info, err := os.Stat(path)
			if err != nil {
				return nil, fmt.Errorf("failed to stat path %s: %w", path, err)
			}

			if info.IsDir() {
				// Find all Go files in directory
				err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if !info.IsDir() && strings.HasSuffix(filePath, ".go") {
						result = append(result, filePath)
					}
					return nil
				})
				if err != nil {
					return nil, fmt.Errorf("failed to walk directory %s: %w", path, err)
				}
			} else if strings.HasSuffix(path, ".go") {
				result = append(result, path)
			}
		}
	}

	return result, nil
}

// shouldIgnore checks if a file should be ignored
func (r *Reviewer) shouldIgnore(path string) bool {
	for _, pattern := range r.config.Ignore {
		matched, err := filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}
	}

	// If include patterns are specified, file must match one
	if len(r.config.Include) > 0 {
		for _, pattern := range r.config.Include {
			matched, err := filepath.Match(pattern, path)
			if err == nil && matched {
				return false
			}
		}
		return true
	}

	// If exclude patterns are specified, file must not match any
	for _, pattern := range r.config.Exclude {
		matched, err := filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}
	}

	return false
}

// reviewGoFile reviews a single Go file
func (r *Reviewer) reviewGoFile(path string) error {
	// Parse the file
	f, err := parser.ParseFile(r.fset, path, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}

	// Apply all relevant rules
	r.checkFunctionLength(f, path)
	r.checkComments(f, path)
	r.checkErrorHandling(f, path)
	r.checkNilErrors(f, path)
	r.checkUnusedImports(f, path)

	return nil
}

// checkFunctionLength checks if functions are too long
func (r *Reviewer) checkFunctionLength(f *ast.File, path string) {
	maxLength, ok := r.config.Rules["max_function_length"].(int)
	if !ok {
		maxLength = 100 // Default
	}

	ast.Inspect(f, func(n ast.Node) bool {
		// Check function declarations
		if fn, ok := n.(*ast.FuncDecl); ok {
			if fn.Body == nil {
				return true
			}

			// Calculate function length (in lines)
			startPos := r.fset.Position(fn.Pos())
			endPos := r.fset.Position(fn.End())
			lineCount := endPos.Line - startPos.Line

			if lineCount > maxLength {
				r.addIssue(&Issue{
					File:     path,
					Line:     startPos.Line,
					Severity: "warning",
					Message:  fmt.Sprintf("Function %s is too long (%d lines, max %d)", fn.Name.Name, lineCount, maxLength),
					Rule:     "max_function_length",
				})
			}
		}
		return true
	})
}

// checkComments checks if exported functions and types have comments
func (r *Reviewer) checkComments(f *ast.File, path string) {
	requireComments, ok := r.config.Rules["require_comments"].(bool)
	if !ok || !requireComments {
		return
	}

	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			// Check exported functions
			if d.Name.IsExported() {
				if d.Doc == nil || len(d.Doc.List) == 0 {
					pos := r.fset.Position(d.Pos())
					r.addIssue(&Issue{
						File:     path,
						Line:     pos.Line,
						Severity: "warning",
						Message:  fmt.Sprintf("Exported function %s lacks documentation", d.Name.Name),
						Rule:     "require_comments",
					})
				}
			}
		case *ast.GenDecl:
			// Check exported types, vars, consts
			if d.Tok == token.TYPE || d.Tok == token.VAR || d.Tok == token.CONST {
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Name.IsExported() && (d.Doc == nil || len(d.Doc.List) == 0) {
							pos := r.fset.Position(s.Pos())
							r.addIssue(&Issue{
								File:     path,
								Line:     pos.Line,
								Severity: "warning",
								Message:  fmt.Sprintf("Exported type %s lacks documentation", s.Name.Name),
								Rule:     "require_comments",
							})
						}
					case *ast.ValueSpec:
						for _, name := range s.Names {
							if name.IsExported() && (d.Doc == nil || len(d.Doc.List) == 0) {
								pos := r.fset.Position(name.Pos())
								r.addIssue(&Issue{
									File:     path,
									Line:     pos.Line,
									Severity: "warning",
									Message:  fmt.Sprintf("Exported variable %s lacks documentation", name.Name),
									Rule:     "require_comments",
								})
							}
						}
					}
				}
			}
		}
	}
}

// checkErrorHandling checks for proper error handling
func (r *Reviewer) checkErrorHandling(f *ast.File, path string) {
	enforceErrorHandling, ok := r.config.Rules["enforce_error_handling"].(bool)
	if !ok || !enforceErrorHandling {
		return
	}

	ast.Inspect(f, func(n ast.Node) bool {
		// Look for assignment statements like x, err := foo()
		if assign, ok := n.(*ast.AssignStmt); ok {
			// Check if last value is error
			if len(assign.Rhs) > 0 && len(assign.Lhs) > 1 {
				// Look for calls that might return errors
				if call, ok := assign.Rhs[0].(*ast.CallExpr); ok {
					lastVar := assign.Lhs[len(assign.Lhs)-1]
					if ident, ok := lastVar.(*ast.Ident); ok && ident.Name == "err" {
						// Check if error is checked
						if !r.isErrorChecked(f, ident, assign) {
							pos := r.fset.Position(assign.Pos())
							r.addIssue(&Issue{
								File:     path,
								Line:     pos.Line,
								Severity: "error",
								Message:  "Error is not checked",
								Rule:     "enforce_error_handling",
								Snippet:  fmt.Sprintf("%s", r.nodeSource(assign)),
							})
						}
					}
				}
			}
		}
		return true
	})
}

// isErrorChecked checks if an error variable is checked
func (r *Reviewer) isErrorChecked(f *ast.File, ident *ast.Ident, after ast.Node) bool {
	checked := false
	
	ast.Inspect(f, func(n ast.Node) bool {
		// Skip nodes before the assignment
		if n.Pos() <= after.End() {
			return true
		}

		// Look for if statements
		if ifStmt, ok := n.(*ast.IfStmt); ok {
			if r.containsIdent(ifStmt.Cond, ident.Name) {
				checked = true
				return false
			}
		}
		
		// Error is being returned
		if returnStmt, ok := n.(*ast.ReturnStmt); ok {
			for _, ret := range returnStmt.Results {
				if id, ok := ret.(*ast.Ident); ok && id.Name == ident.Name {
					checked = true
					return false
				}
			}
		}

		return true
	})

	return checked
}

// checkNilErrors checks for nil error checks
func (r *Reviewer) checkNilErrors(f *ast.File, path string) {
	checkNilErrors, ok := r.config.Rules["check_nil_errors"].(bool)
	if !ok || !checkNilErrors {
		return
	}

	ast.Inspect(f, func(n ast.Node) bool {
		// Look for if statements like if err != nil
		if ifStmt, ok := n.(*ast.IfStmt); ok {
			if binExpr, ok := ifStmt.Cond.(*ast.BinaryExpr); ok {
				if binExpr.Op == token.NEQ {
					if ident, ok := binExpr.X.(*ast.Ident); ok && ident.Name == "err" {
						if nilLit, ok := binExpr.Y.(*ast.Ident); ok && nilLit.Name == "nil" {
							// Check if the if block has a return or panic
							if !r.hasReturnOrPanic(ifStmt.Body) {
								pos := r.fset.Position(ifStmt.Pos())
								r.addIssue(&Issue{
									File:     path,
									Line:     pos.Line,
									Severity: "warning",
									Message:  "Error check without proper handling (return or panic)",
									Rule:     "check_nil_errors",
									Snippet:  fmt.Sprintf("%s", r.nodeSource(ifStmt)),
								})
							}
						}
					}
				}
			}
		}
		return true
	})
}

// hasReturnOrPanic checks if a block has a return or panic statement
func (r *Reviewer) hasReturnOrPanic(block *ast.BlockStmt) bool {
	hasRet := false
	ast.Inspect(block, func(n ast.Node) bool {
		if _, ok := n.(*ast.ReturnStmt); ok {
			hasRet = true
			return false
		}
		if expr, ok := n.(*ast.ExprStmt); ok {
			if call, ok := expr.X.(*ast.CallExpr); ok {
				if fn, ok := call.Fun.(*ast.Ident); ok && fn.Name == "panic" {
					hasRet = true
					return false
				}
			}
		}
		return true
	})
	return hasRet
}

// checkUnusedImports checks for unused imports
func (r *Reviewer) checkUnusedImports(f *ast.File, path string) {
	checkUnusedImports, ok := r.config.Rules["unused_imports"].(bool)
	if !ok || !checkUnusedImports {
		return
	}

	// Collect all imports
	imports := make(map[string]string) // name -> path
	for _, imp := range f.Imports {
		var name string
		if imp.Name != nil {
			name = imp.Name.Name
		} else {
			// Extract name from path
			path := strings.Trim(imp.Path.Value, "\"")
			parts := strings.Split(path, "/")
			name = parts[len(parts)-1]
		}
		imports[name] = strings.Trim(imp.Path.Value, "\"")
	}

	// Check for usage
	used := make(map[string]bool)
	ast.Inspect(f, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if x, ok := sel.X.(*ast.Ident); ok {
				used[x.Name] = true
			}
		}
		return true
	})

	// Report unused imports
	for name, path := range imports {
		if !used[name] && name != "_" && name != "." {
			for _, imp := range f.Imports {
				impPath := strings.Trim(imp.Path.Value, "\"")
				if impPath == path {
					pos := r.fset.Position(imp.Pos())
					r.addIssue(&Issue{
						File:     path,
						Line:     pos.Line,
						Severity: "warning",
						Message:  fmt.Sprintf("Unused import: %s", path),
						Rule:     "unused_imports",
					})
					break
				}
			}
		}
	}
}

// containsIdent checks if an expression contains an identifier
func (r *Reviewer) containsIdent(expr ast.Expr, name string) bool {
	contains := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && id.Name == name {
			contains = true
			return false
		}
		return true
	})
	return contains
}

// nodeSource returns the source code for a node
func (r *Reviewer) nodeSource(node ast.Node) string {
	// This is a simplified version. In a real implementation,
	// you would read the source file and extract the relevant portion.
	return fmt.Sprintf("%T at %v", node, r.fset.Position(node.Pos()))
}

// addIssue adds an issue to the reviewer's list
func (r *Reviewer) addIssue(issue *Issue) {
	// Skip if severity doesn't meet threshold
	if !isAtLeastSeverity(issue.Severity, r.config.SeverityThreshold) {
		return
	}

	// Check if we've reached the maximum number of issues
	if r.config.MaxErrors > 0 && len(r.issues) >= r.config.MaxErrors {
		return
	}

	// Check for rule filters
	if len(r.config.Filters) > 0 {
		found := false
		for _, filter := range r.config.Filters {
			if filter == issue.Rule {
				found = true
				break
			}
		}
		if !found {
			return
		}
	}

	r.issues = append(r.issues, issue)
}

// isAtLeastSeverity checks if a severity level meets a threshold
func isAtLeastSeverity(level, threshold string) bool {
	levels := map[string]int{
		"info":    0,
		"warning": 1,
		"error":   2,
	}

	levelVal, ok1 := levels[level]
	thresholdVal, ok2 := levels[threshold]

	if !ok1 || !ok2 {
		return false
	}

	return levelVal >= thresholdVal
}