package core

import (
	"errors"
	"testing"
)

func TestHistoryUndoRedo(t *testing.T) {
	store := New(0)
	history := Track(store, 10)
	requireNoError(t, store.Set(1))
	requireNoError(t, store.Set(2))
	if !history.CanUndo() || history.CanRedo() {
		t.Fatal("history flags after updates")
	}
	if err := history.Undo(); err != nil || store.Get() != 1 {
		t.Fatalf("undo value=%d err=%v", store.Get(), err)
	}
	if err := history.Redo(); err != nil || store.Get() != 2 {
		t.Fatalf("redo value=%d err=%v", store.Get(), err)
	}
	requireNoError(t, store.Set(3))
	if history.CanRedo() {
		t.Fatal("redo should clear after a new update")
	}
	requireNoError(t, history.Close())
}

func TestHistoryCanUndoRedoFollowGroups(t *testing.T) {
	store := New(0)
	history := Track(store, 10)
	defer func() { _ = history.Close() }()
	requireNoError(t, store.Set(1))
	requireNoError(t, store.Set(2))
	requireNoError(t, history.Undo())
	if !history.CanUndo() || !history.CanRedo() {
		t.Fatal("history flags before group")
	}
	end, err := history.BeginGroup()
	requireNoError(t, err)
	status := history.Status().Get()
	if history.CanUndo() || history.CanRedo() || status.CanUndo || status.CanRedo {
		t.Fatalf("CanUndo=%v CanRedo=%v inside a group; status %+v", history.CanUndo(), history.CanRedo(), status)
	}
	if err := history.Undo(); !errors.Is(err, ErrHistoryGroup) {
		t.Fatalf("undo inside group error = %v", err)
	}
	if err := history.Redo(); !errors.Is(err, ErrHistoryGroup) {
		t.Fatalf("redo inside group error = %v", err)
	}
	end()
	if !history.CanUndo() || !history.CanRedo() {
		t.Fatal("history flags after group")
	}
}

func TestHistoryOutlivesClosedSourceStore(t *testing.T) {
	store := New(0)
	history := Track(store, 10)
	requireNoError(t, store.Set(1))
	requireNoError(t, store.Set(2))
	requireNoError(t, history.Undo())
	requireNoError(t, store.Close())
	if err := history.Undo(); !errors.Is(err, ErrClosed) {
		t.Fatalf("undo after store close error = %v", err)
	}
	if err := history.Redo(); !errors.Is(err, ErrClosed) {
		t.Fatalf("redo after store close error = %v", err)
	}
	status := history.Status()
	if state := status.Get(); !history.CanUndo() || !history.CanRedo() || !state.CanUndo || !state.CanRedo || state.Closed {
		t.Fatalf("CanUndo=%v CanRedo=%v after store close; status %+v", history.CanUndo(), history.CanRedo(), state)
	}
	select {
	case <-status.Done():
		t.Fatal("closing the source store closed the history status")
	default:
	}
	requireNoError(t, history.Close())
	select {
	case <-status.Done():
	default:
		t.Fatal("closing history left its status open")
	}
	if state := status.Get(); !state.Closed || state.CanUndo || state.CanRedo {
		t.Fatalf("status after history close %+v", state)
	}
}

func TestHistoryNestedGroupJoinsTheOuterStep(t *testing.T) {
	store := New(0)
	history := Track(store, 10)
	defer func() { _ = history.Close() }()
	end, err := history.BeginGroup()
	requireNoError(t, err)
	requireNoError(t, store.Set(1))
	// The inner group opens after the outer group has recorded a step, so it must keep adding to that step rather than start another.
	inner, err := history.BeginGroup()
	requireNoError(t, err)
	requireNoError(t, store.Set(2))
	inner()
	end()
	requireNoError(t, history.Undo())
	if got := store.Get(); got != 0 {
		t.Fatalf("undo restored %d, want 0", got)
	}
	if history.CanUndo() {
		t.Fatal("the nested group recorded a second undo step")
	}
}
