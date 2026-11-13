package bridge_test

import (
	"context"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/yldm-tech/nodra"
	"github.com/yldm-tech/nodra/bridge"
)

// docBlocks returns the fenced code blocks of docs/bridge.md whose info string is lang, without their fences.
func docBlocks(t *testing.T, lang string) []string {
	t.Helper()
	data, err := os.ReadFile("../docs/bridge.md")
	if err != nil {
		t.Fatal(err)
	}
	var blocks []string
	var current []string
	inside, keep := false, false
	for line := range strings.SplitSeq(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if inside && keep {
				blocks = append(blocks, strings.Join(current, "\n"))
			}
			if !inside {
				keep = strings.TrimPrefix(trimmed, "```") == lang
				current = nil
			}
			inside = !inside
			continue
		}
		if inside {
			current = append(current, line)
		}
	}
	return blocks
}

// docPageState builds the PageState type declared in the Go example of docs/bridge.md, with the declared field names, types and tags.
func docPageState(t *testing.T) reflect.Type {
	t.Helper()
	basic := map[string]reflect.Type{"int": reflect.TypeFor[int](), "string": reflect.TypeFor[string](), "bool": reflect.TypeFor[bool]()}
	for _, block := range docBlocks(t, "go") {
		if !strings.Contains(block, "ExposeProjection(") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), "bridge.md", "package doc\nfunc _() {\n"+block+"\n}\n", 0)
		if err != nil {
			t.Fatalf("docs/bridge.md Go example does not parse: %v", err)
		}
		var spec *ast.TypeSpec
		ast.Inspect(file, func(n ast.Node) bool {
			if s, ok := n.(*ast.TypeSpec); ok && s.Name.Name == "PageState" {
				spec = s
			}
			return spec == nil
		})
		if spec == nil {
			continue
		}
		structType, ok := spec.Type.(*ast.StructType)
		if !ok {
			t.Fatal("docs/bridge.md declares PageState as something other than a struct")
		}
		var fields []reflect.StructField
		for _, field := range structType.Fields.List {
			ident, ok := field.Type.(*ast.Ident)
			if !ok || basic[ident.Name] == nil {
				t.Fatalf("docs/bridge.md PageState field type %T is not int, string or bool", field.Type)
			}
			var tag reflect.StructTag
			if field.Tag != nil {
				value, err := strconv.Unquote(field.Tag.Value)
				if err != nil {
					t.Fatal(err)
				}
				tag = reflect.StructTag(value)
			}
			for _, name := range field.Names {
				fields = append(fields, reflect.StructField{Name: name.Name, Type: basic[ident.Name], Tag: tag})
			}
		}
		return reflect.StructOf(fields)
	}
	t.Fatal("docs/bridge.md does not declare PageState in the Go example that exposes it, so a reader cannot tell which JSON keys the page reads")
	return nil
}

// TestDocsPageStateMatchesPage checks that the page code of docs/bridge.md reads keys the documented PageState is encoded with. The page read state?.count while the document never declared PageState, and the untagged struct its other examples suggest is encoded as {"Count":3}, so the page showed its placeholder forever.
func TestDocsPageStateMatchesPage(t *testing.T) {
	pageType := docPageState(t)
	value := reflect.New(pageType).Elem()
	for i := range value.NumField() {
		if field := value.Field(i); field.Kind() == reflect.Int {
			field.SetInt(3)
		}
	}

	b := bridge.New(bridge.Options{})
	defer b.Close()
	store := nodra.New(struct{ Count int }{Count: 3})
	defer func() { _ = store.Close() }()
	stop, err := bridge.ExposeProjection(b, "counter", store, func(struct{ Count int }) any { return value.Interface() })
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	frame, err := b.Get(context.Background(), "counter")
	if err != nil {
		t.Fatal(err)
	}
	var page map[string]any
	if err := json.Unmarshal(frame.Value, &page); err != nil {
		t.Fatal(err)
	}

	read := regexp.MustCompile(`state\?\.(\w+)`)
	var keys int
	for _, block := range docBlocks(t, "tsx") {
		for _, match := range read.FindAllStringSubmatch(block, -1) {
			keys++
			if _, ok := page[match[1]]; !ok {
				t.Errorf("docs/bridge.md page reads state?.%s, but the documented PageState is sent as %s", match[1], frame.Value)
			}
		}
	}
	if keys == 0 {
		t.Fatal("docs/bridge.md has no tsx block that reads state?.<key>")
	}
}
