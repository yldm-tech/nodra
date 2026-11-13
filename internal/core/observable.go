package core

import (
	"reflect"
	"sync"
)

// Observable is a versioned value source shared by stores, read-only views and derived values. Callbacks may overlap and must treat reference values as read-only. Implementations must use monotonically increasing versions and subscribe without replay; constructors catch up after subscribing. Transaction settlement orders derived values by height, which only Nodra's own derived values carry: an application implementation counts as a source of height zero even when it forwards a derived value, so a Combine over such a wrapper and the derived value's other inputs may see a half-applied Atomic2 or AtomicGroup. An application implementation also cannot tell Combine whether it has caught up with the Stores behind it, so Combine checks a pair that includes one only against its other input.
type Observable[T any] interface {
	Snapshot() Snapshot[T]
	SubscribeSnapshot(func(Snapshot[T])) func()
}

func validObservable[T any](source Observable[T]) bool {
	if isNilInterface(source) {
		return false
	}
	if reader, ok := source.(Reader[T]); ok {
		return sourceStore(reader) != nil
	}
	return true
}

func isNilInterface(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return v.IsNil()
	default:
		return false
	}
}

func (s *Store[T]) SubscribeSnapshot(fn func(Snapshot[T])) func()   { return s.Subscribe(fn) }
func (r ReadOnly[T]) SubscribeSnapshot(fn func(Snapshot[T])) func() { return r.Subscribe(fn) }

// Snapshot returns the memoized value, its accepted source version and the cross-store transaction that produced it. Derived reference values are read-only; equality-suppressed changes still advance the version without notifying.
func (d *Derived[T, V]) Snapshot() Snapshot[V] {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return Snapshot[V]{Value: d.value, Version: d.version, TransactionID: d.transactionID}
}

func (d *Derived[T, V]) SubscribeSnapshot(fn func(Snapshot[V])) func() {
	if d == nil || fn == nil {
		return func() {}
	}
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return func() {}
	}
	if d.snapshotListeners == nil {
		d.snapshotListeners = make(map[uint64]func(Snapshot[V]))
	}
	d.nextID++
	id := d.nextID
	d.snapshotListeners[id] = fn
	d.mu.Unlock()
	var once sync.Once
	return func() { once.Do(func() { d.mu.Lock(); delete(d.snapshotListeners, id); d.mu.Unlock() }) }
}

// Pair contains the latest observed values of two independent sources.
type Pair[A, B any] struct {
	First  A
	Second B
}

