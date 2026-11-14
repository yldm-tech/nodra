package core

import (
	"errors"
	"reflect"
	"sync"
	"time"
)

// Create is the Pinia/Zustand-style constructor name for a typed store.
func Create[T any](initial T) *Store[T] { return New(initial) }

// SelectorOptions configures a selector subscription. With no Equals function, comparable values use == (including pointer identity); other values use reflect.DeepEqual. A custom comparator can use slices.Equal or maps.Equal.
type SelectorOptions[V any] struct {
	Immediate bool
	Equals    func(previous, next V) bool
}

// SubscribeSelectorWith calls fn with current and previous selected values. It catches up after registration and ignores obsolete computations. Callbacks run on writer goroutines and may overlap, as with Store.Subscribe.
func SubscribeSelectorWith[T any, V any](source Observable[T], selector func(T) V, fn func(current, previous V), options SelectorOptions[V]) func() {
	if !validObservable(source) || selector == nil || fn == nil {
		return func() {}
	}
	return subscribeSelection(source.Snapshot, source.SubscribeSnapshot, selector, fn, options)
}

func subscribeSelection[T, V any](snapshot func() Snapshot[T], subscribe func(func(Snapshot[T])) func(), selector func(T) V, fn func(current, previous V), options SelectorOptions[V]) func() {
	compare := options.Equals
	if compare == nil {
		compare = selectorEqualFor[V]()
	}
	var mu sync.Mutex
	initial := snapshot()
	previous := selector(initial.Value)
	version := initial.Version
	if options.Immediate {
		var zero V
		fn(previous, zero)
	}
	changed := func(snapshot Snapshot[T]) {
		mu.Lock()
		stale := snapshot.Version <= version
		mu.Unlock()
		if stale {
			return
		}
		current := selector(snapshot.Value)
		for {
			mu.Lock()
			if snapshot.Version <= version {
				mu.Unlock()
				return
			}
			old, comparedVersion := previous, version
			mu.Unlock()
			// Comparators may panic or read the source without holding our lock.
			equal := compare(old, current)
			mu.Lock()
			if snapshot.Version <= version {
				mu.Unlock()
				return
			}
			if version != comparedVersion {
				mu.Unlock()
				continue // Compare again against the newer baseline.
			}
			previous, version = current, snapshot.Version
			mu.Unlock()
			if !equal {
				fn(current, old)
			}
			return
		}
	}
	stop := subscribe(changed)
	initialized := false
	defer func() {
		if !initialized {
			stop()
		}
	}()
	// Catch changes between reading the initial value and registering. The version check also avoids recomputing when a callback already caught up.
	changed(snapshot())
	initialized = true
	return stop
}

// selectorEqualFor chooses the default selector equality once per subscription. A comparable type that cannot hold an interface value can never hold a non-comparable dynamic value, so selectorEqual would always take its == branch for it; such types compare with == directly and skip the per-call reflection. Every other type keeps selectorEqual.
func selectorEqualFor[V any]() func(previous, next V) bool {
	if t := reflect.TypeFor[V](); t.Comparable() && !holdsInterface(t) {
		return func(previous, next V) bool { return any(previous) == any(next) }
	}
	return selectorEqual[V]
}

// holdsInterface reports whether a value of t can contain an interface value inline, through struct fields or array elements. Pointers, channels and other reference kinds compare by identity, so their targets do not matter.
func holdsInterface(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Interface:
		return true
	case reflect.Array:
		return holdsInterface(t.Elem())
	case reflect.Struct:
		for i := range t.NumField() {
			if holdsInterface(t.Field(i).Type) {
				return true
			}
		}
	}
	return false
}

// selectorEqual preserves == semantics when dynamic values are comparable too. Interfaces and structs containing interfaces can hold slices/maps at runtime.
func selectorEqual[V any](previous, next V) bool {
	a, b := reflect.ValueOf(previous), reflect.ValueOf(next)
	if (!a.IsValid() || a.Comparable()) && (!b.IsValid() || b.Comparable()) {
		return any(previous) == any(next)
	}
	return reflect.DeepEqual(previous, next)
}

