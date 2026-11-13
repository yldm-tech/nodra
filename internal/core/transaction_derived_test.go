package core

import (
	"fmt"
	"slices"
	"sync"
	"testing"
)

func TestCombinePublishesOnceForAtomic2(t *testing.T) {
	first, second := New(0), New(0)
	combined := Combine(first, second, func(a, b int) int { return a + b }, func(a, b int) bool { return a == b })
	defer func() { _ = combined.Close() }()
	var values []int
	stop := combined.Subscribe(func(value int) { values = append(values, value) })
	defer stop()
	if err := Atomic2(first, second, func(a, b *int) error { *a, *b = 1, 2; return nil }); err != nil {
		t.Fatal(err)
	}
	if got := combined.Get(); got != 3 {
		t.Fatalf("combined = %d, want 3", got)
	}
	if len(values) != 1 || values[0] != 3 {
		t.Fatalf("notifications = %v, want [3]", values)
	}
}

func TestDiamondDerivedPublishesOnlyFinalAtomicValue(t *testing.T) {
	first, second := New(0), New(0)
	left := Combine(first, second, func(a, b int) int { return a + b }, func(a, b int) bool { return a == b })
	right := Combine(first, second, func(a, b int) int { return a * b }, func(a, b int) bool { return a == b })
	final := Combine(left, right, func(sum, product int) int { return sum*100 + product }, func(a, b int) bool { return a == b })
	defer func() { _ = final.Close(); _ = right.Close(); _ = left.Close() }()
	var values []int
	stop := final.Subscribe(func(value int) { values = append(values, value) })
	defer stop()
	if err := Atomic2(first, second, func(a, b *int) error { *a, *b = 1, 2; return nil }); err != nil {
		t.Fatal(err)
	}
	if got := final.Get(); got != 302 {
		t.Fatalf("final = %d, want 302", got)
	}
	if len(values) != 1 || values[0] != 302 {
		t.Fatalf("diamond notifications = %v, want [302]", values)
	}
}

// pushObservable lets a test publish arbitrary snapshots, including several with one transaction ID.
type pushObservable[T any] struct {
	mu        sync.Mutex
	current   Snapshot[T]
	listeners []func(Snapshot[T])
}

func (p *pushObservable[T]) Snapshot() Snapshot[T] {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.current
}

func (p *pushObservable[T]) SubscribeSnapshot(fn func(Snapshot[T])) func() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.listeners = append(p.listeners, fn)
	return func() {}
}

func (p *pushObservable[T]) publish(snapshot Snapshot[T]) {
	p.mu.Lock()
	p.current = snapshot
	listeners := slices.Clone(p.listeners)
	p.mu.Unlock()
	for _, fn := range listeners {
		fn(snapshot)
	}
}

func TestDeferredTransactionCallbackQueuedAfterTheFinalDrainStillRuns(t *testing.T) {
	id := nextTransactionID.Add(1)
	lookedUp, resume := make(chan struct{}), make(chan struct{})
	deferTransactionLookedUp = func(got uint64) {
		if got == id {
			close(lookedUp)
			<-resume
		}
	}
	defer func() { deferTransactionLookedUp = nil }()
	ran, deferred := make(chan struct{}), make(chan struct{})
	runTransactionDispatch(id, []func(){func() {
		// Another goroutine finds this dispatch while it is active and queues on it only after the dispatcher has returned.
		go func() {
			deferTransaction(id, 1, func() { close(ran) })
			close(deferred)
		}()
		<-lookedUp
	}})
	close(resume)
	<-deferred
	select {
	case <-ran:
	default:
		t.Fatal("a callback deferred onto a finished dispatch never ran")
	}
}

