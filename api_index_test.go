package nodra_test

import (
	"go/ast"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// coreMethods returns the exported methods declared in internal/core for each named receiver type.
func coreMethods(t *testing.T, types ...string) map[string][]string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("internal", "core", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	methods := map[string][]string{}
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		for _, decl := range parseGoFile(t, path).Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || !fn.Name.IsExported() {
				continue
			}
			receiver := fn.Recv.List[0].Type
			if star, ok := receiver.(*ast.StarExpr); ok {
				receiver = star.X
			}
			switch generic := receiver.(type) {
			case *ast.IndexExpr:
				receiver = generic.X
			case *ast.IndexListExpr:
				receiver = generic.X
			}
			if name, ok := receiver.(*ast.Ident); ok && slices.Contains(types, name.Name) {
				methods[name.Name] = append(methods[name.Name], fn.Name.Name)
			}
		}
	}
	for _, name := range types {
		if len(methods[name]) == 0 {
			t.Fatalf("internal/core declares no exported methods on %s", name)
		}
	}
	return methods
}

var (
	backtickedCode = regexp.MustCompile("`([A-Za-z0-9_.]+)(?:\\(([^`)]*)\\))?`")
	nameSuffix     = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
)

// backticked returns every identifier written in backticks, including calls such as `nodra.BindField(store, read, write)`, and expands the `Name(Suffix)` shorthand used for paired constructors such as `NewPersistence(WithOptions)`.
func backticked(markdown string) map[string]bool {
	names := map[string]bool{}
	for _, match := range backtickedCode.FindAllStringSubmatch(markdown, -1) {
		name := strings.TrimPrefix(match[1], "nodra.")
		names[name] = true
		if nameSuffix.MatchString(match[2]) {
			names[name+match[2]] = true
		}
	}
	return names
}

// TestPublicAPIIndexCoversRootExports checks that docs/public-api.md, the v1 method index the compatibility policy reviews, names every root function and the methods of the types whose methods Go doc does not render through the root aliases.
func TestPublicAPIIndexCoversRootExports(t *testing.T) {
	names := backticked(readText(t, filepath.Join("docs", "public-api.md")))
	for name := range rootFunctions(t) {
		if !names[name] {
			t.Errorf("docs/public-api.md does not list %s", name)
		}
	}
	for typ, methods := range coreMethods(t, "Store", "History", "Persistence", "AsyncPersistence", "AsyncAction", "Task", "Scope", "Definition", "Field", "Derived") {
		for _, method := range methods {
			if !names[method] && !names[typ+"."+method] {
				t.Errorf("docs/public-api.md does not list %s.%s", typ, method)
			}
		}
	}
}

// TestPackageDocIndexesMethods checks the method index the package comment promises for the types whose methods Go doc omits from the root aliases.
func TestPackageDocIndexesMethods(t *testing.T) {
	doc := parseGoFile(t, "doc.go").Doc.Text()
	words := map[string]bool{}
	for _, word := range regexp.MustCompile(`[A-Za-z0-9]+`).FindAllString(doc, -1) {
		words[word] = true
	}
	for name := range rootFunctions(t) {
		if !words[name] {
			t.Errorf("doc.go does not mention %s", name)
		}
	}
	for typ, methods := range coreMethods(t, "Store", "History", "Persistence", "AsyncPersistence", "AsyncAction", "Task", "Definition", "Field") {
		for _, method := range methods {
			if !words[method] {
				t.Errorf("doc.go does not index %s.%s", typ, method)
			}
		}
	}
}
