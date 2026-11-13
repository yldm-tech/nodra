package core

import (
	"errors"
	"maps"
	"math"
	"reflect"
	"slices"
	"testing"
)

func TestSelectorCollections(t *testing.T) {
	t.Run("slice default", func(t *testing.T) {
		store := New([]int{1})
		var values [][2][]int
		stop := SubscribeSelectorWith(store, func(v []int) []int { return v }, func(cur, prev []int) {
			values = append(values, [2][]int{slices.Clone(cur), slices.Clone(prev)})
		}, SelectorOptions[[]int]{Immediate: true})
		defer stop()
		requireNoError(t, store.Set([]int{1}))
		requireNoError(t, store.Set([]int{1, 2}))
		want := [][2][]int{{{1}, nil}, {{1, 2}, {1}}}
		if !reflect.DeepEqual(values, want) {
			t.Fatalf("values = %v", values)
		}
	})
	t.Run("map custom", func(t *testing.T) {
		store := New(map[string]int{"a": 1})
		var values []map[string]int
		stop := SubscribeSelectorWith(store, func(v map[string]int) map[string]int { return v }, func(cur, _ map[string]int) {
			values = append(values, maps.Clone(cur))
		}, SelectorOptions[map[string]int]{Equals: maps.Equal[map[string]int, map[string]int]})
		defer stop()
		requireNoError(t, store.Set(map[string]int{"a": 1}))
		requireNoError(t, store.Set(map[string]int{"a": 2}))
		if !reflect.DeepEqual(values, []map[string]int{{"a": 2}}) {
			t.Fatalf("values = %v", values)
		}
	})
	t.Run("slice convenience", func(t *testing.T) {
		store := New([]string{"a"})
		calls := 0
		stop := SubscribeSelector(store, func(v []string) []string { return v }, func([]string) { calls++ })
		defer stop()
		requireNoError(t, store.Set([]string{"a"}))
		requireNoError(t, store.Set([]string{"b"}))
		if calls != 1 {
			t.Fatalf("calls = %d", calls)
		}
	})
}

func TestSelectorDefaultEqualityPreservesComparableSemantics(t *testing.T) {
	for _, test := range []struct {
		name  string
		a, b  any
		equal bool
	}{
		{"nil", nil, nil, true},
		{"different types", 1, int64(1), false},
		{"slice in interface", []int{1}, []int{1}, true},
		{"map in interface", map[string]int{"a": 1}, map[string]int{"a": 2}, false},
		{"noncomparable nested interface", struct{ V any }{[]int{1}}, struct{ V any }{[]int{1}}, true},
		{"nil vs empty slice", []int(nil), []int{}, false},
		{"distinct pointers", new(int), new(int), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := New(test.a)
			calls := 0
			stop := SubscribeSelector(store, func(v any) any { return v }, func(any) { calls++ })
			defer stop()
			requireNoError(t, store.Set(test.b))
			if (calls == 0) != test.equal {
				t.Fatalf("calls=%d equal=%v", calls, test.equal)
			}
		})
	}
}

func TestSelectorCustomEqualityRunsBeforeDefaultComparison(t *testing.T) {
	store := New[any]([]int{1})
	calls := 0
	stop := SubscribeSelectorWith(store, func(v any) any { return v }, func(any, any) { calls++ }, SelectorOptions[any]{
		Equals: func(a, b any) bool { return slices.Equal(a.([]int), b.([]int)) },
	})
	defer stop()
	requireNoError(t, store.Set([]int{1}))
	requireNoError(t, store.Set([]int{2}))
	if calls != 1 {
		t.Fatalf("calls = %d", calls)
	}
	var nilCallback func([]int)
	SubscribeSelector(New([]int{}), func(v []int) []int { return v }, nilCallback)()
}

