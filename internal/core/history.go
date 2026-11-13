package core

import (
	"errors"
	"sync"
)

// ErrNoUndo and ErrNoRedo report an empty history stack.
var (
	ErrNoUndo = errors.New("nodra: nothing to undo")
	ErrNoRedo = errors.New("nodra: nothing to redo")
)

// ErrHistoryBatch reports travel, or History.Clear, without an independent publication boundary: inside an open batch of the source store.
var ErrHistoryBatch = errors.New("nodra: cannot undo or redo inside an open batch")

var ErrHistoryGroup = errors.New("nodra: cannot undo or redo inside an open history group")
var ErrInvalidHistory = errors.New("nodra: invalid history size configuration")

// HistoryOptions configures recording. Equals excludes changes to irrelevant fields. Restore merges recorded fields into current state so ignored fields survive travel. Both functions must be pure, must not mutate arguments, and must not reenter the Store or History; they can run while those locks are held. With nil callbacks, every publication records and travel replaces the whole value.
type HistoryOptions[T any] struct {
	Limit int
	// MaxBytes budgets estimated undo and redo payloads together; zero disables sizing. Positive budgets require Size. The current value, Store state, shared allocations and metadata are outside this estimate.
	MaxBytes int64
	// Size estimates one retained value. It must return a nonnegative value and follow Equals/Restore purity and lock rules. Invalid configurations or negative estimates panic with ErrInvalidHistory.
	Size    func(T) int64
	Equals  func(previous, next T) bool
	Restore func(current, recorded T) T
}

// History records successful store snapshots and provides bounded undo/redo.
type History[T any] struct {
	store        *Store[T]
	mu           sync.Mutex
	undo         []historyEntry[T]
	redo         []historyEntry[T]
	current      T
	limit        int
	version      uint64
	stop         func()
	closed       bool
	options      HistoryOptions[T]
	groups       int
	grouped      bool
	bytes        uint64
	currentBytes uint64
	dropped      uint64
	revision     uint64
	state        *Store[HistoryState]
}

// Track starts recording changes to store. limit is the maximum number of undo entries; values below one disable history recording. It is TrackWith with only Limit set, and the same ownership rule applies: closing store does not close History.
func Track[T any](store *Store[T], limit int) *History[T] {
	return TrackWith(store, HistoryOptions[T]{Limit: limit})
}

// TrackWith starts recording changes to store with options. Closing store does not close History: Undo and Redo then return ErrClosed and keep their stacks, CanUndo, CanRedo and Status keep reporting those stacks, and Status stays open, so close History when the store's owner is done with it.
func TrackWith[T any](store *Store[T], options HistoryOptions[T]) *History[T] {
	if options.MaxBytes < 0 || (options.MaxBytes > 0 && options.Size == nil) {
		panic(ErrInvalidHistory)
	}
	h := &History[T]{store: store, limit: options.Limit, options: options, state: New(HistoryState{MaxBytes: options.MaxBytes})}
	if store == nil || options.Limit < 1 {
		h.closed = true
		h.revision++
		h.publishStatus()
		_ = h.state.Close()
		return h
	}
	initial := store.Snapshot()
	h.current, h.version = initial.Value, initial.Version
	h.currentBytes = h.estimate(initial.Value)
	h.stop = store.Subscribe(h.observe)
	initialized := false
	defer func() {
		if !initialized {
			h.stop()
		}
	}()
	// Changes during registration may coalesce into one entry.
	h.observe(store.Snapshot())
	initialized = true
	return h
}

func (h *History[T]) observe(snapshot Snapshot[T]) {
	func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.observeLocked(snapshot)
	}()
	h.publishStatus()
}

func (h *History[T]) observeLocked(snapshot Snapshot[T]) {
	if h.closed || snapshot.Version <= h.version {
		return
	}
	size := h.estimate(snapshot.Value)
	if h.options.Equals == nil || !h.options.Equals(h.current, snapshot.Value) {
		if h.groups == 0 || !h.grouped {
			h.undo = append(h.undo, historyEntry[T]{h.current, h.currentBytes})
			h.bytes += h.currentBytes
			h.grouped = h.groups > 0
		}
		for _, entry := range h.redo {
			h.bytes -= entry.bytes
		}
		h.redo = nil
		h.trimLocked()
	}
	h.current, h.version, h.currentBytes = snapshot.Value, snapshot.Version, size
	h.revision++
}

