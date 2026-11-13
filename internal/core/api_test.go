package core

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

type apiState struct {
	Count int
	Name  string
}

func TestCreatePatchResetAndStateAliases(t *testing.T) {
	store := Create(apiState{Count: 1, Name: "initial"})
	if store.State().Count != 1 {
		t.Fatal("initial state")
	}
	if err := store.Patch(func(state *apiState) { state.Count++; state.Name = "patched" }); err != nil {
		t.Fatal(err)
	}
	if store.State().Count != 2 || store.State().Name != "patched" {
		t.Fatalf("state=%+v", store.State())
	}
	if err := store.Reset(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(store.State(), apiState{Count: 1, Name: "initial"}) {
		t.Fatalf("reset=%+v", store.State())
	}
}

func TestDerivedGetterCachesAndSubscribes(t *testing.T) {
	store := Create(apiState{Count: 2})
	calls := 0
	derived := Derive(store, func(state apiState) int { calls++; return state.Count * 2 }, func(a, b int) bool { return a == b })
	// The second Get must be served from the cache without recomputing.
	first, second := derived.Get(), derived.Get()
	if first != 4 || second != 4 || calls != 1 {
		t.Fatalf("values=%d,%d calls=%d", first, second, calls)
	}
	var values []int
	stop := derived.Subscribe(func(value int) { values = append(values, value) })
	_ = store.Patch(func(state *apiState) { state.Name = "same getter" })
	_ = store.Patch(func(state *apiState) { state.Count = 3 })
	stop()
	if !reflect.DeepEqual(values, []int{6}) {
		t.Fatalf("values=%v", values)
	}
}

func TestSelectorWithPrevious(t *testing.T) {
	store := Create(apiState{Count: 1})
	var got [][2]int
	stop := SubscribeSelectorWith(store, func(state apiState) int { return state.Count }, func(current, previous int) { got = append(got, [2]int{current, previous}) }, SelectorOptions[int]{Immediate: true})
	_ = store.Set(apiState{Count: 2})
	stop()
	if !reflect.DeepEqual(got, [][2]int{{1, 0}, {2, 1}}) {
		t.Fatalf("got=%v", got)
	}
}

func TestActionsAndMutationMetadata(t *testing.T) {
	store := Create(apiState{})
	var actionNames []string
	stopActions := store.SubscribeActions(func(event ActionEvent) { actionNames = append(actionNames, event.Name) })
	var mutations []Mutation[apiState]
	stopMutations := store.SubscribeMutations(func(event Mutation[apiState]) { mutations = append(mutations, event) })
	if err := store.Do("increment", func(state *apiState) error { state.Count++; return nil }); err != nil {
		t.Fatal(err)
	}
	stopActions()
	stopMutations()
	if !reflect.DeepEqual(actionNames, []string{"increment"}) {
		t.Fatalf("actions=%v", actionNames)
	}
	if len(mutations) != 1 || mutations[0].Type != MutationAction || mutations[0].Name != "increment" || mutations[0].Previous.Count != 0 || mutations[0].Value.Count != 1 {
		t.Fatalf("mutations=%+v", mutations)
	}
}

func TestNamedActionFactory(t *testing.T) {
	store := Create(0)
	increment := store.Action("increment", func(value *int) error { *value++; return nil })
	if err := increment(); err != nil || store.Get() != 1 {
		t.Fatalf("value=%d err=%v", store.Get(), err)
	}
}

func TestDerivedDoneClosesWithTheDerivedValue(t *testing.T) {
	store := New(1)
	defer func() { _ = store.Close() }()
	derived := Derive[int, int](store, func(v int) int { return v * 2 }, nil)
	done := derived.Done()
	if done == nil {
		t.Fatal("Done is nil for an open derived value")
	}
	select {
	case <-done:
		t.Fatal("Done closed before Close")
	default:
	}
	requireNoError(t, derived.Close())
	select {
	case <-done:
	default:
		t.Fatal("Done did not close with Close")
	}
	if derived.Done() != done {
		t.Fatal("Done returned a different channel after Close")
	}

	invalid := []interface{ Done() <-chan struct{} }{
		Derive[int, int](nil, func(v int) int { return v }, nil),
		Derive[int, int](store, nil, nil),
		DeriveProjection[int, int](nil, func(v int) int { return v }, nil),
		Combine[int, int, int](nil, store, func(a, b int) int { return a + b }, nil),
	}
	for i, d := range invalid {
		select {
		case <-d.Done():
		default:
			t.Fatalf("invalid derived value %d is not done", i)
		}
	}
	var missing *Derived[int, int]
	if missing.Done() != nil {
		t.Fatal("nil derived value returned a Done channel")
	}
}

func TestDerivedWithNilEqualsNotifiesEveryAcceptedVersion(t *testing.T) {
	store := New(apiState{})
	selectCount := func(s apiState) int { return s.Count }
	unfiltered := Derive[apiState, int](store, selectCount, nil)
	filtered := Derive[apiState, int](store, selectCount, func(a, b int) bool { return a == b })
	defer func() { _ = unfiltered.Close(); _ = filtered.Close() }()
	var unfilteredCalls, filteredCalls int
	unfiltered.Subscribe(func(int) { unfilteredCalls++ })
	filtered.Subscribe(func(int) { filteredCalls++ })
	for _, name := range []string{"a", "b", "c"} {
		requireNoError(t, store.Update(func(s *apiState) { s.Name = name }))
	}
	if unfilteredCalls != 3 || filteredCalls != 0 {
		t.Fatalf("notifications with nil equals = %d, with equals = %d; want 3 and 0", unfilteredCalls, filteredCalls)
	}
	if unfiltered.Snapshot().Version != store.Version() || filtered.Snapshot().Version != store.Version() {
		t.Fatal("derived versions did not follow the source")
	}
}

// lockedObservable is a source whose unsubscribe takes the same lock a writer holds while it runs user code, as a Store's unsubscribe takes the store lock that Update holds during its callback.
type lockedObservable struct {
	mu       sync.Mutex
	stopping chan struct{}
	stopped  bool
}

func (o *lockedObservable) Snapshot() Snapshot[int] { return Snapshot[int]{Value: 1, Version: 1} }

func (o *lockedObservable) SubscribeSnapshot(func(Snapshot[int])) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			close(o.stopping)
			o.mu.Lock()
			o.stopped = true
			o.mu.Unlock()
		})
	}
}

