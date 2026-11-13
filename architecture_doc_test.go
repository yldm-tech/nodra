package nodra_test

import (
	"errors"
	"maps"
	"strings"
	"testing"

	"github.com/yldm-tech/nodra"
)

// TestArchitectureCopyOnWriteSnippet runs the copy-on-write UpdateErr example of docs/architecture.md and keeps the document identical to it. The example used to copy the map with maps.Clone, which returns nil for a nil map, so its first update of a zero-value State panicked with "assignment to entry in nil map".
func TestArchitectureCopyOnWriteSnippet(t *testing.T) {
	type Item struct{ Count int }
	type State struct{ Items map[string]Item }
	run := func(store *nodra.Store[State], updatedItem Item, validate func(State) error) error {
		// architecture copy-on-write snippet: begin
		err := store.UpdateErr(func(state *State) error {
			items := make(map[string]Item, len(state.Items)+1)
			maps.Copy(items, state.Items)
			items["word"] = updatedItem
			state.Items = items
			return validate(*state)
		})
		// architecture copy-on-write snippet: end
		return err
	}

	source := readText(t, "architecture_doc_test.go")
	_, rest, found := strings.Cut(source, "\t\t// architecture copy-on-write snippet: begin\n")
	snippet, _, closed := strings.Cut(rest, "\t\t// architecture copy-on-write snippet: end\n")
	if !found || !closed {
		t.Fatal("architecture_doc_test.go: copy-on-write snippet markers not found")
	}
	var want []string
	for line := range strings.SplitSeq(strings.TrimSuffix(snippet, "\n"), "\n") {
		want = append(want, strings.TrimPrefix(line, "\t\t"))
	}
	var got string
	for _, block := range fencedBlocks(readText(t, "docs/architecture.md")) {
		if strings.Contains(block, "store.UpdateErr(") {
			got = block
		}
	}
	if wantBlock := "```go\n" + strings.Join(want, "\n"); got != wantBlock {
		t.Errorf("docs/architecture.md copy-on-write example differs from the snippet in TestArchitectureCopyOnWriteSnippet:\n got:\n%s\nwant:\n%s", got, wantBlock)
	}

	accept := func(State) error { return nil }
	errInvalid := errors.New("invalid")
	reject := func(State) error { return errInvalid }

	t.Run("zero value", func(t *testing.T) {
		store := nodra.New(State{})
		defer func() { _ = store.Close() }()
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("copy-on-write snippet panicked on a zero-value State: %v", r)
			}
		}()
		if err := run(store, Item{Count: 1}, accept); err != nil {
			t.Fatal(err)
		}
		if got := store.Get().Items["word"]; got != (Item{Count: 1}) {
			t.Fatalf("Items[word] = %+v, want the updated item", got)
		}
	})
	t.Run("retained snapshot", func(t *testing.T) {
		store := nodra.New(State{Items: map[string]Item{"other": {Count: 2}}})
		defer func() { _ = store.Close() }()
		before := store.Get()
		if err := run(store, Item{Count: 1}, accept); err != nil {
			t.Fatal(err)
		}
		if _, ok := before.Items["word"]; ok || len(before.Items) != 1 {
			t.Fatalf("retained snapshot changed to %+v", before.Items)
		}
		if got := store.Get().Items; len(got) != 2 || got["other"] != (Item{Count: 2}) || got["word"] != (Item{Count: 1}) {
			t.Fatalf("Items = %+v, want the other item kept and the word item added", got)
		}
	})
	t.Run("rejected update", func(t *testing.T) {
		initial := map[string]Item{"other": {Count: 2}}
		store := nodra.New(State{Items: initial})
		defer func() { _ = store.Close() }()
		if err := run(store, Item{Count: 1}, reject); !errors.Is(err, errInvalid) {
			t.Fatalf("UpdateErr returned %v, want the validation error", err)
		}
		if _, ok := initial["word"]; ok || store.Version() != 0 {
			t.Fatalf("rejected update changed the state: map %+v, version %d", initial, store.Version())
		}
	})
}