// BeginGroup merges edits across frames into one undo step while still publishing every update. Groups nest; the returned end function is idempotent. Concurrent writers join the group. Undo and Redo are rejected until all groups end. Filtering still applies, so a group containing only ignored changes adds no step.
func (h *History[T]) BeginGroup() (func(), error) {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil, ErrClosed
	}
	if h.groups == 0 {
		h.grouped = false
	}
	h.groups++
	h.revision++
	h.mu.Unlock()
	h.publishStatus()
	var once sync.Once
	return func() {
		once.Do(func() {
			h.mu.Lock()
			h.groups--
			if h.groups == 0 {
				h.grouped = false
			}
			h.revision++
			h.mu.Unlock()
			h.publishStatus()
		})
	}, nil
}

// CanUndo reports whether Undo has a step to restore now. Like HistoryState.CanUndo, it is false while a group is open or History is closed. It does not observe an open Batch, where Undo returns ErrHistoryBatch, or a closed source Store, where Undo returns ErrClosed.
func (h *History[T]) CanUndo() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.canTravelLocked(h.undo)
}

// CanRedo reports whether Redo has a step to restore now, with the same rules as CanUndo and HistoryState.CanRedo.
func (h *History[T]) CanRedo() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.canTravelLocked(h.redo)
}

// canTravelLocked is the availability rule shared by CanUndo, CanRedo and the published HistoryState.
func (h *History[T]) canTravelLocked(stack []historyEntry[T]) bool {
	return !h.closed && h.groups == 0 && len(stack) > 0
}

// Undo restores the previous state.
func (h *History[T]) Undo() error { return h.travel(true) }

// Redo restores the next state.
func (h *History[T]) Redo() error { return h.travel(false) }

// Clear drops every undo and redo step and makes the store's current value the new baseline, so that after Open, New or Revert replaces the document, Undo cannot restore the previous one. Call it after the write that replaced the value; later writes are recorded against that value as usual. Status publishes the empty stacks, so InvalidateHistory and other Status subscribers follow without rebinding; Dropped does not count the cleared steps. Clear returns ErrClosed after Close or for a History that records nothing, ErrHistoryGroup while a group is open, and ErrHistoryBatch inside an open Batch of the source store, where the new value has not been published yet; each leaves the stacks unchanged. It still clears after the source store is closed. Like Undo, it must not be called from a Store update callback, a clone function or a HistoryOptions callback.
func (h *History[T]) Clear() error {
	if h.store == nil {
		return ErrClosed
	}
	defer h.publishStatus()
	// Hold the source lock for reading, in the store-then-history order travel uses, so that Clear is ordered with every commit: a write committed before Clear is part of the new baseline and its pending notification is obsolete, and a later write is recorded against it.
	h.store.mu.RLock()
	defer h.store.mu.RUnlock()
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return ErrClosed
	}
	if h.groups > 0 {
		return ErrHistoryGroup
	}
	if h.store.batch > 0 {
		return ErrHistoryBatch
	}
	current := h.store.copy(h.store.value)
	currentBytes := h.estimate(current)
	// Dropping the slices releases their backing arrays, so no cleared entry keeps an old state reachable.
	h.undo, h.redo, h.bytes = nil, nil, 0
	h.current, h.version, h.currentBytes = current, h.store.version, currentBytes
	h.revision++
	return nil
}

