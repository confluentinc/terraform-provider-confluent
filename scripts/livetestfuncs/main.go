// Command livetestfuncs prints the top-level live-test functions (Test*Live and
// Test*DriftDetection) declared in the given Go files, one name per line.
//
// It is used by scripts/live-test-affected-groups.sh to run exactly the live
// tests belonging to the resources/data sources a PR changed -- instead of a
// whole build-tag group. Parsing the AST (rather than scanning text) means a
// file with no //go:build header, or unusual formatting, is still read
// correctly, and only real top-level test functions are returned.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

func main() {
	fset := token.NewFileSet()
	seen := map[string]bool{}
	for _, path := range os.Args[1:] {
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "livetestfuncs: skipping %s: %v\n", path, err)
			continue
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			name := fn.Name.Name
			if !strings.HasPrefix(name, "Test") {
				continue
			}
			if !strings.HasSuffix(name, "Live") && !strings.HasSuffix(name, "DriftDetection") {
				continue
			}
			if !seen[name] {
				seen[name] = true
				fmt.Println(name)
			}
		}
	}
}
