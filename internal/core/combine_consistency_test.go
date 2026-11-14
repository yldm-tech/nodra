package core

import (
	"cmp"
	"fmt"
	"math/rand/v2"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// These tests hold one Combine invariant: x and y are written only together, so every committed state has x+y == 1000, and neither a projector nor a subscriber may ever see another sum.

const tornTotal = 1000

// tornRecorder counts projector calls and deliveries that saw a sum no committed state had.
type tornRecorder struct {
	mu    sync.Mutex
	count int
	seen  []string
}

func (r *tornRecorder) check(where string, sum int) {
	if sum == tornTotal {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.count++
	if len(r.seen) < 10 {
		r.seen = append(r.seen, fmt.Sprintf("%s saw %d", where, sum))
	}
}

func (r *tornRecorder) sum(where string) func(a, b int) int {
	return func(a, b int) int {
		r.check(where+" projector", a+b)
		return a + b
	}
}

func (r *tornRecorder) verify(t *testing.T) {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.count > 0 {
		t.Fatalf("%d torn pairs, first ones: %v", r.count, r.seen)
	}
}

// valueGate blocks the nth copy a NewWithClone store makes of each gated value until that value is released, and reports when it blocks. Atomic2 copies the new value once to commit it and once more for each subscriber it notifies, so gating the second copy holds a transaction between notifying its first store and its second.
type valueGate struct {
	mu      sync.Mutex
	nth     int
	counts  map[int]int
	held    map[int]chan struct{}
	release map[int]chan struct{}
}

func newValueGate(nth int, values ...int) *valueGate {
	gate := &valueGate{nth: nth, counts: map[int]int{}, held: map[int]chan struct{}{}, release: map[int]chan struct{}{}}
	for _, value := range values {
		gate.held[value] = make(chan struct{})
		gate.release[value] = make(chan struct{})
	}
	return gate
}

func (g *valueGate) clone(value int) int {
	g.mu.Lock()
	g.counts[value]++
	hold := g.counts[value] == g.nth && g.release[value] != nil
	g.mu.Unlock()
	if hold {
		close(g.held[value])
		<-g.release[value]
	}
	return value
}

// TestCombineDoesNotMixConcurrentAtomic2Transactions holds T1 after it has delivered x=990 and lets T2 deliver x=980 before T1 delivers y=10. The pair (980, 10) never existed.
func TestCombineDoesNotMixConcurrentAtomic2Transactions(t *testing.T) {
	var torn tornRecorder
	gate := newValueGate(2, 10, 20)
	x, y := New(tornTotal), NewWithClone(0, gate.clone)
	sum := Combine(x, y, torn.sum("Combine(x, y)"), nil)
	defer func() { _ = sum.Close() }()
	var mu sync.Mutex
	var delivered []int
	sum.Subscribe(func(v int) {
		torn.check("subscriber", v)
		mu.Lock()
		delivered = append(delivered, v)
		mu.Unlock()
	})
	move := func(a, b *int) error { *a -= 10; *b += 10; return nil }
	first, second := make(chan error, 1), make(chan error, 1)
	go func() { first <- Atomic2(x, y, move) }()
	<-gate.held[10]
	go func() { second <- Atomic2(x, y, move) }()
	<-gate.held[20]
	close(gate.release[10])
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	close(gate.release[20])
	if err := <-second; err != nil {
		t.Fatal(err)
	}
	torn.verify(t)
	if got := sum.Get(); got != tornTotal {
		t.Fatalf("Combine = %d after both transactions, want %d", got, tornTotal)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(delivered) == 0 {
		t.Fatal("Combine never published the settled pair")
	}
}

// heldMutations blocks the mutation listener of a store for each gated value until it is released. Atomic2 calls mutation listeners after notifying both stores and before it settles derived values, so this holds a transaction with its derived values still unsettled.
func heldMutations(store *Store[int], values ...int) (held, release map[int]chan struct{}) {
	held, release = map[int]chan struct{}{}, map[int]chan struct{}{}
	for _, value := range values {
		held[value], release[value] = make(chan struct{}), make(chan struct{})
	}
	store.SubscribeMutations(func(m Mutation[int]) {
		if release[m.Value] != nil {
			close(held[m.Value])
			<-release[m.Value]
		}
	})
	return held, release
}

// TestCombineOverDerivedDoesNotMixConcurrentAtomic2Transactions lets T2 deliver y=20 to the Combine while T1 still has to settle Derive(x)=990; settling T1 must not publish (990, 20).
func TestCombineOverDerivedDoesNotMixConcurrentAtomic2Transactions(t *testing.T) {
	var torn tornRecorder
	x, y := New(tornTotal), New(0)
	dx := Derive[int, int](x, func(v int) int { return v }, nil)
	defer func() { _ = dx.Close() }()
	sum := Combine[int, int, int](dx, y, torn.sum("Combine(Derive(x), y)"), nil)
	defer func() { _ = sum.Close() }()
	sum.Subscribe(func(v int) { torn.check("subscriber", v) })
	held, release := heldMutations(y, 10, 20)
	move := func(a, b *int) error { *a -= 10; *b += 10; return nil }
	first, second := make(chan error, 1), make(chan error, 1)
	go func() { first <- Atomic2(x, y, move) }()
	<-held[10]
	go func() { second <- Atomic2(x, y, move) }()
	<-held[20]
	close(release[10])
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	close(release[20])
	if err := <-second; err != nil {
		t.Fatal(err)
	}
	torn.verify(t)
	if got := sum.Get(); got != tornTotal {
		t.Fatalf("Combine = %d after both transactions, want %d", got, tornTotal)
	}
}

// hookedObservable is an application Observable over a Store that runs a test hook once, from Snapshot before it reads the Store or after SubscribeSnapshot registers.
type hookedObservable struct {
	store          *Store[int]
	once           sync.Once
	beforeSnapshot func()
	onSubscribe    func()
}

func (h *hookedObservable) Snapshot() Snapshot[int] {
	if h.beforeSnapshot != nil {
		h.once.Do(h.beforeSnapshot)
	}
	return h.store.Snapshot()
}

func (h *hookedObservable) SubscribeSnapshot(fn func(Snapshot[int])) func() {
	stop := h.store.SubscribeSnapshot(fn)
	if h.onSubscribe != nil {
		h.once.Do(h.onSubscribe)
	}
	return stop
}

// slowSubscriber blocks the first notification y sends until release is closed and reports when it does, which holds that dispatch with its derived values unsettled.
func slowSubscriber(y *Store[int]) (inDispatch, release chan struct{}) {
	inDispatch, release = make(chan struct{}), make(chan struct{})
	var once sync.Once
	y.Subscribe(func(Snapshot[int]) { once.Do(func() { close(inDispatch); <-release }) })
	return inDispatch, release
}

// TestCombineConstructedDuringADispatchReadsACommittedPair commits Atomic2 after Combine has read x and before it reads y. The second input is an application Observable only so the test can run code between the two reads; Combine must notice that x changed.
func TestCombineConstructedDuringADispatchReadsACommittedPair(t *testing.T) {
	var torn tornRecorder
	x, y := New(100), New(900)
	inDispatch, release := slowSubscriber(y)
	done := make(chan error, 1)
	second := &hookedObservable{store: y, beforeSnapshot: func() {
		go func() { done <- Atomic2(x, y, func(a, b *int) error { *a += 10; *b -= 10; return nil }) }()
		<-inDispatch
	}}
	sum := Combine[int, int, int](x, second, torn.sum("Combine(x, y)"), nil)
	defer func() { _ = sum.Close() }()
	got := sum.Get()
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	torn.verify(t)
	if got != tornTotal {
		t.Fatalf("Combine.Get right after construction = %d, want %d", got, tornTotal)
	}
	if got := sum.Get(); got != tornTotal {
		t.Fatalf("Combine.Get after the dispatch = %d, want %d", got, tornTotal)
	}
}

// TestCombineConstructedOverAnUnsettledDerivedReadsACommittedPair builds Combine(Derive(x), y) while Atomic2 has committed x and y but has not settled Derive(x) yet.
func TestCombineConstructedOverAnUnsettledDerivedReadsACommittedPair(t *testing.T) {
	var torn tornRecorder
	x, y := New(100), New(900)
	dx := Derive[int, int](x, func(v int) int { return v }, nil)
	defer func() { _ = dx.Close() }()
	inDispatch, release := slowSubscriber(y)
	done := make(chan error, 1)
	go func() { done <- Atomic2(x, y, func(a, b *int) error { *a += 10; *b -= 10; return nil }) }()
	<-inDispatch
	if got := dx.Get(); got != 100 {
		t.Fatalf("Derive(x) settled before its transaction was released: %d", got)
	}
	sum := Combine[int, int, int](dx, y, torn.sum("Combine(Derive(x), y)"), nil)
	defer func() { _ = sum.Close() }()
	sum.Subscribe(func(v int) { torn.check("subscriber", v) })
	got := sum.Get()
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	torn.verify(t)
	if got != tornTotal {
		t.Fatalf("Combine.Get right after construction = %d, want %d", got, tornTotal)
	}
	if got := sum.Get(); got != tornTotal {
		t.Fatalf("Combine.Get after the dispatch = %d, want %d", got, tornTotal)
	}
}

// TestCombineCatchUpDoesNotPublishAHalfSubscribedTransaction commits Atomic2 after Combine has subscribed to x but before it has subscribed to y, so the transaction settles in the Combine with only x delivered.
func TestCombineCatchUpDoesNotPublishAHalfSubscribedTransaction(t *testing.T) {
	var torn tornRecorder
	x, y := New(100), New(900)
	first := &hookedObservable{store: x}
	first.onSubscribe = func() {
		if err := Atomic2(x, y, func(a, b *int) error { *a += 10; *b -= 10; return nil }); err != nil {
			t.Error(err)
		}
	}
	sum := Combine[int, int, int](first, y, torn.sum("Combine(x, y)"), nil)
	defer func() { _ = sum.Close() }()
	torn.verify(t)
	if got := sum.Get(); got != tornTotal {
		t.Fatalf("Combine.Get = %d, want %d", got, tornTotal)
	}
	if got := x.Get() + y.Get(); got != tornTotal || x.Get() != 110 {
		t.Fatalf("stores = %d, %d", x.Get(), y.Get())
	}
}

// TestCombinePublishesWhileADerivedInputIsBehind holds a write to x before Derive(x) has seen it and then writes y. The Combine cannot use Derive(x)'s old value with the new y, but it must not wait for Derive(x) either: Derive(x) finds its value unchanged and never notifies.
func TestCombinePublishesWhileADerivedInputIsBehind(t *testing.T) {
	// The second copy of 5 is the one Set makes to notify Derive(x).
	gate := newValueGate(2, 5)
	x, y := NewWithClone(0, gate.clone), New(0)
	var mu sync.Mutex
	var projected []Pair[int, int]
	coarse := Derive[int, int](x, func(v int) int { return v / 100 }, func(a, b int) bool { return a == b })
	defer func() { _ = coarse.Close() }()
	pair := Combine[int, int, Pair[int, int]](coarse, y, func(a, b int) Pair[int, int] {
		mu.Lock()
		defer mu.Unlock()
		projected = append(projected, Pair[int, int]{a, b})
		return Pair[int, int]{a, b}
	}, nil)
	defer func() { _ = pair.Close() }()
	done := make(chan error, 1)
	go func() { done <- x.Set(5) }()
	<-gate.held[5]
	if err := y.Set(1); err != nil {
		t.Fatal(err)
	}
	if got := pair.Get(); got != (Pair[int, int]{0, 1}) {
		t.Fatalf("Combine = %+v while Derive(x) is behind, want {0 1}", got)
	}
	close(gate.release[5])
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got := pair.Get(); got != (Pair[int, int]{0, 1}) {
		t.Fatalf("Combine = %+v after both writes settled, want {0 1}", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(projected) != 2 {
		t.Fatalf("projector calls = %v, want the initial pair and {0 1} once", projected)
	}
}

// TestCombineStressNeverPublishesATornPair mixes concurrent Atomic2 and AtomicGroup transactions in both store orders, writes that leave a value unchanged, batches, and Combine construction, over Combines of Stores, derived values of different depths and other Combines. Every projector and subscriber checks the invariant, and once the writers stop every derived value must hold the final state.
func TestCombineStressNeverPublishesATornPair(t *testing.T) {
	const writers, operations, constructions = 8, 300, 150
	var torn tornRecorder
	x, y, z := New(tornTotal), New(0), New(0)
	identity := func(v int) int { return v }
	equal := func(a, b int) bool { return a == b }
	pairOf := func(a, b int) Pair[int, int] { return Pair[int, int]{a, b} }
	dx := Derive[int, int](x, identity, equal)
	dy := Derive[int, int](y, identity, equal)
	ddx := Derive[int, int](dx, identity, nil)
	sums := map[string]*Derived[Pair[int, int], int]{
		"x+y":                 Combine(x, y, torn.sum("x+y"), nil),
		"Derive(x)+y":         Combine[int, int, int](dx, y, torn.sum("Derive(x)+y"), equal),
		"x+Derive(y)":         Combine[int, int, int](x.ReadOnly(), dy, torn.sum("x+Derive(y)"), nil),
		"Derive(Derive(x))+y": Combine[int, int, int](ddx, dy, torn.sum("Derive(Derive(x))+Derive(y)"), nil),
	}
	direct := Combine(x, y, pairOf, nil)
	derived := Combine[int, int, Pair[int, int]](ddx, dy, pairOf, nil)
	// Both inputs are consistent pairs; together they must also come from one committed state.
	both := Combine[Pair[int, int], Pair[int, int], Pair[int, int]](direct, derived, func(p, q Pair[int, int]) Pair[int, int] {
		if p != q {
			torn.check(fmt.Sprintf("Combine(%v, %v) projector", p, q), 0)
		}
		torn.check("Combine of Combines projector", p.First+p.Second)
		return p
	}, nil)
	over := Derive[Pair[int, int], int](both, func(p Pair[int, int]) int { return p.First + p.Second }, nil)
	closers := []interface{ Close() error }{over, both, derived, direct, ddx, dy, dx}
	for name, sum := range sums {
		sum.Subscribe(func(v int) { torn.check(name+" subscriber", v) })
		closers = append(closers, sum)
	}
	over.Subscribe(func(v int) { torn.check("Derive(Combine of Combines) subscriber", v) })
	defer func() {
		for _, c := range closers {
			_ = c.Close()
		}
	}()

	var wg sync.WaitGroup
	for w := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			random := rand.New(rand.NewPCG(uint64(w), 0x5eed))
			for range operations {
				amount := random.IntN(7) - 3
				var err error
				switch random.IntN(6) {
				case 0:
					err = Atomic2(x, y, func(a, b *int) error { *a -= amount; *b += amount; return nil })
				case 1:
					err = Atomic2(y, x, func(b, a *int) error { *a -= amount; *b += amount; return nil })
				case 2:
					err = AtomicGroup(
						AtomicUpdate(y, func(b *int) error { *b += amount; return nil }),
						AtomicUpdate(z, func(c *int) error { *c++; return nil }),
						AtomicUpdate(x, func(a *int) error { *a -= amount; return nil }),
					)
				case 3:
					// A write that leaves the value unchanged gives x a new version that equality-filtered derived values do not notify.
					err = x.Update(func(*int) {})
				case 4:
					err = y.Batch(func() { _ = y.Update(func(*int) {}) })
				case 5:
					err = Atomic2(x, z, func(_ *int, c *int) error { *c++; return nil })
				}
				if err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}
	// Every tenth Combine constructed during the writes stays open, to check that it also reaches the final state.
	var kept []*Derived[Pair[int, int], Pair[int, int]]
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range constructions {
			var sum *Derived[Pair[int, int], int]
			if i%2 == 0 {
				sum = Combine(x, y, torn.sum("constructed x+y"), nil)
			} else {
				sum = Combine[int, int, int](ddx, y, torn.sum("constructed Derive(Derive(x))+y"), nil)
			}
			torn.check("constructed Combine.Get", sum.Get())
			_ = sum.Close()
			if i%10 == 0 {
				kept = append(kept, Combine[int, int, Pair[int, int]](ddx, dy, func(a, b int) Pair[int, int] {
					torn.check("kept Combine projector", a+b)
					return Pair[int, int]{a, b}
				}, nil))
			}
		}
	}()
	wg.Wait()
	for _, c := range kept {
		closers = append(closers, c)
	}
	torn.verify(t)

	final := Pair[int, int]{x.Get(), y.Get()}
	if final.First+final.Second != tornTotal {
		t.Fatalf("stores lost the invariant: %+v", final)
	}
	for name, sum := range sums {
		if got := sum.Get(); got != tornTotal {
			t.Errorf("%s = %d after the writers stopped, want %d", name, got, tornTotal)
		}
	}
	for name, got := range map[string]Pair[int, int]{"Combine(x, y)": direct.Get(), "Combine(Derive(Derive(x)), Derive(y))": derived.Get(), "Combine of Combines": both.Get()} {
		if got != final {
			t.Errorf("%s = %+v after the writers stopped, want the final state %+v", name, got, final)
		}
	}
	for i, c := range kept {
		if got := c.Get(); got != final {
			t.Errorf("Combine constructed during the writes (#%d) = %+v after they stopped, want %+v", i, got, final)
		}
	}
	if got := over.Get(); got != tornTotal {
		t.Errorf("Derive(Combine of Combines) = %d, want %d", got, tornTotal)
	}
}

// TestCombineValuesOnlyMoveForwardWithVersion has concurrent writers count two Stores up independently, so every committed state has each counter at least as high as in any earlier one. Ordered by version, the pairs a Combine publishes must never move a counter back, and once the writers stop the Combine must hold the final state.
func TestCombineValuesOnlyMoveForwardWithVersion(t *testing.T) {
	const writers, operations = 8, 200
	x, y := New(0), New(0)
	pairOf := func(a, b int) Pair[int, int] { return Pair[int, int]{a, b} }
	slowIdentity := func(v int) int {
		for range 50 {
			_ = rand.Int()
		}
		return v
	}
	dx := Derive[int, int](x, slowIdentity, nil)
	defer func() { _ = dx.Close() }()
	combines := map[string]*Derived[Pair[int, int], Pair[int, int]]{
		"Combine(x, y)":         Combine(x, y, pairOf, nil),
		"Combine(Derive(x), y)": Combine[int, int, Pair[int, int]](dx, y, pairOf, nil),
	}
	var mu sync.Mutex
	delivered := make(map[string][]Snapshot[Pair[int, int]])
	for name, c := range combines {
		defer func() { _ = c.Close() }()
		c.SubscribeSnapshot(func(s Snapshot[Pair[int, int]]) {
			mu.Lock()
			delivered[name] = append(delivered[name], s)
			mu.Unlock()
		})
	}
	var wg sync.WaitGroup
	for w := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store := x
			if w%2 == 1 {
				store = y
			}
			for range operations {
				if err := store.Update(func(v *int) { *v++ }); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}
	wg.Wait()

	final := Pair[int, int]{x.Get(), y.Get()}
	mu.Lock()
	defer mu.Unlock()
	for name, c := range combines {
		snapshots := delivered[name]
		slices.SortFunc(snapshots, func(p, q Snapshot[Pair[int, int]]) int { return cmp.Compare(p.Version, q.Version) })
		for i := 1; i < len(snapshots); i++ {
			previous, next := snapshots[i-1], snapshots[i]
			if next.Value.First < previous.Value.First || next.Value.Second < previous.Value.Second {
				t.Errorf("%s moved back: v%d %+v -> v%d %+v", name, previous.Version, previous.Value, next.Version, next.Value)
				break
			}
		}
		if got := c.Get(); got != final {
			t.Errorf("%s = %+v after the writers stopped, want the final state %+v", name, got, final)
		}
	}
}

// bumpingObservable is an application Observable over a Store whose Snapshot returns a new, higher version on every call, which the Observable contract allows.
type bumpingObservable struct {
	store   *Store[int]
	version atomic.Uint64
}

func (o *bumpingObservable) Snapshot() Snapshot[int] {
	return Snapshot[int]{Value: o.store.Get(), Version: o.version.Add(1)}
}

func (o *bumpingObservable) SubscribeSnapshot(fn func(Snapshot[int])) func() {
	return o.store.Subscribe(func(Snapshot[int]) { fn(o.Snapshot()) })
}

// TestCombineOverAnObservableWithChangingVersionsReturns builds Combines, directly and through another Combine, over an application Observable whose version changes on every Snapshot; Combine must not wait for that version to hold still.
func TestCombineOverAnObservableWithChangingVersionsReturns(t *testing.T) {
	x, y := New(1), New(2)
	sum := func(a, b int) int { return a + b }
	built := make(chan [2]*Derived[Pair[int, int], int], 1)
	go func() {
		inner := Combine[int, int, int](&bumpingObservable{store: x}, y, sum, nil)
		outer := Combine[int, int, int](inner, y, sum, nil)
		built <- [2]*Derived[Pair[int, int], int]{inner, outer}
	}()
	var combines [2]*Derived[Pair[int, int], int]
	select {
	case combines = <-built:
	case <-time.After(5 * time.Second):
		t.Fatal("Combine over an Observable whose version changes on every Snapshot did not return")
	}
	inner, outer := combines[0], combines[1]
	defer func() { _ = inner.Close() }()
	defer func() { _ = outer.Close() }()
	if got := inner.Get(); got != 3 {
		t.Fatalf("Combine = %d, want 3", got)
	}
	if got := outer.Get(); got != 5 {
		t.Fatalf("Combine of the Combine = %d, want 5", got)
	}
	_ = x.Set(10)
	_ = y.Set(20)
	if got := inner.Get(); got != 30 {
		t.Fatalf("Combine = %d after both writes, want 30", got)
	}
	if got := outer.Get(); got != 50 {
		t.Fatalf("Combine of the Combine = %d after both writes, want 50", got)
	}
}

// movingObservable is an application Observable over a Store whose Snapshot, while enabled, first runs move, which writes another Store and returns once that write has committed. A Combine that reads the other Store before this one then always finds it moved, as a writer that never pauses can make any read find.
type movingObservable struct {
	store   *Store[int]
	move    func()
	enabled atomic.Bool
}

func (o *movingObservable) Snapshot() Snapshot[int] {
	if o.enabled.Load() {
		o.move()
	}
	return o.store.Snapshot()
}

func (o *movingObservable) SubscribeSnapshot(fn func(Snapshot[int])) func() {
	return o.store.SubscribeSnapshot(fn)
}

// TestCombineConstructionReturnsWhileAWriterKeepsMovingAnInput has a writer goroutine write x between the reads of x and of the second input in every read construction makes itself, so none of them finds a pair; construction used to read until the writer paused. The writer reads nothing while it writes, so a read made while it delivers its own notification finds a pair, also when x reaches Combine through a derived value whose equals filters every write and so notifies nothing.
func TestCombineConstructionReturnsWhileAWriterKeepsMovingAnInput(t *testing.T) {
	pairOf := func(a, b int) Pair[int, int] { return Pair[int, int]{a, b} }
	for _, filtered := range []bool{false, true} {
		t.Run(fmt.Sprintf("filtered=%v", filtered), func(t *testing.T) {
			x, y := New(0), New(7)
			var first Observable[int] = x
			if filtered {
				unchanged := Derive[int, int](x, func(int) int { return 0 }, func(a, b int) bool { return a == b })
				defer func() { _ = unchanged.Close() }()
				first = unchanged
			}
			requests := make(chan chan struct{})
			defer close(requests)
			var writing atomic.Bool
			go func() {
				for done := range requests {
					writing.Store(true)
					if err := x.Update(func(v *int) { *v++ }); err != nil {
						t.Error(err)
					}
					writing.Store(false)
					close(done)
				}
			}()
			second := &movingObservable{store: y}
			second.move = func() {
				if writing.Load() {
					// A read made from the writer's own notification: the writer is not writing now.
					return
				}
				done := make(chan struct{})
				requests <- done
				<-done
			}
			second.enabled.Store(true)
			built := make(chan *Derived[Pair[int, int], Pair[int, int]], 1)
			go func() { built <- Combine[int, int, Pair[int, int]](first, second, pairOf, nil) }()
			var combined *Derived[Pair[int, int], Pair[int, int]]
			select {
			case combined = <-built:
				second.enabled.Store(false)
			case <-time.After(10 * time.Second):
				second.enabled.Store(false)
				_ = (<-built).Close()
				t.Fatal("Combine construction did not return while the writer kept moving x between its reads")
			}
			defer func() { _ = combined.Close() }()
			if got, want := combined.Get(), (Pair[int, int]{first.Snapshot().Value, 7}); got != want {
				t.Fatalf("Combine = %+v after construction, want %+v", got, want)
			}
		})
	}
}

// commitWatch is the clone function of a NewWithClone store that reports the first copy made of each value it was told to expect. Set makes that copy under the store lock when it commits the value, so once it is reported a read of the store waits for the commit.
type commitWatch struct {
	mu       sync.Mutex
	expected map[int]chan struct{}
}

func (w *commitWatch) expect(value int) <-chan struct{} {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.expected == nil {
		w.expected = make(map[int]chan struct{})
	}
	copied := make(chan struct{})
	w.expected[value] = copied
	return copied
}

func (w *commitWatch) clone(value int) int {
	w.mu.Lock()
	defer w.mu.Unlock()
	if copied, ok := w.expected[value]; ok {
		delete(w.expected, value)
		close(copied)
	}
	return value
}

// TestCombinePublishReturnsWhileWritesKeepMovingAnInput writes x, each time from a new goroutine, between a publish's reads of Derive(x) and of y in every read it makes, so none finds a pair; the publish used to read until the writes paused, and the goroutine that wrote y waited with it. Derive(x) filters every write of x as equal, so its subscribers never hear of them, and Combine must still publish the final pair once the writes stop.
func TestCombinePublishReturnsWhileWritesKeepMovingAnInput(t *testing.T) {
	var watch commitWatch
	x, y := NewWithClone(0, watch.clone), New(0)
	unchanged := Derive[int, int](x, func(int) int { return 0 }, func(a, b int) bool { return a == b })
	defer func() { _ = unchanged.Close() }()
	var writes sync.WaitGroup
	var next atomic.Int64
	second := &movingObservable{store: y}
	second.move = func() {
		value := int(next.Add(1))
		committed := watch.expect(value)
		writes.Add(1)
		go func() {
			defer writes.Done()
			if err := x.Set(value); err != nil {
				t.Error(err)
			}
		}()
		<-committed
		// The writer still holds the store lock; this waits until it has committed.
		x.Version()
	}
	combined := Combine[int, int, Pair[int, int]](unchanged, second, func(a, b int) Pair[int, int] { return Pair[int, int]{a, b} }, nil)
	defer func() { _ = combined.Close() }()
	// Derive(x) accepts this write without notifying, so the version the Combine holds for it is out of date and the next publish has to read both inputs.
	if err := x.Set(-1); err != nil {
		t.Fatal(err)
	}
	second.enabled.Store(true)
	published := make(chan error, 1)
	go func() { published <- y.Set(1) }()
	select {
	case err := <-published:
		second.enabled.Store(false)
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		second.enabled.Store(false)
		err := <-published
		writes.Wait()
		t.Fatalf("y.Set did not return while writes kept moving x between Combine's reads (it returned %v once they stopped)", err)
	}
	writes.Wait()
	if got, want := combined.Get(), (Pair[int, int]{0, 1}); got != want {
		t.Fatalf("Combine = %+v after the writes stopped, want %+v", got, want)
	}
}

// TestCombineStaysConsistentWhileAWriterNeverPauses runs a writer that, without pausing, moves amounts between x and y with Atomic2 and rewrites x unchanged, which derived values with an equals filter, while Combines over x, y, those derived values and other Combines are built and published. Construction and every publish must return while the writer still runs, every projector call and delivery must see a committed pair, and once the writer stops every Combine must hold the final state.
func TestCombineStaysConsistentWhileAWriterNeverPauses(t *testing.T) {
	const constructions = 200
	var torn tornRecorder
	x, y := New(tornTotal), New(0)
	identity := func(v int) int { return v }
	equal := func(a, b int) bool { return a == b }
	pairOf := func(a, b int) Pair[int, int] { return Pair[int, int]{a, b} }
	dx := Derive[int, int](x, identity, equal)
	dy := Derive[int, int](y, identity, equal)
	direct := Combine(x, y, pairOf, nil)
	derived := Combine[int, int, Pair[int, int]](dx, dy, pairOf, nil)
	both := Combine[Pair[int, int], Pair[int, int], Pair[int, int]](direct, derived, func(p, q Pair[int, int]) Pair[int, int] {
		if p != q {
			torn.check(fmt.Sprintf("Combine(%v, %v) projector", p, q), 0)
		}
		torn.check("Combine of Combines projector", p.First+p.Second)
		return p
	}, nil)
	closers := []interface{ Close() error }{both, derived, direct, dy, dx}
	defer func() {
		for _, c := range closers {
			_ = c.Close()
		}
	}()
	for name, c := range map[string]*Derived[Pair[int, int], Pair[int, int]]{"Combine(x, y)": direct, "Combine(Derive(x), Derive(y))": derived} {
		c.Subscribe(func(p Pair[int, int]) { torn.check(name+" subscriber", p.First+p.Second) })
	}
	both.Subscribe(func(p Pair[int, int]) { torn.check("Combine of Combines subscriber", p.First+p.Second) })

	// running is closed once the writer has written a while, so the Combines below are built and published during the writes.
	running, stop, stopped := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		defer close(stopped)
		for i := 0; ; i++ {
			if i == 100 {
				close(running)
			}
			select {
			case <-stop:
				return
			default:
			}
			var err error
			if i%3 == 2 {
				err = x.Update(func(*int) {})
			} else {
				amount := i%7 - 3
				err = Atomic2(x, y, func(a, b *int) error { *a -= amount; *b += amount; return nil })
			}
			if err != nil {
				t.Error(err)
				return
			}
		}
	}()
	var kept []*Derived[Pair[int, int], Pair[int, int]]
	built := make(chan struct{})
	go func() {
		defer close(built)
		select {
		case <-running:
		case <-stopped:
			return
		}
		for i := range constructions {
			var sum interface {
				Get() int
				Close() error
			}
			switch i % 3 {
			case 0:
				sum = Combine(x, y, torn.sum("constructed x+y"), nil)
			case 1:
				sum = Combine[int, int, int](dx, y, torn.sum("constructed Derive(x)+y"), nil)
			default:
				sum = Combine[Pair[int, int], int, int](direct, dy, func(p Pair[int, int], b int) int {
					torn.check("constructed Combine(x, y)+Derive(y) projector", p.First+p.Second)
					return p.First + b
				}, nil)
			}
			torn.check("constructed Combine.Get", sum.Get())
			_ = sum.Close()
			// Every twentieth Combine constructed during the writes stays open, to check that it also reaches the final state.
			if i%20 == 0 {
				kept = append(kept, Combine[int, int, Pair[int, int]](dx, y, func(a, b int) Pair[int, int] {
					torn.check("kept Combine projector", a+b)
					return Pair[int, int]{a, b}
				}, nil))
			}
		}
	}()
	select {
	case <-built:
	case <-time.After(30 * time.Second):
		close(stop)
		<-built
		t.Fatal("Combine construction did not return while the writer ran")
	}
	close(stop)
	select {
	case <-stopped:
	case <-time.After(10 * time.Second):
		t.Fatal("the writer did not stop: a publish it triggered did not return")
	}
	for _, c := range kept {
		closers = append(closers, c)
	}
	torn.verify(t)

	final := Pair[int, int]{x.Get(), y.Get()}
	if final.First+final.Second != tornTotal {
		t.Fatalf("stores lost the invariant: %+v", final)
	}
	for name, got := range map[string]Pair[int, int]{"Combine(x, y)": direct.Get(), "Combine(Derive(x), Derive(y))": derived.Get(), "Combine of Combines": both.Get()} {
		if got != final {
			t.Errorf("%s = %+v after the writer stopped, want the final state %+v", name, got, final)
		}
	}
	for i, c := range kept {
		if got := c.Get(); got != final {
			t.Errorf("Combine constructed during the writes (#%d) = %+v after they stopped, want %+v", i, got, final)
		}
	}
}