func (h *History[T]) travel(undo bool) error {
	if h.store == nil {
		return ErrClosed
	}
	defer h.publishStatus()
	var target T
	var targetBytes uint64
	// Choose and commit the history step under the source write lock. Stack changes happen only after the store's commit clone has succeeded, before any notifications can reenter history or a concurrent writer can commit.
	return h.store.updateCommitted("", MutationDirect, func(value *T) error {
		current := Snapshot[T]{Value: *value, Version: h.store.version, TransactionID: h.store.transactionID}
		h.mu.Lock()
		defer h.mu.Unlock()
		if h.closed {
			return ErrClosed
		}
		if h.store.batch > 0 {
			return ErrHistoryBatch
		}
		if h.groups > 0 {
			return ErrHistoryGroup
		}
		h.observeLocked(current)
		stack := h.undo
		if !undo {
			stack = h.redo
		}
		if len(stack) == 0 {
			if undo {
				return ErrNoUndo
			}
			return ErrNoRedo
		}
		target = stack[len(stack)-1].value
		if h.options.Restore != nil {
			target = h.options.Restore(*value, target)
		}
		targetBytes = h.estimate(target)
		*value = target
		return nil
	}, func(version uint64) {
		h.mu.Lock()
		defer h.mu.Unlock()
		if h.closed {
			return
		}
		var zero historyEntry[T]
		if undo {
			h.bytes -= h.undo[len(h.undo)-1].bytes
			h.undo[len(h.undo)-1] = zero
			h.undo = h.undo[:len(h.undo)-1]
			h.redo = append(h.redo, historyEntry[T]{h.current, h.currentBytes})
		} else {
			h.bytes -= h.redo[len(h.redo)-1].bytes
			h.redo[len(h.redo)-1] = zero
			h.redo = h.redo[:len(h.redo)-1]
			h.undo = append(h.undo, historyEntry[T]{h.current, h.currentBytes})
		}
		h.bytes += h.currentBytes
		h.trimLocked()
		h.current, h.version, h.currentBytes = target, version, targetBytes
		h.revision++
	})
}

// Close stops recording future changes. It does not close the underlying store.
func (h *History[T]) Close() error {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return ErrClosed
	}
	h.closed = true
	h.undo = nil
	h.redo = nil
	h.bytes = 0
	h.revision++
	h.mu.Unlock()
	if h.stop != nil {
		h.stop()
	}
	if h.state != nil {
		defer func() { _ = h.state.Close() }()
	}
	h.publishStatus()
	return nil
}

type historyEntry[T any] struct {
	value T
	bytes uint64
}

// HistoryState describes history availability and estimated retained payloads. Revision orders status publications independently of source versions. Bytes excludes the current value and is an application estimate, not process heap usage. CanUndo/CanRedo are false while grouped or closed.
type HistoryState struct {
	Revision    uint64
	UndoEntries int
	RedoEntries int
	Bytes       int64
	MaxBytes    int64
	Dropped     uint64
	Grouped     bool
	Closed      bool
	CanUndo     bool
	CanRedo     bool
}

func (h *History[T]) Status() ReadOnly[HistoryState] { return h.state.ReadOnly() }

func (h *History[T]) estimate(value T) uint64 {
	if h.options.MaxBytes == 0 {
		return 0
	}
	size := h.options.Size(value)
	if size < 0 {
		panic(ErrInvalidHistory)
	}
	return uint64(size)
}

func (h *History[T]) trimLocked() {
	for len(h.undo) > h.limit || (h.options.MaxBytes > 0 && h.bytes > uint64(h.options.MaxBytes)) {
		stack := &h.undo
		if len(*stack) == 0 {
			stack = &h.redo
		}
		if len(*stack) == 0 {
			break
		}
		h.bytes -= (*stack)[0].bytes
		var zero historyEntry[T]
		(*stack)[0] = zero
		*stack = (*stack)[1:]
		h.dropped++
	}
}

var errHistoryStatusUnchanged = errors.New("nodra: history status unchanged")

// Status subscribers run after releasing both the history lock and source write lock. A delayed publisher cannot replace a newer status or revive a closed status Store.
func (h *History[T]) publishStatus() {
	if h.state == nil {
		return
	}
	h.mu.Lock()
	state := HistoryState{Revision: h.revision, UndoEntries: len(h.undo), RedoEntries: len(h.redo), Bytes: int64(h.bytes), MaxBytes: h.options.MaxBytes, Dropped: h.dropped, Grouped: h.groups > 0, Closed: h.closed, CanUndo: h.canTravelLocked(h.undo), CanRedo: h.canTravelLocked(h.redo)}
	h.mu.Unlock()
	_ = h.state.UpdateErr(func(current *HistoryState) error {
		if current.Revision >= state.Revision {
			return errHistoryStatusUnchanged
		}
		*current = state
		return nil
	})
}