// Combine derives a value from two stores or derived values. Atomic2 and AtomicGroup transaction notifications are settled through the derived graph before downstream callbacks run, and Combine projects and publishes only input pairs that some committed state contained, also while concurrent writers update the same stores and while Combine is being constructed: when one input has not yet caught up with a write the other already reflects, Combine reads both inputs again, recomputing derived inputs that are behind, instead of using the mixed pair. Independent writes are eventually consistent, and concurrent writers may see fewer intermediate values. This covers Stores, ReadOnly views and Nodra derived values; an application Observable counts as a source of height zero, so one that forwards a derived value is not settled after it, and Combine checks a pair that includes one only against its other input, so a write that reaches the application Observable between Combine's reads of its two inputs can still give a mixed pair. Old source versions and obsolete computations cannot replace newer ones. Close the returned derived value to detach both sources. Projectors/comparators must be pure, concurrency-safe and keep source references immutable; another Combine that reads this one before it has caught up with its inputs may also call the projector, without changing the value.
func Combine[A, B, V any](first Observable[A], second Observable[B], project func(A, B) V, equals func(V, V) bool) *Derived[Pair[A, B], V] {
	if !validObservable(first) || !validObservable(second) || project == nil {
		return closedDerived[Pair[A, B], V]()
	}
	height := max(observableHeight(first), observableHeight(second)) + 1
	firstLinks, secondLinks := linksOf(first), linksOf(second)
	// read returns a pair both inputs held at one moment: each input was still current after both were read, so it held its value from before the other was read until then. An application Observable cannot tell and counts as current, so a pair that includes one is checked only against the other input, and read never waits for its version to hold still. own reports which values are the inputs' own rather than recomputed, and current whether each value is still current.
	read := func() (firstValue Snapshot[A], secondValue Snapshot[B], own [2]bool, current [2]func() bool) {
		for {
			firstValue, own[0], current[0] = firstLinks.read()
			secondValue, own[1], current[1] = secondLinks.read()
			if current[0]() && current[1]() {
				return firstValue, secondValue, own, current
			}
		}
	}
	a, b, own, initial := read()
	// An input is provisional while it holds a value that read recomputed because the derived input had not caught up yet; initial then tells whether the Stores that value was computed from are unchanged. The input's own value replaces it at the next notification or catch-up.
	provisional := [2]bool{!own[0], !own[1]}
	// consistent reports whether both inputs still hold the values they had at the given versions, so the pair of them existed together.
	consistent := func(firstVersion, secondVersion uint64, provisional [2]bool) bool {
		firstCurrent := initial[0]
		if !provisional[0] {
			firstCurrent = func() bool { return freshAt(firstLinks.fresh, firstVersion) }
		}
		secondCurrent := initial[1]
		if !provisional[1] {
			secondCurrent = func() bool { return freshAt(secondLinks.fresh, secondVersion) }
		}
		return firstCurrent() && secondCurrent()
	}
	// mu guards the state below. publishMu orders publishes and is taken before mu, never while holding it.
	var mu, publishMu sync.Mutex
	transactionID := uint64(0)
	if a.TransactionID != 0 && a.TransactionID == b.TransactionID {
		transactionID = a.TransactionID
	}
	// pair holds the newest accepted inputs and pairVersion counts its changes. published is the newest pair handed downstream, stamped with the pairVersion it was published at, and publishedCurrent tells whether it is still current.
	pair := Pair[A, B]{a.Value, b.Value}
	pairVersion := uint64(0)
	published := Snapshot[Pair[A, B]]{Value: pair, TransactionID: transactionID}
	publishedCurrent := func() bool { return initial[0]() && initial[1]() }
	// Without freshness for both inputs, as with an application Observable, a current published pair does not show that nothing new arrived.
	verifiable := firstLinks.fresh != nil && secondLinks.fresh != nil
	pendingTransactions := make(map[uint64]bool)
	snapshot := func() Snapshot[Pair[A, B]] {
		mu.Lock()
		defer mu.Unlock()
		return published
	}
	links := sourceLinks[Pair[A, B]]{
		fresh: func(version uint64) bool {
			mu.Lock()
			matches, current := published.Version == version, publishedCurrent
			mu.Unlock()
			return matches && current()
		},
		read: func() (Snapshot[Pair[A, B]], bool, func() bool) {
			firstValue, secondValue, _, current := read()
			return Snapshot[Pair[A, B]]{Value: Pair[A, B]{firstValue.Value, secondValue.Value}}, false, func() bool { return current[0]() && current[1]() }
		},
	}
	subscribe := func(fn func(Snapshot[Pair[A, B]])) (stop func()) {
		var stops []func()
		var once sync.Once
		stop = func() {
			once.Do(func() {
				for _, cancel := range stops {
					cancel()
				}
			})
		}
		ready := false
		defer func() {
			if !ready {
				stop()
			}
		}()
		// publish hands the newest pair downstream when both inputs still hold it. Otherwise one input has not caught up with a write the other may already reflect, so publish reads a pair both inputs hold now instead; the input still catching up later notifies a value that is already published, or none when equals filters it. publishMu serializes the check or read with the update of published, so a higher version always carries a pair checked or read later, and a publisher that finds the published pair still current compares it with the newest one: two publishers cannot finish in the opposite order and leave an older pair at the higher version. It is released before fn; the derived value drops a delivery that arrives after a newer one.
		publish := func(transactionID uint64) {
			publishMu.Lock()
			mu.Lock()
			stillPublished := publishedCurrent
			if pairVersion == published.Version {
				mu.Unlock()
				publishMu.Unlock()
				return
			}
			candidate := Snapshot[Pair[A, B]]{Value: pair, Version: pairVersion, TransactionID: transactionID}
			firstVersion, secondVersion, inputsProvisional := a.Version, b.Version, provisional
			mu.Unlock()
			if verifiable && stillPublished() {
				// The published pair is still what both inputs hold, so the notification only caught up with it.
				publishMu.Unlock()
				return
			}
			check := func() bool { return consistent(firstVersion, secondVersion, inputsProvisional) }
			if !check() {
				firstValue, secondValue, _, current := read()
				candidate.Value, check = Pair[A, B]{firstValue.Value, secondValue.Value}, func() bool { return current[0]() && current[1]() }
			}
			// Publishes are serialized and pairVersion only grows, so candidate.Version is above published.Version.
			mu.Lock()
			published, publishedCurrent = candidate, check
			mu.Unlock()
			publishMu.Unlock()
			fn(candidate)
		}
		// schedule publishes at once outside a transaction and after the transaction has settled every lower height inside one.
		schedule := func(transactionID uint64) {
			if transactionID == 0 {
				publish(0)
				return
			}
			mu.Lock()
			queued := pendingTransactions[transactionID]
			pendingTransactions[transactionID] = true
			mu.Unlock()
			if queued {
				return
			}
			deferTransaction(transactionID, height, func() {
				mu.Lock()
				delete(pendingTransactions, transactionID)
				mu.Unlock()
				publish(transactionID)
			})
		}
		acceptA := func(next Snapshot[A]) {
			mu.Lock()
			// The input's own value replaces a provisional one even at the same version.
			if next.Version < a.Version || next.Version == a.Version && !provisional[0] {
				mu.Unlock()
				return
			}
			a, provisional[0] = next, false
			pair.First, pairVersion = next.Value, pairVersion+1
			mu.Unlock()
			schedule(next.TransactionID)
		}
		acceptB := func(next Snapshot[B]) {
			mu.Lock()
			if next.Version < b.Version || next.Version == b.Version && !provisional[1] {
				mu.Unlock()
				return
			}
			b, provisional[1] = next, false
			pair.Second, pairVersion = next.Value, pairVersion+1
			mu.Unlock()
			schedule(next.TransactionID)
		}
		stops = append(stops, first.SubscribeSnapshot(acceptA))
		stops = append(stops, second.SubscribeSnapshot(acceptB))
		acceptA(first.Snapshot())
		acceptB(second.Snapshot())
		ready = true
		return stop
	}
	return derive(nil, snapshot, subscribe, func(pair Pair[A, B]) V { return project(pair.First, pair.Second) }, equals, height, links)
}