// Derived is a memoized getter derived from an observable value. It recomputes when the source publishes a newer version; a Combine that reads it before it has caught up with its source may also call its selector, without changing the derived value. With an equals function it notifies listeners only when the selected value changes; with a nil equals every accepted source version replaces the value and notifies, so pass an equality function when the derived value filters changes, for example for Invalidate or the bridge. Within an Atomic2 or AtomicGroup, a derived value settles after every source that is a Store, a ReadOnly view or another derived value; an application Observable counts as a source of height zero, even when it forwards a derived value. Done is closed by Close; closing a source does not close the derived value, which keeps its last value. Reference values and listener arguments remain read-only.
type Derived[T any, V any] struct {
	source            *Store[T]
	selector          func(T) V
	equals            func(V, V) bool
	mu                sync.RWMutex
	value             V
	version           uint64
	transactionID     uint64
	listeners         map[uint64]func(V)
	snapshotListeners map[uint64]func(Snapshot[V])
	// filteredListeners hear of writes below the derived value that reached it without a notification, which listeners and snapshotListeners do not: source versions its equals filtered, and versions a source below it filtered.
	filteredListeners map[uint64]func(transactionID uint64)
	nextID            uint64
	stop              func()
	closed            bool
	done              chan struct{}
	height            int

	// sourceFresh and readSource come from the source's sourceLinks.
	sourceFresh func(uint64) bool
	readSource  func() (Snapshot[T], bool, func() bool)
}

// closedDerived is the result of an invalid construction: closed from the start, with Done already closed.
func closedDerived[T, V any]() *Derived[T, V] {
	done := make(chan struct{})
	close(done)
	return &Derived[T, V]{closed: true, done: done}
}

func (d *Derived[T, V]) transactionHeight() int { return d.height }

func Derive[T any, V any](source Observable[T], selector func(T) V, equals func(V, V) bool) *Derived[T, V] {
	if !validObservable(source) {
		return closedDerived[T, V]()
	}
	sourceFiltered := func(fn func(uint64)) func() { return subscribeFiltered(source, fn) }
	return derive(nil, source.Snapshot, source.SubscribeSnapshot, sourceFiltered, selector, equals, observableHeight(source)+1, linksOf(source))
}

// DeriveProjection memoizes an owned projection without whole-source clones. It follows SubscribeProjection's ownership contract; returned reference values and listener arguments are read-only. A Combine that reads the derived value before it has caught up with source may also call project, without changing the derived value.
func DeriveProjection[T any, V any](source Reader[T], project func(T) V, equals func(V, V) bool) *Derived[T, V] {
	store := sourceStore(source)
	if store == nil {
		return closedDerived[T, V]()
	}
	links := sourceLinks[T]{fresh: store.fresh, read: func() (Snapshot[T], bool, func() bool) {
		snapshot := store.borrowedSnapshot()
		return snapshot, true, func() bool { return store.fresh(snapshot.Version) }
	}}
	return derive(store, store.borrowedSnapshot, store.subscribeBorrowed, nil, project, equals, 1, links)
}

