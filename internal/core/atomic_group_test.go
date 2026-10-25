package core

import (
	"errors"
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
