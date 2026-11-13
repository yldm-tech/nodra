package core

import (
	"errors"
	"reflect"
	"sort"
	"sync"
	"time"
)

// AtomicOp is one typed update in an AtomicGroup. Create operations with AtomicUpdate; the fields are intentionally private so an operation cannot be constructed without the store lock and rollback guarantees.
type AtomicOp struct {
	ptr         uintptr
	newPrepared func() atomicParticipant
	valid       bool
}

type atomicNotification struct {
	notify func(uint64)
}

type atomicParticipant struct {
	ptr     uintptr
	lock    func()
	unlock  func()
	prepare func() error
	commit  func(uint64) atomicNotification
}

// AtomicUpdate prepares one store update for an AtomicGroup. The callback sees a private candidate and may return an error to abort the complete group. It must not call Store methods directly; use the provided pointer instead.
func AtomicUpdate[T any](store *Store[T], fn func(*T) error) AtomicOp {
	if store == nil || fn == nil {
		return AtomicOp{}
	}
	return AtomicOp{
		ptr:   reflect.ValueOf(store).Pointer(),
		valid: true,
		newPrepared: func() atomicParticipant {
			var before, prepared T
			return atomicParticipant{
				ptr:    reflect.ValueOf(store).Pointer(),
				lock:   store.mu.Lock,
				unlock: store.mu.Unlock,
				prepare: func() error {
					if store.closed {
						return ErrClosed
					}
					before = store.value
					next := store.copy(before)
					if err := fn(&next); err != nil {
						return err
					}
					// Clone before any participant is committed. A clone panic therefore cannot leave an earlier participant visible.
					next = store.copy(next)
					prepared = next
					return nil
				},
				commit: func(transactionID uint64) atomicNotification {
					store.transactionID = transactionID
					store.value = prepared
					store.version++
					snapshot, listeners, mutations := store.publishLocked(before, MutationTransaction, "atomic-group")
					return atomicNotification{notify: func(id uint64) {
						store.notify(listeners, snapshot)
						store.notifyMutationWithTransaction(mutations, Mutation[T]{TransactionID: id, Type: MutationTransaction, Name: "atomic-group", Previous: before, Value: snapshot.Value, Version: snapshot.Version, At: time.Now()})
					}}
				},
			}
		},
	}
}

// AtomicGroup commits any number of typed Store updates as one visibility boundary. Every callback is prepared while all participating store locks are held; an error or panic leaves every store unchanged, with preparation panics returned as PanicError. Lock order is stable across calls, so groups built in opposite order cannot deadlock. Notifications run after all locks are released and may safely reenter their stores. A notification panic propagates unchanged without rolling back the committed stores.
func AtomicGroup(ops ...AtomicOp) (err error) {
	return atomicGroupNamed("atomic-group", ops...)
}

func atomicGroupNamed(_ string, ops ...AtomicOp) (err error) {
	if len(ops) == 0 {
		return errors.New("nodra: invalid atomic transaction")
	}
	locks := make([]atomicParticipant, len(ops))
	seen := make(map[uintptr]struct{}, len(ops))
	for i, op := range ops {
		if !op.valid || op.ptr == 0 || op.newPrepared == nil {
			return errors.New("nodra: invalid atomic transaction")
		}
		if _, ok := seen[op.ptr]; ok {
			return errors.New("nodra: duplicate store in atomic transaction")
		}
		seen[op.ptr] = struct{}{}
		locks[i] = op.newPrepared()
	}
	sort.Slice(locks, func(i, j int) bool { return locks[i].ptr < locks[j].ptr })
	storeTransactionMu.Lock()
	locked := 0
	transactionHeld := true
	defer func() {
		for i := locked - 1; i >= 0; i-- {
			locks[i].unlock()
		}
		if transactionHeld {
			storeTransactionMu.Unlock()
			if recovered := recover(); recovered != nil {
				err = &PanicError{Value: recovered}
			}
		}
	}()
	for i := range locks {
		locks[i].lock()
		locked++
	}
	for _, op := range locks {
		if err = op.prepare(); err != nil {
			return err
		}
	}
	transactionID := nextTransactionID.Add(1)
	notifications := make([]atomicNotification, 0, len(locks))
	for _, op := range locks {
		notifications = append(notifications, op.commit(transactionID))
	}
	for i := locked - 1; i >= 0; i-- {
		locks[i].unlock()
	}
	locked = 0
	storeTransactionMu.Unlock()
	transactionHeld = false
	dispatched := make([]func(), 0, len(notifications))
	for _, notification := range notifications {
		notification := notification
		dispatched = append(dispatched, func() { notification.notify(transactionID) })
	}
	runTransactionDispatch(transactionID, dispatched)
	return nil
}

