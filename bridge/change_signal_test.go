package bridge_test

import (
	"context"
	"fmt"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/yldm-tech/nodra"
	"github.com/yldm-tech/nodra/bridge"
)

// clonesPerUpdateWithWatches counts the clones a NewWithClone store makes per update while watches pages follow the exposure of source(store), waiting after each update until every watch has sent its frame.
func clonesPerUpdateWithWatches(t *testing.T, watches int, source func(*nodra.Store[[]int]) nodra.Observable[[]int]) int64 {
	t.Helper()
	const updates = 10
	var clones atomic.Int64
	store := nodra.NewWithClone(make([]int, 64), func(value []int) []int {
		clones.Add(1)
		return slices.Clone(value)
	})
	defer func() { _ = store.Close() }()
	b := bridge.New(bridge.Options{})
	defer b.Close()
	if _, err := bridge.ExposeProjection(b, "first", source(store), func(value []int) int { return value[0] }); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	pages := make([]*frames, watches)
	done := make([]<-chan error, watches)
	for i := range pages {
		pages[i] = newFrames()
		done[i] = watch(ctx, b, "first", 0, pages[i])
		pages[i].wait(t, 1)
	}
	clones.Store(0)
	for n := 1; n <= updates; n++ {
		if err := store.Update(func(value *[]int) { (*value)[0]++ }); err != nil {
			t.Fatal(err)
		}
		for _, page := range pages {
			page.wait(t, n+1)
		}
	}
	got := clones.Load()
	cancel()
	for _, d := range done {
		if err := result(t, d); err != nil {
			t.Fatalf("a canceled watch returned %v", err)
		}
	}
	return got / updates
}

func TestAWatchDoesNotCloneTheStateForItsChangeSignal(t *testing.T) {
	base := clonesPerUpdateWithWatches(t, 0, func(store *nodra.Store[[]int]) nodra.Observable[[]int] { return store })
	for _, test := range []struct {
		name   string
		source func(*nodra.Store[[]int]) nodra.Observable[[]int]
	}{
		{"store", func(store *nodra.Store[[]int]) nodra.Observable[[]int] { return store }},
		{"read-only", func(store *nodra.Store[[]int]) nodra.Observable[[]int] { return store.ReadOnly() }},
		{"read-only pointer", func(store *nodra.Store[[]int]) nodra.Observable[[]int] {
			view := store.ReadOnly()
			return &view
		}},
	} {
		for _, watches := range []int{1, 4} {
			t.Run(fmt.Sprintf("%s/watches=%d", test.name, watches), func(t *testing.T) {
				// Each watch still reads one snapshot per frame it sends; the change signal itself must add nothing.
				got := clonesPerUpdateWithWatches(t, watches, test.source)
				if want := base + int64(watches); got != want {
					t.Fatalf("clones per update = %d, want %d (%d without watches)", got, want, base)
				}
			})
		}
	}
}

// overridingStore is an application Observable that embeds a Store and defines its own SubscribeSnapshot.
type overridingStore struct {
	*nodra.Store[int]
	subscribed atomic.Int32
}

func (s *overridingStore) SubscribeSnapshot(fn func(nodra.Snapshot[int])) func() {
	s.subscribed.Add(1)
	return s.Store.SubscribeSnapshot(fn)
}

func TestAWatchOfAnApplicationObservableUsesItsSubscribeSnapshot(t *testing.T) {
	source := &overridingStore{Store: nodra.New(1)}
	defer func() { _ = source.Close() }()
	b := bridge.New(bridge.Options{})
	defer b.Close()
	if _, err := bridge.Expose[int](b, "count", source); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	f := newFrames()
	done := watch(ctx, b, "count", 0, f)
	f.wait(t, 1)
	if err := source.Set(2); err != nil {
		t.Fatal(err)
	}
	if got := f.wait(t, 2); decode[int](t, got[1]) != 2 {
		t.Fatalf("frames = %+v", got)
	}
	cancel()
	if err := result(t, done); err != nil {
		t.Fatalf("a canceled watch returned %v", err)
	}
	if got := source.subscribed.Load(); got != 1 {
		t.Fatalf("SubscribeSnapshot calls = %d, want 1", got)
	}
}
