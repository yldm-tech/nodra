package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/yldm-tech/nodra"
)

func TestTheNativeViewAndThePageShareTheStore(t *testing.T) {
	store := nodra.New(appState{Token: "secret"})
	defer func() { _ = store.Close() }()
	b, err := newBridge(store)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	ctx := context.Background()

	// The native button changes what the page reads, and the page never reads the token.
	native := ui.NewTester(func(c *ui.Context) { view(c, store) }, 360, 260)
	if err := native.Click("+1"); err != nil {
		t.Fatal(err)
	}
	frame, err := b.Get(ctx, "counter")
	if err != nil {
		t.Fatal(err)
	}
	if string(frame.Value) != `{"count":1}` {
		t.Fatalf("the page reads %s", frame.Value)
	}

	// The page's actions change what the native view shows.
	if err := b.Dispatch(ctx, "counter/add", json.RawMessage(`{"by":4}`)); err != nil {
		t.Fatal(err)
	}
	native.Frame()
	if !native.HasText("Count: 5") {
		t.Fatalf("the native view shows %q", native.Texts())
	}
	if err := b.Dispatch(ctx, "counter/reset", nil); err != nil || store.Get().Count != 0 {
		t.Fatalf("reset: %v, count %d", err, store.Get().Count)
	}
	if got := store.Get().Token; got != "secret" {
		t.Fatalf("an action replaced the token: %q", got)
	}
}