// transactionDispatch keeps derived notifications inside one cross-store commit boundary. Notifications remain synchronous. Callbacks deferred by derived values are drained lowest height first, so every source of a derived value settles before the value itself, and in arrival order within one height. Once a drain finds nothing left the dispatch is done, and a callback deferred after that runs at once.
type transactionDispatch struct {
	mu      sync.Mutex
	pending []deferredTransaction
	done    bool
}

type deferredTransaction struct {
	height int
	fn     func()
}

// transactionNode is implemented by derived values. Their height is one more than their highest source; stores, read-only views and application observables have height zero.
type transactionNode interface {
	transactionHeight() int
}

func observableHeight[T any](source Observable[T]) int {
	if node, ok := source.(transactionNode); ok {
		return node.transactionHeight()
	}
	return 0
}

var transactionDispatches = struct {
	sync.Mutex
	active map[uint64]*transactionDispatch
}{active: make(map[uint64]*transactionDispatch)}

func beginTransactionDispatch(id uint64) *transactionDispatch {
	dispatch := &transactionDispatch{}
	transactionDispatches.Lock()
	transactionDispatches.active[id] = dispatch
	transactionDispatches.Unlock()
	return dispatch
}

// deferTransactionLookedUp, when a test sets it, runs between finding the active dispatch and queueing on it.
var deferTransactionLookedUp func(uint64)

// deferTransaction runs fn once transaction id has settled every queued callback of a lower height, or at once when the transaction is not being dispatched.
func deferTransaction(id uint64, height int, fn func()) {
	if id == 0 || fn == nil {
		if fn != nil {
			fn()
		}
		return
	}
	transactionDispatches.Lock()
	dispatch := transactionDispatches.active[id]
	transactionDispatches.Unlock()
	if deferTransactionLookedUp != nil {
		deferTransactionLookedUp(id)
	}
	if dispatch == nil {
		fn()
		return
	}
	dispatch.mu.Lock()
	if dispatch.done {
		// The dispatcher has drained its last callback and will not look at the queue again.
		dispatch.mu.Unlock()
		fn()
		return
	}
	dispatch.pending = append(dispatch.pending, deferredTransaction{height: height, fn: fn})
	dispatch.mu.Unlock()
}

// next removes the queued callbacks of the lowest height, in arrival order. When nothing is queued it marks the dispatch done and returns nil.
func (dispatch *transactionDispatch) next() []func() {
	dispatch.mu.Lock()
	defer dispatch.mu.Unlock()
	if len(dispatch.pending) == 0 {
		dispatch.done = true
		return nil
	}
	lowest := dispatch.pending[0].height
	for _, deferred := range dispatch.pending[1:] {
		lowest = min(lowest, deferred.height)
	}
	var run []func()
	rest := dispatch.pending[:0]
	for _, deferred := range dispatch.pending {
		if deferred.height == lowest {
			run = append(run, deferred.fn)
		} else {
			rest = append(rest, deferred)
		}
	}
	clear(dispatch.pending[len(rest):])
	dispatch.pending = rest
	return run
}

func runTransactionDispatch(id uint64, notifications []func()) {
	dispatch := beginTransactionDispatch(id)
	defer func() {
		// After a notification panic nothing drains the queue any more, so later callbacks must not wait on it.
		dispatch.mu.Lock()
		dispatch.done = true
		dispatch.mu.Unlock()
		transactionDispatches.Lock()
		delete(transactionDispatches.active, id)
		transactionDispatches.Unlock()
	}()
	for _, notification := range notifications {
		notification()
	}
	for run := dispatch.next(); run != nil; run = dispatch.next() {
		for _, callback := range run {
			callback()
		}
	}
}
