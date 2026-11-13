package core

import (
	"errors"
	"sync/atomic"
	"testing"
)

// TestHistoryRejectsTravelAfterClose pins the closed guard in travel: Close empties both stacks, so without it Undo and Redo would report an empty stack instead of ErrClosed.
func TestHistoryRejectsTravelAfterClose(t *testing.T) {
	store := New(0)
	history := Track(store, 10)
	requireNoError(t, store.Set(1))
	requireNoError(t, store.Set(2))
	requireNoError(t, history.Undo())
	requireNoError(t, history.Close())
	if err := history.Undo(); !errors.Is(err, ErrClosed) {
		t.Fatalf("undo after close error = %v", err)
	}
	if err := history.Redo(); !errors.Is(err, ErrClosed) {
		t.Fatalf("redo after close error = %v", err)
	}
	if got := store.Get(); got != 1 {
		t.Fatalf("travel after close changed the store to %d", got)
	}
}

// TestHistoryRejectsGroupAfterClose pins the closed guard in BeginGroup.
func TestHistoryRejectsGroupAfterClose(t *testing.T) {
	store := New(0)
	history := Track(store, 10)
	requireNoError(t, history.Close())
	end, err := history.BeginGroup()
	if !errors.Is(err, ErrClosed) || end != nil {
		t.Fatalf("begin group after close = %v, end function %v", err, end != nil)
	}
	if state := history.Status().Get(); state.Grouped {
		t.Fatalf("status after a rejected group: %+v", state)
	}
}

// TestHistoryCloseDuringTravelCommit closes History while Undo or Redo is blocked in the store's commit clone, after travel chose its target and released the history lock. The committed callback then finds both stacks emptied by Close and must leave them alone instead of popping an entry that is gone.
func TestHistoryCloseDuringTravelCommit(t *testing.T) {
	for _, tc := range []struct {
		name   string
		undo   bool
		target int
	}{
		{name: "undo", undo: true, target: 1},
		{name: "redo", undo: false, target: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var block atomic.Bool
			entered := make(chan struct{})
			release := make(chan struct{})
			// The commit clone of the travel target blocks once: travel copies the current value first and the target second, so the first clone of the target is the commit clone.
			store := NewWithClone(0, func(v int) int {
				if v == tc.target && block.CompareAndSwap(true, false) {
					close(entered)
					<-release
				}
				return v
			})
			history := Track(store, 10)
			requireNoError(t, store.Set(1))
			requireNoError(t, store.Set(2))
			if !tc.undo {
				requireNoError(t, history.Undo())
			}
			// Close marks History closed and empties its stacks under the history lock, then blocks unsubscribing until travel releases the store lock. This hook reports the moment between the two.
			closing := make(chan struct{})
			stop := history.stop
			history.stop = func() {
				close(closing)
				stop()
			}
			block.Store(true)
			traveled := make(chan any, 1)
			go func() {
				var err error
				defer func() {
					if p := recover(); p != nil {
						traveled <- p
						return
					}
					traveled <- err
				}()
				if tc.undo {
					err = history.Undo()
				} else {
					err = history.Redo()
				}
			}()
			<-entered
			closed := make(chan error, 1)
			go func() { closed <- history.Close() }()
			<-closing
			close(release)
			if result := <-traveled; result != nil {
				t.Fatalf("travel concurrent with close returned or panicked with %v", result)
			}
			requireNoError(t, <-closed)
			if got := store.Get(); got != tc.target {
				t.Fatalf("store = %d, want the committed target %d", got, tc.target)
			}
			state := history.Status().Get()
			if !state.Closed || state.UndoEntries != 0 || state.RedoEntries != 0 || state.Bytes != 0 || state.CanUndo || state.CanRedo {
				t.Fatalf("status after close during travel: %+v", state)
			}
		})
	}
}
