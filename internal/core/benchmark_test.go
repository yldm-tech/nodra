package core

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
)

func BenchmarkActionScheduling(b *testing.B) {
	for _, policy := range []ActionPolicy{LatestWins, QueueAll} {
		b.Run(fmt.Sprintf("policy=%d", policy), func(b *testing.B) {
			store := New(0)
			action, err := NewAsyncActionWithOptions(store, "increment", func(_ context.Context, value int, progress func(float64)) (int, error) {
				progress(.5)
				return value, nil
			}, func(state *int, value int) error { *state += value; return nil }, ActionOptions[int]{Policy: policy, MaxConcurrent: 4})
			if err != nil {
				b.Fatal(err)
			}
			b.Cleanup(func() { _ = action.Close(context.Background()); _ = store.Close() })
			b.ReportAllocs()
			for b.Loop() {
				task, err := action.Run(context.Background(), 1)
				if err != nil {
					b.Fatal(err)
				}
				if err := task.Wait(context.Background()); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkDerivedProjectionPipeline(b *testing.B) {
	for _, projected := range []bool{false, true} {
		b.Run(fmt.Sprintf("projection=%v", projected), func(b *testing.B) {
			store := NewWithClone(make([]int, 10000), slices.Clone[[]int])
			project := func(value []int) int { return value[0] }
			var derived *Derived[[]int, int]
			if projected {
				derived = DeriveProjection(store, project, nil)
			} else {
				derived = Derive(store, project, nil)
			}
			chained := Derive(derived, func(value int) int { return value * 2 }, nil)
			b.Cleanup(func() { _ = chained.Close(); _ = derived.Close(); _ = store.Close() })
			b.ReportAllocs()
			for b.Loop() {
				if err := store.Update(func(value *[]int) { (*value)[0]++ }); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkCombinedValues(b *testing.B) {
	a, bSource := New(0), New(1)
	combined := Combine(a, bSource, func(a, b int) int { return a + b }, nil)
	b.Cleanup(func() { _ = combined.Close(); _ = a.Close(); _ = bSource.Close() })
	b.ReportAllocs()
	for b.Loop() {
		if err := a.Update(func(value *int) { *value++ }); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStoreUpdate(b *testing.B) {
	for _, listeners := range []int{0, 1, 100} {
		b.Run(fmt.Sprintf("listeners=%d", listeners), func(b *testing.B) {
			store := New(0)
			for range listeners {
				store.Subscribe(func(Snapshot[int]) {})
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_ = store.Update(func(v *int) { *v++ })
			}
		})
	}
}

func BenchmarkClonedMapUpdate(b *testing.B) {
	initial := make(map[int]int, 1000)
	for n := range 1000 {
		initial[n] = n
	}
	store := NewWithClone(initial, maps.Clone[map[int]int])
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = store.Update(func(v *map[int]int) { (*v)[0]++ })
	}
}

func BenchmarkSliceSelector(b *testing.B) {
	for _, custom := range []bool{false, true} {
		b.Run(fmt.Sprintf("custom=%v", custom), func(b *testing.B) {
			values := make([]int, 1000)
			store := New(values)
			options := SelectorOptions[[]int]{}
			if custom {
				options.Equals = slices.Equal[[]int]
			}
			stop := SubscribeSelectorWith(store, func(v []int) []int { return slices.Clone(v) }, func([]int, []int) {}, options)
			defer stop()
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_ = store.Set(values)
			}
		})
	}
}

func BenchmarkLargeStateReads(b *testing.B) {
	store := NewWithClone(make([]int, 10000), slices.Clone[[]int])
	for _, projected := range []bool{false, true} {
		b.Run(fmt.Sprintf("projection=%v", projected), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if projected {
					_ = Select(store, func(v []int) int { return len(v) })
				} else {
					_ = len(store.Get())
				}
			}
		})
	}
}

func BenchmarkLargeStateSelectors(b *testing.B) {
	for _, projected := range []bool{false, true} {
		for _, listeners := range []int{1, 10} {
			b.Run(fmt.Sprintf("projection=%v/listeners=%d", projected, listeners), func(b *testing.B) {
				store := NewWithClone(make([]int, 10000), slices.Clone[[]int])
				project := func(v []int) int { return v[0] }
				for range listeners {
					if projected {
						SubscribeProjection(store, project, func(int) {})
					} else {
						SubscribeSelector(store, project, func(int) {})
					}
				}
				b.ReportAllocs()
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						_ = store.Update(func(v *[]int) { (*v)[0]++ })
					}
				})
			})
		}
	}
}

// BenchmarkSelectorDefaultEquality measures the default selector equality on a small struct projection whose value does not change, so every notification compares and none calls back.
func BenchmarkSelectorDefaultEquality(b *testing.B) {
	type state struct {
		Title, Status string
		Count, Edits  int
	}
	type projection struct {
		Title, Status string
		Count         int
	}
	project := func(value state) projection { return projection{value.Title, value.Status, value.Count} }
	for _, selectors := range []int{1, 20} {
		b.Run(fmt.Sprintf("selectors=%d", selectors), func(b *testing.B) {
			store := New(state{Title: "title", Status: "open"})
			b.Cleanup(func() { _ = store.Close() })
			for range selectors {
				SubscribeSelector(store, project, func(projection) {})
			}
			b.ReportAllocs()
			for b.Loop() {
				_ = store.Update(func(value *state) { value.Edits++ })
			}
		})
	}
}

// BenchmarkChangeSignalClones counts the clones of a large NewWithClone state per Update with one helper subscribed that uses a publication only as a change signal. AutoSave and AutoSaveWithErrors save synchronously, and each save takes one snapshot of its own.
func BenchmarkChangeSignalClones(b *testing.B) {
	for _, helper := range []struct {
		name      string
		subscribe func(*testing.B, *Store[[]int])
	}{
		{"none", func(*testing.B, *Store[[]int]) {}},
		{"InvalidateTarget", func(b *testing.B, store *Store[[]int]) {
			b.Cleanup(InvalidateTarget(store, new(invalidationTargetFake)))
		}},
		{"Invalidate", func(b *testing.B, store *Store[[]int]) {
			b.Cleanup(Invalidate(store, &mygo.Window{}))
		}},
		{"AutoSave", func(b *testing.B, store *Store[[]int]) {
			b.Cleanup(NewPersistence(store, discardStorage{}, discardCodec{}).AutoSave(context.Background()))
		}},
		{"AutoSaveWithErrors", func(b *testing.B, store *Store[[]int]) {
			_, stop := NewPersistence(store, discardStorage{}, discardCodec{}).AutoSaveWithErrors(context.Background())
			b.Cleanup(stop)
		}},
		{"AutoSaveAsync", func(b *testing.B, store *Store[[]int]) {
			async, err := NewPersistence(store, discardStorage{}, discardCodec{}).AutoSaveAsync(context.Background(), AsyncOptions{Debounce: time.Hour})
			if err != nil {
				b.Fatal(err)
			}
			b.Cleanup(func() { _ = async.Close(context.Background()) })
		}},
	} {
		b.Run(helper.name, func(b *testing.B) {
			var clones atomic.Int64
			store := NewWithClone(make([]int, 10000), func(value []int) []int {
				clones.Add(1)
				return slices.Clone(value)
			})
			b.Cleanup(func() { _ = store.Close() })
			helper.subscribe(b, store)
			clones.Store(0)
			b.ReportAllocs()
			for b.Loop() {
				_ = store.Update(func(value *[]int) { (*value)[0]++ })
			}
			b.ReportMetric(float64(clones.Load())/float64(b.N), "clones/op")
		})
	}
}

// discardStorage and discardCodec keep persistence I/O and encoding out of the clone benchmarks.
type discardStorage struct{}

func (discardStorage) Load(context.Context) ([]byte, error) { return nil, nil }
func (discardStorage) Save(context.Context, []byte) error   { return nil }

type discardCodec struct{}

func (discardCodec) Encode([]int) ([]byte, error) { return nil, nil }
func (discardCodec) Decode([]byte) ([]int, error) { return nil, nil }
