package core

import (
	"errors"
	"testing"
)

type clearDoc struct{ Path, Text string }

// TestHistoryClearRebaselinesAfterReplacingTheDocument is the File > Open case: after the store is given another document, Clear keeps Undo from restoring the previous one, and later edits are recorded against the new document.
func TestHistoryClearRebaselinesAfterReplacingTheDocument(t *testing.T) {
	store := New(clearDoc{Path: "a.txt", Text: "A"})
	history := Track(store, 10)
	defer func() { _ = history.Close() }()
	requireNoError(t, store.Update(func(d *clearDoc) { d.Text = "A edited" }))
	requireNoError(t, history.Undo())
	requireNoError(t, store.Set(clearDoc{Path: "b.txt", Text: "B"}))
	requireNoError(t, history.Clear())
	if history.CanUndo() || history.CanRedo() {
		t.Fatalf("CanUndo=%v CanRedo=%v after Clear", history.CanUndo(), history.CanRedo())
	}
	if err := history.Undo(); !errors.Is(err, ErrNoUndo) {
		t.Fatalf("undo after clear error = %v", err)
	}
	if err := history.Redo(); !errors.Is(err, ErrNoRedo) {
		t.Fatalf("redo after clear error = %v", err)
	}
	if got := store.Get(); got.Path != "b.txt" || got.Text != "B" {
		t.Fatalf("store after rejected travel = %+v", got)
	}
	requireNoError(t, store.Update(func(d *clearDoc) { d.Text = "B edited" }))
	requireNoError(t, history.Undo())
	if got := store.Get(); got.Path != "b.txt" || got.Text != "B" {
		t.Fatalf("undo after clear restored %+v, want the new baseline", got)
	}
	if err := history.Undo(); !errors.Is(err, ErrNoUndo) {
		t.Fatalf("second undo after clear error = %v", err)
	}
	requireNoError(t, history.Redo())
	if got := store.Get(); got.Text != "B edited" {
		t.Fatalf("redo after clear restored %+v", got)
	}
}

// TestHistoryClearPublishesStatus checks that Status subscribers, such as nativeui.InvalidateHistory, see the emptied stacks without rebinding, and that the byte budget starts over.
func TestHistoryClearPublishesStatus(t *testing.T) {
	store := New(0)
	history := TrackWith(store, HistoryOptions[int]{Limit: 10, MaxBytes: 100, Size: func(int) int64 { return 10 }})
	defer func() { _ = history.Close() }()
	requireNoError(t, store.Set(1))
	requireNoError(t, store.Set(2))
	requireNoError(t, history.Undo())
	before := history.Status().Get()
	if !before.CanUndo || !before.CanRedo || before.Bytes != 20 {
		t.Fatalf("status before clear: %+v", before)
	}
	var seen []HistoryState
	stop := history.Status().Subscribe(func(s Snapshot[HistoryState]) { seen = append(seen, s.Value) })
	defer stop()
	requireNoError(t, history.Clear())
	if len(seen) != 1 {
		t.Fatalf("clear published %d status updates: %+v", len(seen), seen)
	}
	after := seen[0]
	if after.Revision <= before.Revision || after.UndoEntries != 0 || after.RedoEntries != 0 || after.Bytes != 0 || after.CanUndo || after.CanRedo || after.Closed || after.Dropped != before.Dropped {
		t.Fatalf("status after clear: %+v, before %+v", after, before)
	}
	requireNoError(t, store.Set(3))
	if state := history.Status().Get(); state.UndoEntries != 1 || state.Bytes != 10 || !state.CanUndo {
		t.Fatalf("status after recording again: %+v", state)
	}
}

// TestHistoryClearIgnoresTheReplacingWritesNotification covers a Clear that runs between a write's commit and History's notification of it: the write is already part of the new baseline, so the late notification must not record the previous value, while the next write is recorded against the baseline.
func TestHistoryClearIgnoresTheReplacingWritesNotification(t *testing.T) {
	store := New(0)
	history := Track(store, 10)
	defer func() { _ = history.Close() }()
	requireNoError(t, store.Set(1))
	// Hold History's notifications back so that Clear deterministically runs before they arrive.
	history.stop()
	var held []Snapshot[int]
	stop := store.Subscribe(func(s Snapshot[int]) { held = append(held, s) })
	defer stop()
	requireNoError(t, store.Set(2))
	requireNoError(t, history.Clear())
	history.observe(held[0])
	if history.CanUndo() {
		t.Fatalf("history recorded a step for the write Clear rebaselined to: %+v", history.Status().Get())
	}
	requireNoError(t, store.Set(3))
	history.observe(held[1])
	requireNoError(t, history.Undo())
	if got := store.Get(); got != 2 {
		t.Fatalf("undo after clear restored %d, want the baseline 2", got)
	}
}

func TestHistoryClearRejections(t *testing.T) {
	t.Run("group", func(t *testing.T) {
		store := New(0)
		history := Track(store, 10)
		defer func() { _ = history.Close() }()
		requireNoError(t, store.Set(1))
		end, err := history.BeginGroup()
		requireNoError(t, err)
		if err := history.Clear(); !errors.Is(err, ErrHistoryGroup) {
			t.Fatalf("clear inside group error = %v", err)
		}
		end()
		if !history.CanUndo() {
			t.Fatal("a rejected clear dropped the undo stack")
		}
	})
	t.Run("batch", func(t *testing.T) {
		store := New(0)
		history := Track(store, 10)
		defer func() { _ = history.Close() }()
		requireNoError(t, store.Set(1))
		requireNoError(t, store.BatchErr(func() error {
			requireNoError(t, store.Set(2))
			if err := history.Clear(); !errors.Is(err, ErrHistoryBatch) {
				t.Fatalf("clear inside batch error = %v", err)
			}
			return nil
		}))
		if state := history.Status().Get(); state.UndoEntries != 2 {
			t.Fatalf("a rejected clear changed the stacks: %+v", state)
		}
	})
	t.Run("closed", func(t *testing.T) {
		store := New(0)
		history := Track(store, 10)
		requireNoError(t, history.Close())
		if err := history.Clear(); !errors.Is(err, ErrClosed) {
			t.Fatalf("clear after close error = %v", err)
		}
		if state := history.Status().Get(); !state.Closed {
			t.Fatalf("status after a rejected clear: %+v", state)
		}
	})
	t.Run("disabled", func(t *testing.T) {
		for _, history := range []*History[int]{Track[int](nil, 10), Track(New(0), 0)} {
			if err := history.Clear(); !errors.Is(err, ErrClosed) {
				t.Fatalf("clear of a disabled history error = %v", err)
			}
		}
	})
}

// TestHistoryClearAfterSourceStoreCloses checks that Clear still empties the stacks once the source Store is closed, where travel returns ErrClosed, so an app can disable Undo for a document it has shut.
func TestHistoryClearAfterSourceStoreCloses(t *testing.T) {
	store := New(0)
	history := Track(store, 10)
	defer func() { _ = history.Close() }()
	requireNoError(t, store.Set(1))
	requireNoError(t, store.Close())
	requireNoError(t, history.Clear())
	if state := history.Status().Get(); state.UndoEntries != 0 || state.CanUndo || state.Closed {
		t.Fatalf("status after clear of a closed source: %+v", state)
	}
}