// checkSelectorEqualFor requires the equality chosen for V to agree with the reflective selectorEqual and with want, and to take the == fast path exactly when fast is set.
func checkSelectorEqualFor[V any](t *testing.T, a, b V, want, fast bool) {
	t.Helper()
	if got := holdsInterface(reflect.TypeFor[V]()); reflect.TypeFor[V]().Comparable() && got == fast {
		t.Errorf("%v: holdsInterface = %v, want fast path %v", reflect.TypeFor[V](), got, fast)
	}
	if got, reflective := selectorEqualFor[V]()(a, b), selectorEqual(a, b); got != want || reflective != want {
		t.Errorf("%v: selectorEqualFor = %v, selectorEqual = %v, want %v", reflect.TypeFor[V](), got, reflective, want)
	}
}

func TestSelectorEqualForMatchesReflectiveEquality(t *testing.T) {
	type projection struct {
		Title, Status string
		Count         int
	}
	type withInterface struct{ V any }
	type withPointerToInterface struct{ V *any }
	type nested struct {
		Inner [2]struct{ Err error }
	}
	first, second := new(int), new(int)
	var shared any = []int{1}
	nan := math.NaN()
	checkSelectorEqualFor(t, 1, 1, true, true)
	checkSelectorEqualFor(t, "a", "b", false, true)
	checkSelectorEqualFor(t, nan, nan, false, true)
	checkSelectorEqualFor(t, projection{"a", "b", 1}, projection{"a", "b", 1}, true, true)
	checkSelectorEqualFor(t, projection{"a", "b", 1}, projection{"a", "b", 2}, false, true)
	checkSelectorEqualFor(t, [2]float64{nan, 1}, [2]float64{nan, 1}, false, true)
	checkSelectorEqualFor(t, first, first, true, true)
	checkSelectorEqualFor(t, first, second, false, true)
	checkSelectorEqualFor(t, withPointerToInterface{&shared}, withPointerToInterface{&shared}, true, true)
	checkSelectorEqualFor[any](t, nil, nil, true, false)
	checkSelectorEqualFor[any](t, []int{1}, []int{1}, true, false)
	checkSelectorEqualFor[any](t, 1, int64(1), false, false)
	checkSelectorEqualFor[any](t, nan, nan, false, false)
	checkSelectorEqualFor(t, withInterface{[]int{1}}, withInterface{[]int{1}}, true, false)
	checkSelectorEqualFor(t, withInterface{map[string]int{"a": 1}}, withInterface{map[string]int{"a": 2}}, false, false)
	checkSelectorEqualFor(t, [1]any{[]int{1}}, [1]any{[]int{2}}, false, false)
	checkSelectorEqualFor(t, nested{}, nested{}, true, false)
	checkSelectorEqualFor(t, errors.ErrUnsupported, errors.ErrUnsupported, true, false)
	checkSelectorEqualFor(t, []int{1}, []int{1}, true, false)
	checkSelectorEqualFor(t, map[string]int{"a": 1}, map[string]int{"a": 2}, false, false)
}

func TestSelectorDefaultEqualityAllocatesNothingForPlainComparableValues(t *testing.T) {
	type state struct {
		Title, Status string
		Count, Edits  int
	}
	type projection struct {
		Title, Status string
		Count         int
	}
	project := func(value state) projection { return projection{value.Title, value.Status, value.Count} }
	allocsPerUpdate := func(options SelectorOptions[projection]) float64 {
		store := New(state{Title: "title", Status: "open"})
		defer func() { _ = store.Close() }()
		for range 10 {
			defer SubscribeSelectorWith(store, project, func(projection, projection) {}, options)()
		}
		return testing.AllocsPerRun(100, func() {
			requireNoError(t, store.Update(func(value *state) { value.Edits++ }))
		})
	}
	// The selectors see an unchanged projection, so the only difference between the two runs is the equality function.
	explicit := allocsPerUpdate(SelectorOptions[projection]{Equals: func(a, b projection) bool { return a == b }})
	if def := allocsPerUpdate(SelectorOptions[projection]{}); def > explicit {
		t.Fatalf("default equality allocates %.0f per update across 10 selectors, want %.0f like an explicit ==", def, explicit)
	}
}