func TestDerivedCloseDoesNotHoldItsLockWhileUnsubscribing(t *testing.T) {
	source := &lockedObservable{stopping: make(chan struct{})}
	derived := Derive[int, int](source, func(v int) int { return v * 2 }, nil)
	// The writer holds the source lock, waits until Close is unsubscribing, and then reads the derived value, like an Update callback that reads a value derived from its store.
	source.mu.Lock()
	read := make(chan int, 1)
	go func() {
		defer source.mu.Unlock()
		<-source.stopping
		read <- derived.Get()
	}()
	closed := make(chan error, 1)
	go func() { closed <- derived.Close() }()
	select {
	case got := <-read:
		if got != 2 {
			t.Fatalf("Get during Close = %d, want 2", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("deadlock: Close holds the derived lock while unsubscribing, and the writer holds the source lock while reading the derived value")
	}
	requireNoError(t, <-closed)
	select {
	case <-derived.Done():
	default:
		t.Fatal("Done did not close with Close")
	}
}

func TestDerivedCloseRacingAnUpdateThatReadsIt(t *testing.T) {
	store := New(1)
	doubled := Derive[int, int](store, func(v int) int { return v * 2 }, nil)
	entered, proceed, abort, updated := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan error, 1)
	// On a regression the update gives up instead of staying blocked: it returns, so the store lock and the transaction gate are released, the old Close can finish unsubscribing, and later tests in the package that take the gate still run. The derived read runs in its own goroutine so the update can give up even while that read is blocked on the derived lock.
	go func() {
		updated <- store.Update(func(v *int) {
			close(entered)
			select {
			case <-proceed:
			case <-abort:
				return
			}
			read := make(chan int, 1)
			go func() { read <- doubled.Get() }()
			select {
			case got := <-read:
				*v = got + 1
			case <-abort:
			}
		})
	}()
	fail := func(msg string) {
		close(abort)
		if err := <-updated; err != nil {
			t.Errorf("update after abort: %v", err)
		}
		t.Fatal(msg)
	}
	<-entered
	closed := make(chan error, 1)
	go func() { closed <- doubled.Close() }()
	// Close marks the value closed before it unsubscribes, so Done shows it has released the derived lock or is about to wait for the store lock.
	select {
	case <-doubled.Done():
	case <-time.After(5 * time.Second):
		fail("deadlock: Close is waiting for the store lock while it holds the derived lock")
	}
	close(proceed)
	select {
	case err := <-updated:
		requireNoError(t, err)
	case <-time.After(5 * time.Second):
		fail("deadlock: the update cannot read the derived value while Close holds it")
	}
	requireNoError(t, <-closed)
	if got := store.Get(); got != 3 {
		t.Fatalf("store = %d, want 3", got)
	}
	requireNoError(t, store.Close())
}
