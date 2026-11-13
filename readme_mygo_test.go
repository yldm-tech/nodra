//go:build !nodra_nomygo

package nodra_test

import (
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/yldm-tech/nodra"
)

type readmeMyGoState struct{ Count int }

// readmeMyGo is the README MyGo snippet compiled as code. TestREADMEMyGoSnippet checks that README.md shows exactly the lines between the markers.
func readmeMyGo(store *nodra.Store[readmeMyGoState], view func(*ui.Context)) {
	type State = readmeMyGoState
	// README MyGo snippet: begin
	mygo.App.WhenReady(func() {
		window := mygo.NewWindow(mygo.WindowOptions{Title: "Counter", Content: ui.View(view)})
		// Redraws stop when the window closes. Keep the returned stop only to end them sooner: a defer here would end them as soon as WhenReady returns.
		nodra.InvalidateSelector(store, window, func(s State) int {
			return s.Count
		})
	})
	// README MyGo snippet: end
}

// TestREADMEMyGoSnippet keeps the README MyGo snippet compiling and identical to readmeMyGo. The snippet used to call InvalidateSelector on a window and defer its stop: outside WhenReady mygo.NewWindow panics, and inside it the deferred stop ends the redraws as soon as the callback returns.
func TestREADMEMyGoSnippet(t *testing.T) {
	_ = readmeMyGo
	source := readText(t, "readme_mygo_test.go")
	_, rest, found := strings.Cut(source, "\t// README MyGo snippet: begin\n")
	snippet, _, closed := strings.Cut(rest, "\t// README MyGo snippet: end\n")
	if !found || !closed {
		t.Fatal("readme_mygo_test.go: README MyGo snippet markers not found")
	}
	var want []string
	for line := range strings.SplitSeq(strings.TrimSuffix(snippet, "\n"), "\n") {
		want = append(want, strings.TrimPrefix(line, "\t"))
	}
	for _, line := range want {
		if strings.HasPrefix(strings.TrimSpace(line), "defer ") {
			t.Fatalf("README MyGo snippet defers %q inside WhenReady, which runs as soon as the callback returns", strings.TrimSpace(line))
		}
	}
	wantBlock := "```go\n" + strings.Join(want, "\n")
	var got string
	for _, block := range fencedBlocks(readText(t, "README.md")) {
		if strings.Contains(block, "nodra.InvalidateSelector(") {
			got = block
		}
	}
	if got != wantBlock {
		t.Errorf("README.md MyGo snippet differs from readmeMyGo:\n got:\n%s\nwant:\n%s", got, wantBlock)
	}
}
