package nodra

import "sync"

// transactionDispatch keeps derived notifications inside one cross-store commit boundary. Notifications remain synchronous; callbacks registered by derived sources are drained before the boundary is considered complete.
type transactionDispatch struct {
	mu      sync.Mutex
	pending []func()
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

func deferTransaction(id uint64, fn func()) {
	if id == 0 || fn == nil {
		if fn != nil {
			fn()
		}
		return
	}
	transactionDispatches.Lock()
	dispatch := transactionDispatches.active[id]
	transactionDispatches.Unlock()
	if dispatch == nil {
		fn()
		return
	}
	dispatch.mu.Lock()
	dispatch.pending = append(dispatch.pending, fn)
	dispatch.mu.Unlock()
}

func runTransactionDispatch(id uint64, notifications []func()) {
	dispatch := beginTransactionDispatch(id)
	defer func() {
		transactionDispatches.Lock()
		delete(transactionDispatches.active, id)
		transactionDispatches.Unlock()
	}()
	for _, notification := range notifications {
		notification()
	}
	for {
		dispatch.mu.Lock()
		pending := dispatch.pending
		dispatch.pending = nil
		dispatch.mu.Unlock()
		if len(pending) == 0 {
			return
		}
		for _, callback := range pending {
			callback()
		}
	}
}
