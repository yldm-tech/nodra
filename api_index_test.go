package nodra_test

import (
	"go/ast"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// aliasedCoreTypes returns every type api.go re-exports from internal/core under its own name, the types whose methods Go doc does not list under the root package.
func aliasedCoreTypes(t *testing.T) []string {
	t.Helper()
	var types []string
	for _, decl := range parseGoFile(t, "api.go").Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || !typeSpec.Assign.IsValid() {
				continue
			}
			target := typeSpec.Type
			switch generic := target.(type) {
			case *ast.IndexExpr:
				target = generic.X
			case *ast.IndexListExpr:
				target = generic.X
			}
			selector, ok := target.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			if pkg, ok := selector.X.(*ast.Ident); ok && pkg.Name == "core" && selector.Sel.Name == typeSpec.Name.Name {
				types = append(types, typeSpec.Name.Name)
			}
		}
	}
	if !slices.Contains(types, "Store") {
		t.Fatalf("api.go aliases %v, which does not include Store", types)
	}
	return types
}

// coreMethods returns the exported methods declared in internal/core for each named receiver type. Error and Unwrap are left out: they only implement the error interface, which the error types document by being errors.
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
			if !ok || fn.Recv == nil || !fn.Name.IsExported() || fn.Name.Name == "Error" || fn.Name.Name == "Unwrap" {
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
	if len(methods["Store"]) == 0 {
		t.Fatal("internal/core declares no exported methods on Store")
	}
	return methods
}

var (
	backtickedCode = regexp.MustCompile("`([A-Za-z0-9_.]+)(?:\\(([^`)]*)\\))?`")
	nameSuffix     = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
	typeParameters = regexp.MustCompile(`\[[A-Z][A-Za-z0-9, ]*\]`)
	exportedName   = regexp.MustCompile(`\b[A-Z][A-Za-z0-9]*(?:\.[A-Z][A-Za-z0-9]*)?`)
	listSeparator  = regexp.MustCompile(`^(?:,\s*(?:(?:and|or)\s+)?|\s+(?:and|or)\s+)$`)
	listIntro      = regexp.MustCompile(`^\s+provides\s+$`)
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

// indexedMethods returns the methods an index lists for each type. A method list starts at a qualified name such as History.Undo, or at "Type provides", and continues through the names joined to it by commas, "and" or "or", so `History.Undo`, `Redo`, and `Close` lists three History methods. A bare name anywhere else belongs to no type, so a method name shared by several types counts only for the types whose lists contain it.
func indexedMethods(text string) map[string]map[string]bool {
	text = typeParameters.ReplaceAllString(strings.ReplaceAll(text, "`", ""), "")
	methods := map[string]map[string]bool{}
	add := func(typ, method string) {
		if methods[typ] == nil {
			methods[typ] = map[string]bool{}
		}
		methods[typ][method] = true
	}
	list, previous, previousEnd := "", "", -1
	for _, location := range exportedName.FindAllStringIndex(text, -1) {
		name, gap := text[location[0]:location[1]], ""
		if previousEnd >= 0 {
			gap = text[previousEnd:location[0]]
		}
		switch typ, method, qualified := strings.Cut(name, "."); {
		case qualified:
			list = typ
			add(typ, method)
		case previous != "" && !strings.Contains(previous, ".") && listIntro.MatchString(gap):
			list = previous
			add(list, name)
		case list != "" && listSeparator.MatchString(gap):
			add(list, name)
		default:
			list = ""
		}
		previous, previousEnd = name, location[1]
	}
	return methods
}

// TestIndexedMethodsIsPerType pins the list rule the index tests rely on: a method shared by several types must be listed under each type, not merely appear somewhere in the text.
func TestIndexedMethodsIsPerType(t *testing.T) {
	for _, tc := range []struct {
		text      string
		typ       string
		listed    []string
		notListed []string
	}{
		{"`History.Undo`, `Redo`, `CanUndo`, and `Close` are the lifecycle.", "History", []string{"Undo", "Redo", "CanUndo", "Close"}, nil},
		{"`Definition.New` returns a store whose `ID` is the definition's id.", "Definition", []string{"New"}, []string{"ID"}},
		{"Store provides Get, ID and Close. BindField returns a Field through Field.Set and Field.Edit.", "Field", []string{"Set", "Edit"}, []string{"Get", "ID", "Close"}},
		{"Store provides Get, ID and Close. BindField returns a Field through Field.Set and Field.Edit.", "Store", []string{"Get", "ID", "Close"}, []string{"Set", "Edit"}},
		{"`Derived[T, V]` provides `Get`, `Snapshot`, and `Close`.", "Derived", []string{"Get", "Snapshot", "Close"}, nil},
		{"| Read | `Store.Get`, `State` | `NewWithClone` isolates owned data. |", "Store", []string{"Get", "State"}, []string{"NewWithClone"}},
		{"Task.Done and Task.Status expose completion. Callers must handle Flush or Close errors.", "Task", []string{"Done", "Status"}, []string{"Flush", "Close"}},
	} {
		got := indexedMethods(tc.text)[tc.typ]
		for _, method := range tc.listed {
			if !got[method] {
				t.Errorf("%q: %s.%s is not listed; got %v", tc.text, tc.typ, method, got)
			}
		}
		for _, method := range tc.notListed {
			if got[method] {
				t.Errorf("%q: %s.%s is listed but should not be", tc.text, tc.typ, method)
			}
		}
	}
}

// TestPublicAPIIndexCoversRootExports checks that docs/public-api.md, the v1 method index the compatibility policy reviews, names every root function and lists every exported method of each re-exported internal/core type under that type, because Go doc does not render those methods through the root aliases.
func TestPublicAPIIndexCoversRootExports(t *testing.T) {
	text := readText(t, filepath.Join("docs", "public-api.md"))
	names := backticked(text)
	for name := range rootFunctions(t) {
		if !names[name] {
			t.Errorf("docs/public-api.md does not list %s", name)
		}
	}
	indexed := indexedMethods(text)
	for typ, methods := range coreMethods(t, aliasedCoreTypes(t)...) {
		for _, method := range methods {
			if !indexed[typ][method] {
				t.Errorf("docs/public-api.md does not list %s.%s in a %s method list", typ, method, typ)
			}
		}
	}
}

// TestPackageDocIndexesMethods checks the method index the package comment promises: every root function, and every exported method of each re-exported internal/core type listed under that type.
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
	indexed := indexedMethods(doc)
	for typ, methods := range coreMethods(t, aliasedCoreTypes(t)...) {
		for _, method := range methods {
			if !indexed[typ][method] {
				t.Errorf("doc.go does not index %s.%s in a %s method list", typ, method, typ)
			}
		}
	}
}
