package nodra_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These tests keep the Go doc comments of the root package, the only supported import path, in step with the contracts of the internal/core declarations it re-exports.

func parseGoFile(t *testing.T, path string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func readText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Git on Windows may check text files out with CRLF line endings; the tests compare and search text by "\n".
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

// rootFunctionFiles hold the root package's exported functions; api_mygo.go has the MyGo helpers that the nodra_nomygo tag leaves out, and is parsed from disk whatever the build tags.
var rootFunctionFiles = []string{"api.go", "api_mygo.go"}

// rootFunctions returns the doc comment of every exported function declared in rootFunctionFiles, keyed by name.
func rootFunctions(t *testing.T) map[string]string {
	t.Helper()
	functions := map[string]string{}
	for _, file := range rootFunctionFiles {
		for _, decl := range parseGoFile(t, file).Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.IsExported() {
				functions[fn.Name.Name] = fn.Doc.Text()
			}
		}
	}
	if len(functions) == 0 {
		t.Fatal("api.go declares no exported functions")
	}
	return functions
}

// TestRootFunctionsHaveDocComments checks that go doc and pkg.go.dev show the contract of every root function, because the root package is the only supported import path and readers never see the internal/core declarations.
func TestRootFunctionsHaveDocComments(t *testing.T) {
	functions := rootFunctions(t)
	for name, doc := range functions {
		if !strings.HasPrefix(doc, name+" ") {
			t.Errorf("api.go: %s has no doc comment starting with its name; got %q", name, doc)
		}
	}
	// The ownership rules are the contracts most likely to corrupt state when missed, so they must be on the constructors themselves.
	for name, phrases := range map[string][]string{
		"New":             {"shallow", "NewWithClone"},
		"Create":          {"New"},
		"Define":          {"shallow", "DefineWithClone"},
		"NewWithClone":    {"copy all mutable references", "concurrent", "reenter", "nil clone"},
		"DefineWithClone": {"NewWithClone"},
		"BindField":       {"ErrInvalidBinding"},
		"Invalidate":      {"window closes", "repeatedly", "destroyed"},
	} {
		for _, phrase := range phrases {
			if !strings.Contains(functions[name], phrase) {
				t.Errorf("api.go: the doc comment of %s does not mention %q", name, phrase)
			}
		}
	}
	for _, file := range []string{"invalidate.go", "invalidate_mygo.go"} {
		for _, decl := range parseGoFile(t, filepath.Join("internal", "core", file)).Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.IsExported() && !strings.HasPrefix(fn.Doc.Text(), fn.Name.Name+" ") {
				t.Errorf("internal/core/%s: %s has no doc comment", file, fn.Name.Name)
			}
		}
	}
}
