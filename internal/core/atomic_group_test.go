package core

import (
	"errors"
	"maps"
	"reflect"
	"slices"
	"sync"
	"testing"
)

func TestAtomicGroupCommitsAnyNumberOfStores(t *testing.T) {
	first, second, third := New(1), New("a"), New(true)
	var transactions []uint64
	var mu sync.Mutex
	stops := []func(){
		first.SubscribeMutations(func(m Mutation[int]) { mu.Lock(); transactions = append(transactions, m.TransactionID); mu.Unlock() }),
		second.SubscribeMutations(func(m Mutation[string]) { mu.Lock(); transactions = append(transactions, m.TransactionID); mu.Unlock() }),
		third.SubscribeMutations(func(m Mutation[bool]) { mu.Lock(); transactions = append(transactions, m.TransactionID); mu.Unlock() }),
	}
	defer func() {
		for _, stop := range stops {
			stop()
		}
	}()
	if err := AtomicGroup(
		AtomicUpdate(first, func(value *int) error { *value = 2; return nil }),
		AtomicUpdate(second, func(value *string) error { *value = "b"; return nil }),
		AtomicUpdate(third, func(value *bool) error { *value = false; return nil }),
	); err != nil {
		t.Fatal(err)
	}
	if first.Get() != 2 || second.Get() != "b" || third.Get() {
		t.Fatalf("group values = %d, %q, %v", first.Get(), second.Get(), third.Get())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(transactions) != 3 || transactions[0] == 0 || transactions[0] != transactions[1] || transactions[1] != transactions[2] {
		t.Fatalf("transaction IDs = %v", transactions)
	}
}

func TestAtomicGroupRollsBackEveryCandidate(t *testing.T) {
	first, second, third := New(1), New(2), New(3)
	want := errors.New("abort")
	if err := AtomicGroup(
		AtomicUpdate(first, func(value *int) error { *value = 10; return nil }),
		AtomicUpdate(second, func(value *int) error { *value = 20; return want }),
		AtomicUpdate(third, func(value *int) error { *value = 30; return nil }),
	); !errors.Is(err, want) {
		t.Fatalf("error = %v", err)
	}
	if first.Get() != 1 || second.Get() != 2 || third.Get() != 3 {
		t.Fatalf("rollback values = %d, %d, %d", first.Get(), second.Get(), third.Get())
	}
	if err := AtomicGroup(
		AtomicUpdate(first, func(value *int) error { *value = 10; return nil }),
		AtomicUpdate(second, func(value *int) error { panic("boom") }),
	); err == nil {
		t.Fatal("panic was not converted to an error")
	}
	if first.Get() != 1 || second.Get() != 2 {
		t.Fatalf("panic rollback values = %d, %d", first.Get(), second.Get())
	}
}

func TestAtomicGroupRejectsDuplicateStores(t *testing.T) {
	store := New(0)
	if err := AtomicGroup(AtomicUpdate(store, func(*int) error { return nil }), AtomicUpdate(store, func(*int) error { return nil })); err == nil {
		t.Fatal("duplicate store accepted")
	}
}

type retainedBox struct {
	items []int
	tags  map[string]int
}

func cloneRetainedBox(b retainedBox) retainedBox {
	return retainedBox{items: slices.Clone(b.items), tags: maps.Clone(b.tags)}
}

func TestTransactionCommitsDoNotShareStateWithARetainedCandidate(t *testing.T) {
	newStore := func() *Store[retainedBox] {
		return NewWithClone(retainedBox{items: []int{1}, tags: map[string]int{"a": 1}}, cloneRetainedBox)
	}
	edit := func(retained **retainedBox) func(*retainedBox) {
		return func(b *retainedBox) { *retained = b; b.items[0] = 2; b.tags["a"] = 2 }
	}
	paths := []struct {
		name   string
		commit func(*Store[retainedBox], func(*retainedBox)) error
	}{
		{"AtomicGroup", func(s *Store[retainedBox], fn func(*retainedBox)) error {
			return AtomicGroup(AtomicUpdate(s, func(b *retainedBox) error { fn(b); return nil }))
		}},
		{"Atomic2", func(s *Store[retainedBox], fn func(*retainedBox)) error {
			return Atomic2(s, New(0), func(b *retainedBox, other *int) error { fn(b); *other = 1; return nil })
		}},
		{"Update", func(s *Store[retainedBox], fn func(*retainedBox)) error { return s.Update(fn) }},
	}
	for _, path := range paths {
		t.Run(path.name, func(t *testing.T) {
			store := newStore()
			defer func() { _ = store.Close() }()
			var retained *retainedBox
			requireNoError(t, path.commit(store, edit(&retained)))
			version := store.Version()
			retained.items[0] = 99
			retained.tags["a"] = 99
			got := store.Get()
			if got.items[0] != 2 || got.tags["a"] != 2 || store.Version() != version {
				t.Fatalf("a retained candidate changed owned state: items %v tags %v version %d, want [2] map[a:2] version %d", got.items, got.tags, store.Version(), version)
			}
		})
	}
}

func TestAtomicGroupAndAtomic2RejectOneClosedParticipant(t *testing.T) {
	// AtomicGroup prepares its participants in store address order, whatever the argument order, so closedLow decides which one it prepares first: with closedLow the closed store is rejected before the open store's callback runs, without it the open store is prepared and then the closed one is rejected. Atomic2 checks both stores before its callback, in argument order.
	paths := []struct {
		name      string
		closedLow bool
		run       func(open, closed *Store[int]) error
	}{
		{"AtomicGroup open prepared first", false, func(open, closed *Store[int]) error {
			return AtomicGroup(AtomicUpdate(closed, func(v *int) error { *v = 20; return nil }), AtomicUpdate(open, func(v *int) error { *v = 2; return nil }))
		}},
		{"AtomicGroup closed prepared first", true, func(open, closed *Store[int]) error {
			return AtomicGroup(AtomicUpdate(open, func(v *int) error { *v = 2; return nil }), AtomicUpdate(closed, func(v *int) error { *v = 20; return nil }))
		}},
		{"Atomic2", false, func(open, closed *Store[int]) error {
			return Atomic2(open, closed, func(a, b *int) error { *a, *b = 2, 20; return nil })
		}},
		{"Atomic2 closed first", false, func(open, closed *Store[int]) error {
			return Atomic2(closed, open, func(a, b *int) error { *a, *b = 20, 2; return nil })
		}},
	}
	for _, path := range paths {
		t.Run(path.name, func(t *testing.T) {
			low, high := New(0), New(0)
			if reflect.ValueOf(high).Pointer() < reflect.ValueOf(low).Pointer() {
				low, high = high, low
			}
			open, closed := low, high
			if path.closedLow {
				open, closed = high, low
			}
			requireNoError(t, open.Set(1))
			requireNoError(t, closed.Set(10))
			defer func() { _ = open.Close() }()
			requireNoError(t, closed.Close())
			openVersion, closedVersion := open.Version(), closed.Version()
			if err := path.run(open, closed); !errors.Is(err, ErrClosed) {
				t.Fatalf("err = %v, want ErrClosed", err)
			}
			if open.Get() != 1 || closed.Get() != 10 || open.Version() != openVersion || closed.Version() != closedVersion {
				t.Fatalf("state changed: open %d v%d (want 1 v%d), closed %d v%d (want 10 v%d)", open.Get(), open.Version(), openVersion, closed.Get(), closed.Version(), closedVersion)
			}
		})
	}
}