// derive builds a derived value at height, which is one more than the height of its highest source. subscribeFiltered, when not nil, registers for the writes below the source that reached it without a notification; the derived value passes them on to its own filtered listeners, so a Combine above hears of them at any depth.
func derive[T any, V any](store *Store[T], snapshot func() Snapshot[T], subscribe func(func(Snapshot[T])) func(), subscribeFiltered func(func(uint64)) func(), selector func(T) V, equals func(V, V) bool, height int, links sourceLinks[T]) *Derived[T, V] {
	d := &Derived[T, V]{source: store, selector: selector, equals: equals, listeners: make(map[uint64]func(V)), done: make(chan struct{}), height: height}
	if selector == nil {
		d.closed = true
		close(d.done)
		return d
	}
	initial := snapshot()
	d.value = selector(initial.Value)
	d.version = initial.Version
	d.transactionID = initial.TransactionID
	d.sourceFresh, d.readSource = links.fresh, links.read
	var pendingMu sync.Mutex
	// pendingTransactions holds the newest snapshot of each transaction whose settlement is queued.
	pendingTransactions := make(map[uint64]Snapshot[T])
	process := func(input Snapshot[T]) {
		d.mu.RLock()
		stale := d.closed || input.Version <= d.version
		d.mu.RUnlock()
		if stale {
			return
		}
		next := selector(input.Value)
		for {
			d.mu.RLock()
			if d.closed || input.Version <= d.version {
				d.mu.RUnlock()
				return
			}
			old, comparedVersion := d.value, d.version
			d.mu.RUnlock()
			equal := d.equals != nil && d.equals(old, next)
			d.mu.Lock()
			if d.closed || input.Version <= d.version {
				d.mu.Unlock()
				return
			}
			if d.version != comparedVersion {
				d.mu.Unlock()
				continue
			}
			// Advance even when equal, but only after comparison succeeds.
			d.version = input.Version
			d.transactionID = input.TransactionID
			if equal {
				filtered := d.filteredLocked()
				d.mu.Unlock()
				for _, fn := range filtered {
					fn(input.TransactionID)
				}
				return
			}
			d.value = next
			listeners := make([]func(V), 0, len(d.listeners))
			for _, fn := range d.listeners {
				listeners = append(listeners, fn)
			}
			snapshotListeners := make([]func(Snapshot[V]), 0, len(d.snapshotListeners))
			for _, fn := range d.snapshotListeners {
				snapshotListeners = append(snapshotListeners, fn)
			}
			d.mu.Unlock()
			for _, fn := range listeners {
				fn(next)
			}
			for _, fn := range snapshotListeners {
				fn(Snapshot[V]{Value: next, Version: input.Version, TransactionID: input.TransactionID})
			}
			return
		}
	}
	changed := func(input Snapshot[T]) {
		if input.TransactionID == 0 {
			process(input)
			return
		}
		transactionID := input.TransactionID
		pendingMu.Lock()
		if pending, queued := pendingTransactions[transactionID]; queued {
			if input.Version > pending.Version {
				pendingTransactions[transactionID] = input
			}
			pendingMu.Unlock()
			return
		}
		pendingTransactions[transactionID] = input
		pendingMu.Unlock()
		deferTransaction(transactionID, d.height, func() {
			pendingMu.Lock()
			latest := pendingTransactions[transactionID]
			delete(pendingTransactions, transactionID)
			pendingMu.Unlock()
			process(latest)
		})
	}
	d.stop = subscribe(changed)
	if subscribeFiltered != nil {
		stopChanged, stopFiltered := d.stop, subscribeFiltered(d.passFiltered)
		d.stop = func() { stopChanged(); stopFiltered() }
	}
	initialized := false
	defer func() {
		if !initialized {
			d.stop()
		}
	}()
	changed(snapshot())
	initialized = true
	return d
}

// filteredLocked returns the filtered listeners; the caller holds d.mu.
func (d *Derived[T, V]) filteredLocked() []func(uint64) {
	filtered := make([]func(uint64), 0, len(d.filteredListeners))
	for _, fn := range d.filteredListeners {
		filtered = append(filtered, fn)
	}
	return filtered
}

// passFiltered tells the filtered listeners of a write below the source that reached it without a notification. The source's version moved, so this value is no longer fresh, and a Combine that could not read a pair because of the write must hear of it to publish again.
func (d *Derived[T, V]) passFiltered(transactionID uint64) {
	d.mu.Lock()
	if d.closed || len(d.filteredListeners) == 0 {
		d.mu.Unlock()
		return
	}
	filtered := d.filteredLocked()
	d.mu.Unlock()
	for _, fn := range filtered {
		fn(transactionID)
	}
}

// Done is closed when Close detaches the derived value, and is already closed for one built from an invalid source or a nil selector. It is nil for a nil Derived.
func (d *Derived[T, V]) Done() <-chan struct{} {
	if d == nil {
		return nil
	}
	return d.done
}
func (d *Derived[T, V]) Get() V { d.mu.RLock(); defer d.mu.RUnlock(); return d.value }
func (d *Derived[T, V]) Subscribe(fn func(V)) func() {
	if d == nil || fn == nil {
		return func() {}
	}
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return func() {}
	}
	d.nextID++
	id := d.nextID
	d.listeners[id] = fn
	d.mu.Unlock()
	var once sync.Once
	return func() { once.Do(func() { d.mu.Lock(); delete(d.listeners, id); d.mu.Unlock() }) }
}
func (d *Derived[T, V]) Close() error {
	if d == nil {
		return ErrClosed
	}
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return ErrClosed
	}
	d.closed = true
	stop := d.stop
	d.listeners = nil
	d.snapshotListeners = nil
	d.filteredListeners = nil
	if d.done != nil {
		close(d.done)
	}
	d.mu.Unlock()
	// Unsubscribe without the derived lock: a Store's unsubscribe takes the store lock, which an update holds while its callback may read this value. A notification that arrives meanwhile sees closed and is dropped.
	if stop != nil {
		stop()
	}
	return nil
}

