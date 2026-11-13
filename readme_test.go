package nodra_test

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yldm-tech/nodra"
)

// runREADMEPersistence is the README persistence snippet compiled as code. TestREADMEPersistenceSnippet checks that README.md shows exactly the lines between the markers. State is a type parameter so the snippet can name the application state type without declaring one in this package.
func runREADMEPersistence[State any](ctx, shutdownCtx context.Context, store *nodra.Store[State]) error {
	// README persistence snippet: begin
	persistence := nodra.NewPersistence(store, nodra.FileStorage{Path: "state.json", Backup: true}, nodra.JSONCodec[State]{})
	// A missing file is a first start. Stop on any other load error: autosave would overwrite the file.
	if err := persistence.Load(ctx); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	writer, err := persistence.AutoSaveAsync(ctx, nodra.AsyncOptions{Debounce: 100 * time.Millisecond})
	if err != nil {
		return err
	}
	defer func() {
		if err := writer.Close(shutdownCtx); err != nil {
			// The store still owns the data: retry with persistence.Save using a fresh context, or report the error.
			log.Printf("save state: %v", err)
		}
	}()
	// README persistence snippet: end
	return nil
}

// fencedBlocks returns the fenced code blocks of a markdown document in order, each with its info string.
func fencedBlocks(markdown string) []string {
	var blocks []string
	var current []string
	inside := false
	for line := range strings.SplitSeq(markdown, "\n") {
		if strings.HasPrefix(line, "```") {
			if inside {
				blocks = append(blocks, strings.Join(current, "\n"))
				current = nil
			} else {
				current = []string{line}
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

// TestREADMEPersistenceSnippet runs the README persistence snippet and keeps README.md identical to it. The snippet used to discard the AutoSaveAsync error, so a canceled startup context or a closed store made the deferred Close panic on a nil writer.
func TestREADMEPersistenceSnippet(t *testing.T) {
	source := readText(t, "readme_test.go")
	_, rest, found := strings.Cut(source, "\t// README persistence snippet: begin\n")
	snippet, _, closed := strings.Cut(rest, "\t// README persistence snippet: end\n")
	if !found || !closed {
		t.Fatal("readme_test.go: README persistence snippet markers not found")
	}
	var want []string
	for line := range strings.SplitSeq(strings.TrimSuffix(snippet, "\n"), "\n") {
		want = append(want, strings.TrimPrefix(line, "\t"))
	}
	var got string
	for _, block := range fencedBlocks(readText(t, "README.md")) {
		if strings.Contains(block, "AutoSaveAsync") {
			got = block
		}
	}
	if wantBlock := "```go\n" + strings.Join(want, "\n"); got != wantBlock {
		t.Errorf("README.md persistence snippet differs from runREADMEPersistence:\n got:\n%s\nwant:\n%s", got, wantBlock)
	}

	t.Chdir(t.TempDir())
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	closedStore := nodra.New(struct{ Count int }{})
	_ = closedStore.Close()
	for _, tc := range []struct {
		name  string
		ctx   context.Context
		store *nodra.Store[struct{ Count int }]
		want  error
	}{
		{"canceled startup context", canceled, nodra.New(struct{ Count int }{}), context.Canceled},
		{"closed store", context.Background(), closedStore, nodra.ErrClosed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("README persistence snippet panicked: %v", r)
				}
			}()
			if err := runREADMEPersistence(tc.ctx, context.Background(), tc.store); !errors.Is(err, tc.want) {
				t.Fatalf("README persistence snippet returned %v, want %v", err, tc.want)
			}
		})
	}
	t.Run("invalid startup file", func(t *testing.T) {
		if err := os.WriteFile("state.json", []byte("not json"), 0o600); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove("state.json") }()
		var loadErr *nodra.LoadError
		if err := runREADMEPersistence(context.Background(), context.Background(), nodra.New(struct{ Count int }{})); !errors.As(err, &loadErr) || loadErr.Stage != "decode" {
			t.Fatalf("README persistence snippet returned %v, want a decode LoadError", err)
		}
		if data := readText(t, "state.json"); data != "not json" {
			t.Fatalf("state.json = %q, want the invalid file left untouched", data)
		}
	})
	t.Run("saves on return", func(t *testing.T) {
		store := nodra.New(struct{ Count int }{Count: 7})
		defer func() { _ = store.Close() }()
		if err := runREADMEPersistence(context.Background(), context.Background(), store); err != nil {
			t.Fatal(err)
		}
		var saved struct{ Count int }
		if err := json.Unmarshal([]byte(readText(t, "state.json")), &saved); err != nil || saved.Count != 7 {
			t.Fatalf("state.json holds %+v (%v), want the store value", saved, err)
		}
	})
}
