package nodra

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestAtomic2CommitsBothStoresBeforeNotifications(t *testing.T) {
	left, right := New(1), New("a")
	defer func() { _ = left.Close() }()
	defer func() { _ = right.Close() }()
	var mu sync.Mutex
	var leftMutations, rightMutations []Mutation[uint64]
	left.SubscribeMutations(func(event Mutation[int]) {
		if left.Get() != 2 || right.Get() != "b" {
			t.Error("left notification observed a partial transaction")
		}
		if event.Type != MutationTransaction || event.TransactionID == 0 {
			t.Errorf("left metadata = %+v", event)
		}
	})
	right.SubscribeMutations(func(event Mutation[string]) {
		if left.Get() != 2 || right.Get() != "b" {
			t.Error("right notification observed a partial transaction")
		}
		if event.Type != MutationTransaction || event.TransactionID == 0 {
			t.Errorf("right metadata = %+v", event)
		}
	})
	_ = left.Subscribe(func(Snapshot[int]) {
		mu.Lock()
		leftMutations = append(leftMutations, Mutation[uint64]{Version: left.Version()})
		mu.Unlock()
	})
	_ = right.Subscribe(func(Snapshot[string]) {
		mu.Lock()
		rightMutations = append(rightMutations, Mutation[uint64]{Version: right.Version()})
		mu.Unlock()
	})
	if err := Atomic2(left, right, func(a *int, b *string) error { *a++; *b = "b"; return nil }); err != nil {
		t.Fatal(err)
	}
	if left.Get() != 2 || right.Get() != "b" || left.Version() != 1 || right.Version() != 1 {
		t.Fatalf("committed values: %d/%s versions %d/%d", left.Get(), right.Get(), left.Version(), right.Version())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(leftMutations) != 1 || len(rightMutations) != 1 {
		t.Fatalf("notifications: %d/%d", len(leftMutations), len(rightMutations))
	}
}

func TestAtomic2RollsBackOnErrorAndPanic(t *testing.T) {
	left, right := New(1), New(2)
	defer func() { _ = left.Close() }()
	defer func() { _ = right.Close() }()
	for _, test := range []struct {
		name string
		fn   func(*int, *int) error
		want error
	}{
		{"error", func(a, b *int) error { *a, *b = 3, 4; return errors.New("reject") }, errors.New("reject")},
		{"panic", func(a, b *int) error { *a, *b = 5, 6; panic("boom") }, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := Atomic2(left, right, test.fn)
			if test.want != nil && err == nil {
				t.Fatalf("error = %v", err)
			}
			if test.want == nil {
				var panicErr *PanicError
				if !errors.As(err, &panicErr) {
					t.Fatalf("panic error = %v", err)
				}
			}
			if left.Get() != 1 || right.Get() != 2 || left.Version() != 0 || right.Version() != 0 {
				t.Fatalf("rollback: %d/%d v%d/%d", left.Get(), right.Get(), left.Version(), right.Version())
			}
		})
	}
}

func TestAtomic2OrdersLocksAndBlocksReaders(t *testing.T) {
	left, right := New(0), New(0)
	defer func() { _ = left.Close() }()
	defer func() { _ = right.Close() }()
	started, release := make(chan struct{}), make(chan struct{})
	go func() {
		if err := Atomic2(left, right, func(a, b *int) error { *a, *b = 1, 1; close(started); <-release; return nil }); err != nil {
			t.Error(err)
		}
	}()
	<-started
	readDone := make(chan struct{})
	go func() { _ = left.Get(); _ = right.Get(); close(readDone) }()
	select {
	case <-readDone:
		t.Fatal("reader crossed an in-flight transaction")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	select {
	case <-readDone:
	case <-time.After(time.Second):
		t.Fatal("reader remained blocked after transaction")
	}
	if left.Get() != 1 || right.Get() != 1 {
		t.Fatal("transaction did not publish both stores")
	}
	// Reverse order in another goroutine must not deadlock.
	done := make(chan error, 1)
	go func() { done <- Atomic2(right, left, func(a, b *int) error { *a = 2; *b = 2; return nil }) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("reverse transaction deadlocked")
	}
	_ = context.Background()
}