// MutationType describes why a store published a snapshot.
type MutationType string

const (
	MutationDirect      MutationType = "direct"
	MutationPatch       MutationType = "patch"
	MutationAction      MutationType = "action"
	MutationReset       MutationType = "reset"
	MutationTransaction MutationType = "transaction"
)

// Mutation contains Pinia-like subscription metadata.
type Mutation[T any] struct {
	ActionID      uint64
	TransactionID uint64
	Type          MutationType
	Name          string
	Previous      T
	Value         T
	Version       uint64
	At            time.Time
}

// ActionEvent describes one completed action.
type ActionEvent struct {
	Metrics   TaskMetrics
	ID        uint64
	RunID     uint64
	Key       string
	Version   uint64
	Committed bool
	StoreID   string
	Name      string
	Started   time.Time
	Finished  time.Time
	Duration  time.Duration
	Err       error
}

// ActionHooks observes action lifecycle events. Before runs before the mutation; After runs only on success; Error runs only on failure.
type ActionHooks struct {
	Before func(ActionEvent)
	After  func(ActionEvent)
	Error  func(ActionEvent)
}

// SubscribeActionHooks registers lifecycle hooks and returns an unsubscribe function.
func (s *Store[T]) SubscribeActionHooks(hooks ActionHooks) func() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return func() {}
	}
	s.nextID++
	id := s.nextID
	s.actionHooks[id] = hooks
	s.mu.Unlock()
	var once sync.Once
	return func() { once.Do(func() { s.mu.Lock(); delete(s.actionHooks, id); s.mu.Unlock() }) }
}

func (s *Store[T]) SubscribeActions(fn func(ActionEvent)) func() {
	if fn == nil {
		return func() {}
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return func() {}
	}
	s.nextID++
	id := s.nextID
	s.actionListeners[id] = fn
	s.mu.Unlock()
	var once sync.Once
	return func() { once.Do(func() { s.mu.Lock(); delete(s.actionListeners, id); s.mu.Unlock() }) }
}

// Do runs a named mutation and emits an action event after it completes.
func (s *Store[T]) Do(name string, fn func(*T) error) error {
	if fn == nil {
		return errors.New("nodra: nil action")
	}
	event, hooks := s.newAction(name)
	for _, hook := range hooks {
		if hook.Before != nil {
			hook.Before(event)
		}
	}
	err := s.updateCommitted(name, MutationAction, fn, func(version uint64) { event.Version = version; event.Committed = true }, event.ID)
	s.finishAction(event, hooks, err)
	return err
}

func (s *Store[T]) newAction(name string) (ActionEvent, []ActionHooks) {
	started := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextAction++
	event := ActionEvent{ID: s.nextAction, StoreID: s.id, Name: name, Started: started, Version: s.version}
	hooks := make([]ActionHooks, 0, len(s.actionHooks))
	for _, hook := range s.actionHooks {
		hooks = append(hooks, hook)
	}
	return event, hooks
}

func (s *Store[T]) finishAction(event ActionEvent, hooks []ActionHooks, err error) {
	finished := time.Now()
	event.Finished = finished
	event.Duration = finished.Sub(event.Started)
	event.Err = err
	s.mu.RLock()
	listeners := make([]func(ActionEvent), 0, len(s.actionListeners))
	for _, listener := range s.actionListeners {
		listeners = append(listeners, listener)
	}
	s.mu.RUnlock()
	for _, listener := range listeners {
		listener(event)
	}
	for _, hook := range hooks {
		if err == nil {
			if hook.After != nil {
				hook.After(event)
			}
		} else if hook.Error != nil {
			hook.Error(event)
		}
	}
}

// Action returns a reusable named action function, matching Pinia action definitions while keeping Go's explicit function types.
func (s *Store[T]) Action(name string, fn func(*T) error) func() error {
	return func() error { return s.Do(name, fn) }
}
