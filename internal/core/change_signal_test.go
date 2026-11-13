package core

import (
	"context"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
)

// countingCloneStore returns a NewWithClone store whose clone function counts its calls.
func countingCloneStore() (*Store[[]int], *atomic.Int64) {
	clones := new(atomic.Int64)
	store := NewWithClone(make([]int, 64), func(value []int) []int {
		clones.Add(1)
		return slices.Clone(value)
	})
	return store, clones
}

// clonesPerUpdate counts the clones made by updates of a NewWithClone store after setup has subscribed its helper.
func clonesPerUpdate(t *testing.T, setup func(*Store[[]int])) int64 {
	t.Helper()
	const updates = 10
	store, clones := countingCloneStore()
	t.Cleanup(func() { _ = store.Close() })
	setup(store)
	clones.Store(0)
	for range updates {
		requireNoError(t, store.Update(func(value *[]int) { (*value)[0]++ }))
	}
	return clones.Load() / updates
}

func TestChangeSignalHelpersDoNotCloneTheState(t *testing.T) {
	base := clonesPerUpdate(t, func(*Store[[]int]) {})
	for _, test := range []struct {
		name string
		// extra counts the clones the helper itself needs per update, such as the snapshot a synchronous save writes.
		extra int64
		setup func(*testing.T, *Store[[]int])
	}{
		{"InvalidateTarget store", 0, func(t *testing.T, store *Store[[]int]) {
			t.Cleanup(InvalidateTarget(store, new(invalidationTargetFake)))
		}},
		{"InvalidateTarget read-only", 0, func(t *testing.T, store *Store[[]int]) {
			t.Cleanup(InvalidateTarget(store.ReadOnly(), new(invalidationTargetFake)))
		}},
		{"InvalidateTarget read-only pointer", 0, func(t *testing.T, store *Store[[]int]) {
			view := store.ReadOnly()
			t.Cleanup(InvalidateTarget(&view, new(invalidationTargetFake)))
		}},
		{"Invalidate", 0, func(t *testing.T, store *Store[[]int]) {
			t.Cleanup(Invalidate(store, &mygo.Window{}))
		}},
		{"AutoSave", 1, func(t *testing.T, store *Store[[]int]) {
			t.Cleanup(NewPersistence(store, &memoryStorage{}, jsonCodec[[]int]{}).AutoSave(context.Background()))
		}},
		{"AutoSaveWithErrors", 1, func(t *testing.T, store *Store[[]int]) {
			_, stop := NewPersistence(store, &memoryStorage{}, jsonCodec[[]int]{}).AutoSaveWithErrors(context.Background())
			t.Cleanup(stop)
		}},
		{"AutoSaveAsync", 0, func(t *testing.T, store *Store[[]int]) {
			// The hour-long debounce keeps the worker from saving, and so from cloning, while updates are counted.
			async, err := NewPersistence(store, &memoryStorage{}, jsonCodec[[]int]{}).AutoSaveAsync(context.Background(), AsyncOptions{Debounce: time.Hour})
			requireNoError(t, err)
			t.Cleanup(func() { _ = async.Close(context.Background()) })
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := clonesPerUpdate(t, func(store *Store[[]int]) { test.setup(t, store) })
			if want := base + test.extra; got != want {
				t.Fatalf("clones per update = %d, want %d (%d without the helper)", got, want, base)
			}
		})
	}
}

func TestChangeSignalHelpersStillObserveEveryPublication(t *testing.T) {
	store, _ := countingCloneStore()
	defer func() { _ = store.Close() }()
	direct, readOnly := new(invalidationTargetFake), new(invalidationTargetFake)
	defer InvalidateTarget(store, direct)()
	defer InvalidateTarget(store.ReadOnly(), readOnly)()
	storage := &memoryStorage{}
	defer NewPersistence(store, storage, jsonCodec[[]int]{}).AutoSave(context.Background())()
	async, err := NewPersistence(store, &memoryStorage{}, jsonCodec[[]int]{}).AutoSaveAsync(context.Background(), AsyncOptions{Debounce: time.Hour})
	requireNoError(t, err)
	defer func() { _ = async.Close(context.Background()) }()
	for range 3 {
		requireNoError(t, store.Update(func(value *[]int) { (*value)[0]++ }))
	}
	if direct.calls != 3 || readOnly.calls != 3 {
		t.Fatalf("invalidations = %d store, %d read-only, want 3 each", direct.calls, readOnly.calls)
	}
	saved, err := jsonCodec[[]int]{}.Decode(storage.data)
	requireNoError(t, err)
	if saved[0] != 3 {
		t.Fatalf("AutoSave wrote %d, want the latest value 3", saved[0])
	}
	async.mu.Lock()
	requested := async.requested
	async.mu.Unlock()
	if requested != store.Version() {
		t.Fatalf("AutoSaveAsync requested version %d, want %d", requested, store.Version())
	}
}

func TestSubscribeVersionsDropsOutOfOrderNotifications(t *testing.T) {
	store := New(0)
	defer func() { _ = store.Close() }()
	var delivered []uint64
	defer store.subscribeVersions(func(version uint64) { delivered = append(delivered, version) })()
	store.mu.RLock()
	listeners := store.listenersCopyLocked()
	store.mu.RUnlock()
	// Concurrent writers can deliver an older publication after a newer one; Subscribe drops it, and so must the borrowed change signal.
	for _, version := range []uint64{5, 3, 5, 6} {
		store.notify(listeners, Snapshot[int]{Version: version})
	}
	if !slices.Equal(delivered, []uint64{5, 6}) {
		t.Fatalf("delivered versions = %v, want [5 6]", delivered)
	}
}

// subscriptionCountingStore is an application Observable that embeds a Store and wraps its SubscribeSnapshot.
type subscriptionCountingStore struct {
	*Store[int]
	subscriptions int
}

func (s *subscriptionCountingStore) SubscribeSnapshot(fn func(Snapshot[int])) func() {
	s.subscriptions++
	return s.Store.SubscribeSnapshot(fn)
}

func TestInvalidateTargetKeepsApplicationSubscribeSnapshot(t *testing.T) {
	source := &subscriptionCountingStore{Store: New(0)}
	defer func() { _ = source.Close() }()
	target := new(invalidationTargetFake)
	defer InvalidateTarget[int](source, target)()
	defer Invalidate[int](source, &mygo.Window{})()
	if source.subscriptions != 2 {
		t.Fatalf("application SubscribeSnapshot calls = %d, want 2", source.subscriptions)
	}
	requireNoError(t, source.Set(1))
	if target.calls != 1 {
		t.Fatalf("invalidations = %d, want 1", target.calls)
	}
}