// sourceLinks is what a derived value knows about its source beyond snapshots and subscriptions; Combine uses it to tell whether its two inputs come from one committed state.
type sourceLinks[T any] struct {
	// fresh reports whether the source is still at version and every Store below it is still at the version its value was computed from. A nil fresh means the source cannot tell, as for an application Observable, and counts as current.
	fresh func(version uint64) bool
	// read returns the source's value computed from the current state of the Stores below it, recomputing derived values that have not caught up yet; whether that is the source's own value at the snapshot's version rather than a recomputed one; and a check that the Stores it was computed from are unchanged.
	read func() (Snapshot[T], bool, func() bool)
}

// freshNode is implemented by Stores, ReadOnly views and derived values.
type freshNode interface {
	fresh(version uint64) bool
}

// readThroughNode is implemented by derived values.
type readThroughNode[T any] interface {
	readThrough() (Snapshot[T], bool, func() bool)
}

func linksOf[T any](source Observable[T]) sourceLinks[T] {
	var links sourceLinks[T]
	if node, ok := source.(freshNode); ok {
		links.fresh = node.fresh
	}
	if node, ok := source.(readThroughNode[T]); ok {
		links.read = node.readThrough
	} else {
		fresh := links.fresh
		links.read = func() (Snapshot[T], bool, func() bool) {
			snapshot := source.Snapshot()
			return snapshot, true, func() bool { return freshAt(fresh, snapshot.Version) }
		}
	}
	return links
}

func freshAt(fresh func(uint64) bool, version uint64) bool {
	return fresh == nil || fresh(version)
}

// fresh reports whether the store is still at version. A closed store publishes nothing more, so it counts as current.
func (s *Store[T]) fresh(version uint64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed || s.version == version
}

func (r ReadOnly[T]) fresh(version uint64) bool {
	if r.store == nil {
		return true
	}
	return r.store.fresh(version)
}

// fresh reports whether the derived value is still at version and its source is still at the version that value was computed from. A version it accepted but filtered as equal counts as a change: its source may have left the value and come back meanwhile, so only an unchanged version shows that the value held throughout. A closed derived value publishes nothing more, so it counts as current.
func (d *Derived[T, V]) fresh(version uint64) bool {
	d.mu.RLock()
	closed, current, sourceFresh := d.closed, d.version, d.sourceFresh
	d.mu.RUnlock()
	if closed {
		return true
	}
	return version == current && freshAt(sourceFresh, current)
}

// readThrough returns the derived value when it is current and otherwise recomputes it from its source's current value, without changing the derived value.
func (d *Derived[T, V]) readThrough() (Snapshot[V], bool, func() bool) {
	snapshot := d.Snapshot()
	current := func() bool { return d.fresh(snapshot.Version) }
	if d.readSource == nil || current() {
		return snapshot, true, current
	}
	source, _, sourceCurrent := d.readSource()
	return Snapshot[V]{Value: d.selector(source.Value), Version: snapshot.Version, TransactionID: snapshot.TransactionID}, false, sourceCurrent
}
