package core

import (
	"context"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"
)

type counterState struct {
	Count int
	Name  string
}

// requireNoError fails the test immediately when a store operation that is expected to succeed returns an error.
func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestStorePublishesSnapshotsAfterUpdates(t *testing.T) {
	store := New(counterState{Count: 1})
	var got []Snapshot[counterState]
	unsubscribe := store.Subscribe(func(snapshot Snapshot[counterState]) { got = append(got, snapshot) })

	requireNoError(t, store.Update(func(state *counterState) { state.Count++ }))
	requireNoError(t, store.Set(counterState{Count: 3, Name: "done"}))
	unsubscribe()
	requireNoError(t, store.Update(func(state *counterState) { state.Count++ }))

	if len(got) != 2 {
		t.Fatalf("notifications = %d, want 2", len(got))
	}
	if got[0].Version != 1 || got[1].Version != 2 {
		t.Fatalf("versions = %d, %d", got[0].Version, got[1].Version)
	}
	if got[1].Value.Count != 3 || got[1].Value.Name != "done" {
		t.Fatalf("snapshot = %#v", got[1].Value)
	}
	if store.Version() != 3 || store.Snapshot().Value.Count != 4 {
		t.Fatalf("store = version %d, value %#v", store.Version(), store.Snapshot().Value)
	}
}

func TestBatchCoalescesNestedUpdates(t *testing.T) {
	store := New(0)
	var got []Snapshot[int]
	store.Subscribe(func(snapshot Snapshot[int]) { got = append(got, snapshot) })

	requireNoError(t, store.Batch(func() {
		requireNoError(t, store.Set(1))
		requireNoError(t, store.Batch(func() {
			requireNoError(t, store.Set(2))
			requireNoError(t, store.Set(3))
		}))
		requireNoError(t, store.Set(4))
	}))

	if !reflect.DeepEqual(got, []Snapshot[int]{{Value: 4, Version: 1}}) {
		t.Fatalf("snapshots = %#v", got)
	}
}

func TestSelectorOnlyPublishesWhenSelectionChanges(t *testing.T) {
	store := New(counterState{Count: 1, Name: "a"})
	var got []int
	unsubscribe := SubscribeSelector(store, func(state counterState) int { return state.Count }, func(value int) { got = append(got, value) })
	defer unsubscribe()

	requireNoError(t, store.Update(func(state *counterState) { state.Name = "b" }))
	requireNoError(t, store.Update(func(state *counterState) { state.Count = 2 }))
	requireNoError(t, store.Update(func(state *counterState) { state.Count = 2 }))

	if !reflect.DeepEqual(got, []int{2}) {
		t.Fatalf("selected values = %#v", got)
	}
}

func TestWatchStopsWithContext(t *testing.T) {
	store := New(0)
	ctx, cancel := context.WithCancel(context.Background())
	updates := store.Watch(ctx, 1)
	requireNoError(t, store.Set(1))
	select {
	case snapshot := <-updates:
		if snapshot.Value != 1 {
			t.Fatalf("value = %d", snapshot.Value)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for update")
	}
	cancel()
	// The cancel takes effect asynchronously, so this update may still be delivered; the channel must close either way.
	requireNoError(t, store.Set(2))
	deadline := time.After(5 * time.Second)
	for {
		select {
		case _, ok := <-updates:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("watch channel not closed after cancel")
		}
	}
}

func TestStoreSupportsConcurrentUpdates(t *testing.T) {
	store := New(0)
	const workers = 8
	const updates = 100
	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			for range updates {
				if err := store.Update(func(value *int) { *value++ }); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}
	wg.Wait()
	if store.Snapshot().Value != workers*updates {
		t.Fatalf("value = %d", store.Snapshot().Value)
	}
}

func TestUpdateCallbackReadsAnotherStoreWhileATransactionWaits(t *testing.T) {
	a, b := New(0), New(41)
	c, d := New(0), New(0)
	entered, release := make(chan struct{}), make(chan struct{})
	updated := make(chan error, 1)
	go func() {
		updated <- a.Update(func(v *int) {
			close(entered)
			<-release
			*v = b.Get() + int(b.Version()) + 1
		})
	}()
	<-entered
	committed := make(chan error, 1)
	go func() { committed <- Atomic2(c, d, func(x, y *int) error { *x, *y = 1, 2; return nil }) }()
	// A writer queued on the transaction gate makes TryRLock fail while a.Update still holds its read lock.
	for waiting := false; !waiting; {
		select {
		case err := <-committed:
			t.Fatalf("Atomic2 finished while an update held the gate: %v", err)
		default:
		}
		if storeTransactionMu.TryRLock() {
			storeTransactionMu.RUnlock()
			runtime.Gosched()
			continue
		}
		waiting = true
	}
	close(release)
	select {
	case err := <-updated:
		requireNoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("an update reading another store deadlocked against a waiting Atomic2")
	}
	select {
	case err := <-committed:
		requireNoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Atomic2 did not finish after the update")
	}
	if got := a.Get(); got != 42 {
		t.Fatalf("a = %d, want 42", got)
	}
	if c.Get() != 1 || d.Get() != 2 {
		t.Fatalf("transaction values = %d, %d", c.Get(), d.Get())
	}
}

func TestCloseInsideABatchGivesTheFinalValueANewVersion(t *testing.T) {
	store := New(0)
	requireNoError(t, store.Set(1))
	published := store.Snapshot()
	if err := store.Batch(func() {
		requireNoError(t, store.Set(2))
		requireNoError(t, store.Close())
	}); err != nil {
		t.Fatal(err)
	}
	final := store.Snapshot()
	if final.Value != 2 {
		t.Fatalf("final value = %d, want the committed 2", final.Value)
	}
	if final.Version <= published.Version {
		t.Fatalf("final snapshot %+v reuses version %d, which was published with value %d", final, published.Version, published.Value)
	}
	if err := store.Close(); err != ErrClosed {
		t.Fatalf("second Close = %v, want ErrClosed", err)
	}
	if got := store.Snapshot(); got != final {
		t.Fatalf("snapshot after repeated Close = %+v, want %+v", got, final)
	}
}
