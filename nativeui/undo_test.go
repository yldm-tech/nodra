package nativeui_test

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/yldm-tech/nodra"
	"github.com/yldm-tech/nodra/nativeui"
)

type doc struct {
	Count int
}

// app is a window's view over a tracked store, with a text input whose text stays out of the store.
type app struct {
	store   *nodra.Store[doc]
	history *nodra.History[doc]
	draft   string
}

func newApp(t *testing.T) *app {
	t.Helper()
	store := nodra.New(doc{})
	history := nodra.Track(store, 10)
	t.Cleanup(func() {
		_ = history.Close()
		_ = store.Close()
	})
	return &app{store: store, history: history}
}

func (a *app) view(c *ui.Context) {
	nativeui.UndoShortcuts(c, a.history)
	ui.Column(c).Fill().Padding(12).Gap(8).Children(func() {
		ui.Text(c, fmt.Sprintf("Count: %d", a.store.Get().Count))
		ui.TextInput(c, &a.draft).Label("Draft")
	})
}

func (a *app) set(t *testing.T, n int) {
	t.Helper()
	if err := a.store.Update(func(d *doc) { d.Count = n }); err != nil {
		t.Fatal(err)
	}
}

func (a *app) count() int { return a.store.Get().Count }

func TestTheShortcutsUndoAndRedo(t *testing.T) {
	a := newApp(t)
	a.set(t, 1)
	a.set(t, 2)
	tester := ui.NewTester(a.view, 320, 200)

	tester.Key(ui.Cmd, ui.KeyZ)
	if got := a.count(); got != 1 {
		t.Fatalf("after Undo the count is %d, want 1", got)
	}
	tester.Frame()
	tester.Key(ui.Cmd, ui.KeyZ)
	if got := a.count(); got != 0 {
		t.Fatalf("after a second Undo the count is %d, want 0", got)
	}
	tester.Frame()
	tester.Key(ui.Cmd|ui.Shift, ui.KeyZ)
	if got := a.count(); got != 1 {
		t.Fatalf("after Redo the count is %d, want 1", got)
	}
	if runtime.GOOS == "windows" {
		tester.Frame()
		tester.Key(ui.Ctrl, ui.KeyY)
		if got := a.count(); got != 2 {
			t.Fatalf("after Ctrl+Y the count is %d, want 2", got)
		}
	}
}

func TestNothingIsRegisteredWhileHistoryCannotAct(t *testing.T) {
	a := newApp(t)
	tester := ui.NewTester(a.view, 320, 200)
	// Nothing to undo or redo: the keys change nothing and nothing fails.
	tester.Key(ui.Cmd, ui.KeyZ)
	tester.Key(ui.Cmd|ui.Shift, ui.KeyZ)
	if got := a.count(); got != 0 {
		t.Fatalf("the count is %d", got)
	}
	// A change made after the frame was built registers Undo in the next one.
	a.set(t, 5)
	tester.Frame()
	tester.Key(ui.Cmd, ui.KeyZ)
	if got := a.count(); got != 0 {
		t.Fatalf("after Undo the count is %d, want 0", got)
	}
}

func TestAFocusedTextInputKeepsItsOwnUndo(t *testing.T) {
	a := newApp(t)
	a.set(t, 7)
	tester := ui.NewTester(a.view, 320, 200)
	if err := tester.Click("Draft"); err != nil {
		t.Fatal(err)
	}
	tester.Type("abc")
	tester.Frame()
	if a.draft != "abc" {
		t.Fatalf("typing into the text input gave %q", a.draft)
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	tester.Frame()
	if got := a.count(); got != 7 {
		t.Fatalf("Undo in a text input changed the store to %d", got)
	}
	if a.draft == "abc" {
		t.Fatalf("Undo in a text input left its text %q", a.draft)
	}
}

func TestTheEditMenusUndoActsOnHistoryOnMacOS(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("on Linux and Windows the keys reach the shortcuts before the menu")
	}
	a := newApp(t)
	a.set(t, 3)
	tester := ui.NewTester(a.view, 320, 200)
	tester.Command("undo")
	tester.Frame()
	if got := a.count(); got != 0 {
		t.Fatalf("after the Edit menu's Undo the count is %d, want 0", got)
	}
	tester.Command("redo")
	tester.Frame()
	if got := a.count(); got != 3 {
		t.Fatalf("after the Edit menu's Redo the count is %d, want 3", got)
	}
}

func TestNilArgumentsDoNothing(t *testing.T) {
	tester := ui.NewTester(func(c *ui.Context) {
		nativeui.UndoShortcuts[doc](c, nil)
		ui.Text(c, "ok")
	}, 100, 100)
	tester.Key(ui.Cmd, ui.KeyZ)
	nativeui.InvalidateHistory[doc](nil, nil)()
	a := newApp(t)
	nativeui.InvalidateHistory(a.history, nil)()
}