func TestDerivedSettlesTheLatestSnapshotOfATransaction(t *testing.T) {
	source := &pushObservable[int]{}
	derived := Derive[int, int](source, func(v int) int { return v }, nil)
	defer func() { _ = derived.Close() }()
	var values []int
	derived.Subscribe(func(v int) { values = append(values, v) })
	id := nextTransactionID.Add(1)
	runTransactionDispatch(id, []func(){
		func() { source.publish(Snapshot[int]{Value: 1, Version: 1, TransactionID: id}) },
		func() { source.publish(Snapshot[int]{Value: 2, Version: 2, TransactionID: id}) },
	})
	if got := derived.Snapshot(); got.Value != 2 || got.Version != 2 || got.TransactionID != id {
		t.Fatalf("derived = %+v, want the latest snapshot of transaction %d", got, id)
	}
	if len(values) != 1 || values[0] != 2 {
		t.Fatalf("notifications = %v, want [2]", values)
	}
}

func TestTransactionsSettleSourcesOfUnequalDepthFirst(t *testing.T) {
	identity := func(v int) int { return v }
	graphs := []struct {
		name  string
		build func(a, b *Store[int]) (*Derived[Pair[int, int], Pair[int, int]], func())
	}{
		{"store and derived", func(a, b *Store[int]) (*Derived[Pair[int, int], Pair[int, int]], func()) {
			right := Derive[int, int](b, identity, nil)
			combined := Combine[int, int, Pair[int, int]](a, right, func(x, y int) Pair[int, int] { return Pair[int, int]{x, y} }, nil)
			return combined, func() { _ = combined.Close(); _ = right.Close() }
		}},
		{"derived chain and store", func(a, b *Store[int]) (*Derived[Pair[int, int], Pair[int, int]], func()) {
			inner := Derive[int, int](a, identity, nil)
			left := Derive[int, int](inner, identity, nil)
			combined := Combine[int, int, Pair[int, int]](left, b, func(x, y int) Pair[int, int] { return Pair[int, int]{x, y} }, nil)
			return combined, func() { _ = combined.Close(); _ = left.Close(); _ = inner.Close() }
		}},
	}
	commits := []struct {
		name   string
		commit func(a, b *Store[int], value int) error
	}{
		{"Atomic2(a, b)", func(a, b *Store[int], value int) error {
			return Atomic2(a, b, func(x, y *int) error { *x, *y = value, value; return nil })
		}},
		{"Atomic2(b, a)", func(a, b *Store[int], value int) error {
			return Atomic2(b, a, func(y, x *int) error { *x, *y = value, value; return nil })
		}},
		{"AtomicGroup", func(a, b *Store[int], value int) error {
			set := func(v *int) error { *v = value; return nil }
			return AtomicGroup(AtomicUpdate(a, set), AtomicUpdate(b, set))
		}},
	}
	for _, graph := range graphs {
		for _, commit := range commits {
			t.Run(graph.name+"/"+commit.name, func(t *testing.T) {
				a, b := New(0), New(0)
				combined, closeGraph := graph.build(a, b)
				defer closeGraph()
				var pairs []Pair[int, int]
				combined.Subscribe(func(p Pair[int, int]) { pairs = append(pairs, p) })
				over := Derive[Pair[int, int], string](combined, func(p Pair[int, int]) string { return fmt.Sprintf("%d/%d", p.First, p.Second) }, nil)
				defer func() { _ = over.Close() }()
				var labels []string
				over.Subscribe(func(label string) { labels = append(labels, label) })
				for value := 1; value <= 2; value++ {
					if err := commit.commit(a, b, value); err != nil {
						t.Fatal(err)
					}
				}
				want := []Pair[int, int]{{1, 1}, {2, 2}}
				if fmt.Sprint(pairs) != fmt.Sprint(want) {
					t.Fatalf("combined notifications = %v, want %v", pairs, want)
				}
				if fmt.Sprint(labels) != "[1/1 2/2]" {
					t.Fatalf("derived notifications = %v, want [1/1 2/2]", labels)
				}
				if got := over.Get(); got != "2/2" {
					t.Fatalf("derived = %q, want 2/2", got)
				}
			})
		}
	}
}
