package bridge_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/yldm-tech/nodra"
	"github.com/yldm-tech/nodra/bridge"
)

type state struct {
	Count  int    `json:"count"`
	Secret string `json:"secret"`
	Note   string `json:"note"`
}

type pageState struct {
	Count int `json:"count"`
}

func project(s state) pageState { return pageState{Count: s.Count} }

// frames collects what a watch sends, and lets a test wait for a frame.
type frames struct {
	mu   sync.Mutex
	got  []bridge.Frame
	more chan struct{}
}

func newFrames() *frames { return &frames{more: make(chan struct{}, 64)} }

func (f *frames) send(frame bridge.Frame) error {
	f.mu.Lock()
	f.got = append(f.got, frame)
	f.mu.Unlock()
	f.more <- struct{}{}
	return nil
}

func (f *frames) all() []bridge.Frame {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]bridge.Frame(nil), f.got...)
}

// wait waits until n frames arrived.
func (f *frames) wait(t *testing.T, n int) []bridge.Frame {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for len(f.all()) < n {
		select {
		case <-f.more:
		case <-deadline:
			t.Fatalf("waited for %d frames, got %+v", n, f.all())
		}
	}
	return f.all()
}

// watch runs Watch on its own goroutine and returns its result channel.
func watch(ctx context.Context, b *bridge.Bridge, name string, after uint64, f *frames) <-chan error {
	done := make(chan error, 1)
	go func() { done <- b.Watch(ctx, name, after, f.send) }()
	return done
}

func result(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("the watch did not end")
		return nil
	}
}

func decode[V any](t *testing.T, frame bridge.Frame) V {
	t.Helper()
	var v V
	if err := json.Unmarshal(frame.Value, &v); err != nil {
		t.Fatalf("frame %s: %v", frame.Value, err)
	}
	return v
}

func TestExposeSendsTheValueAsJSON(t *testing.T) {
	store := nodra.New(state{Count: 1, Note: "n"})
	defer func() { _ = store.Close() }()
	b := bridge.New(bridge.Options{})
	defer b.Close()
	if _, err := bridge.Expose(b, "all", store); err != nil {
		t.Fatal(err)
	}
	frame, err := b.Get(context.Background(), "all")
	if err != nil {
		t.Fatal(err)
	}
	if got := decode[state](t, frame); frame.Name != "all" || frame.Version != 0 || got != (state{Count: 1, Note: "n"}) {
		t.Fatalf("frame = %+v, value %+v", frame, got)
	}
}

func TestAProjectionKeepsTheRestInGo(t *testing.T) {
	store := nodra.New(state{Count: 1, Secret: "token"})
	defer func() { _ = store.Close() }()
	b := bridge.New(bridge.Options{})
	defer b.Close()
	if _, err := bridge.ExposeProjection(b, "counter", store, project); err != nil {
		t.Fatal(err)
	}
	frame, err := b.Get(context.Background(), "counter")
	if err != nil {
		t.Fatal(err)
	}
	if string(frame.Value) != `{"count":1}` {
		t.Fatalf("value = %s", frame.Value)
	}
}

func TestWatchSendsOnlyChangesOfTheProjection(t *testing.T) {
	store := nodra.New(state{Count: 1})
	defer func() { _ = store.Close() }()
	b := bridge.New(bridge.Options{})
	defer b.Close()
	if _, err := bridge.ExposeProjection(b, "counter", store, project); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	f := newFrames()
	done := watch(ctx, b, "counter", 0, f)
	if got := f.wait(t, 1); decode[pageState](t, got[0]).Count != 1 {
		t.Fatalf("first frame = %+v", got[0])
	}
	// A field the projection leaves out changes: the page has nothing new to see.
	if err := store.Update(func(s *state) { s.Note = "unseen" }); err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *state) { s.Count = 2 }); err != nil {
		t.Fatal(err)
	}
	got := f.wait(t, 2)
	if len(got) != 2 || decode[pageState](t, got[1]).Count != 2 || got[1].Version != 2 {
		t.Fatalf("frames = %+v", got)
	}
	cancel()
	if err := result(t, done); err != nil {
		t.Fatalf("a canceled watch returned %v", err)
	}
}

func TestWatchSkipsTheVersionThePageHolds(t *testing.T) {
	store := nodra.New(state{Count: 1})
	defer func() { _ = store.Close() }()
	if err := store.Update(func(s *state) { s.Count = 2 }); err != nil {
		t.Fatal(err)
	}
	b := bridge.New(bridge.Options{})
	defer b.Close()
	if _, err := bridge.ExposeProjection(b, "counter", store, project); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := newFrames()
	done := watch(ctx, b, "counter", store.Version(), f)
	if err := store.Update(func(s *state) { s.Count = 3 }); err != nil {
		t.Fatal(err)
	}
	got := f.wait(t, 1)
	if decode[pageState](t, got[0]).Count != 3 {
		t.Fatalf("frames = %+v; the held version should not be sent again", got)
	}
	cancel()
	_ = result(t, done)
}

// The send here blocks while the page is busy. A page's MyGo channel blocks only once about 1 MiB of frames waits for the page, so this models a page past that window.
func TestTheLatestFrameWinsWhileThePageIsSlow(t *testing.T) {
	store := nodra.New(state{})
	defer func() { _ = store.Close() }()
	b := bridge.New(bridge.Options{})
	defer b.Close()
	if _, err := bridge.ExposeProjection(b, "counter", store, project); err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	var mu sync.Mutex
	var counts []int
	send := func(frame bridge.Frame) error {
		var v pageState
		_ = json.Unmarshal(frame.Value, &v)
		mu.Lock()
		counts = append(counts, v.Count)
		n := len(counts)
		mu.Unlock()
		if n == 1 {
			<-release // the page takes its first frame slowly
		}
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- b.Watch(ctx, "counter", 0, send) }()
	for {
		mu.Lock()
		n := len(counts)
		mu.Unlock()
		if n == 1 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	for range 100 {
		if err := store.Update(func(s *state) { s.Count++ }); err != nil {
			t.Fatal(err)
		}
	}
	close(release)
	deadline := time.Now().Add(5 * time.Second)
	for {
		mu.Lock()
		last := counts[len(counts)-1]
		n := len(counts)
		mu.Unlock()
		if last == 100 {
			if n > 3 {
				t.Fatalf("sent %d frames for 100 changes made while the page was busy; they should coalesce", n)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the last frame is %d, want 100", last)
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	_ = result(t, done)
}

func TestAWatchEndsWhenItsStateGoesAway(t *testing.T) {
	for _, tc := range []struct {
		name string
		end  func(stop func(), store *nodra.Store[state], b *bridge.Bridge)
	}{
		{"exposure stopped", func(stop func(), _ *nodra.Store[state], _ *bridge.Bridge) { stop() }},
		{"store closed", func(_ func(), store *nodra.Store[state], _ *bridge.Bridge) { _ = store.Close() }},
		{"bridge closed", func(_ func(), _ *nodra.Store[state], b *bridge.Bridge) { b.Close() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := nodra.New(state{})
			defer func() { _ = store.Close() }()
			b := bridge.New(bridge.Options{})
			defer b.Close()
			stop, err := bridge.ExposeProjection(b, "counter", store, project)
			if err != nil {
				t.Fatal(err)
			}
			f := newFrames()
			done := watch(context.Background(), b, "counter", 0, f)
			f.wait(t, 1)
			tc.end(stop, store, b)
			if err := result(t, done); !errors.Is(err, bridge.ErrClosed) {
				t.Fatalf("watch returned %v, want ErrClosed", err)
			}
		})
	}
}

func TestAPageThatClosesItsChannelEndsTheWatchQuietly(t *testing.T) {
	store := nodra.New(state{})
	defer func() { _ = store.Close() }()
	b := bridge.New(bridge.Options{})
	defer b.Close()
	if _, err := bridge.Expose(b, "all", store); err != nil {
		t.Fatal(err)
	}
	err := b.Watch(context.Background(), "all", 0, func(bridge.Frame) error { return mygo.ErrChannelClosed })
	if err != nil {
		t.Fatalf("watch returned %v", err)
	}
}

func TestAValueThatDoesNotEncodeFailsTheCall(t *testing.T) {
	store := nodra.New(state{})
	defer func() { _ = store.Close() }()
	b := bridge.New(bridge.Options{})
	defer b.Close()
	if _, err := bridge.ExposeProjection(b, "bad", store, func(state) func() { return func() {} }); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Get(context.Background(), "bad"); err == nil {
		t.Fatal("Get encoded a function")
	}
	if err := b.Watch(context.Background(), "bad", 0, func(bridge.Frame) error { return nil }); err == nil {
		t.Fatal("Watch encoded a function")
	}
}

func TestDispatchDecodesTheArgument(t *testing.T) {
	store := nodra.New(state{})
	defer func() { _ = store.Close() }()
	b := bridge.New(bridge.Options{})
	defer b.Close()
	type step struct {
		By int `json:"by"`
	}
	if _, err := bridge.Handle(b, "add", func(_ context.Context, arg step) error {
		return store.Update(func(s *state) { s.Count += arg.By })
	}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := b.Dispatch(ctx, "add", json.RawMessage(`{"by":3}`)); err != nil {
		t.Fatal(err)
	}
	// A missing or null argument is the zero value.
	for _, raw := range []json.RawMessage{nil, json.RawMessage(`null`), json.RawMessage(" ")} {
		if err := b.Dispatch(ctx, "add", raw); err != nil {
			t.Fatalf("argument %q: %v", raw, err)
		}
	}
	if got := store.Get().Count; got != 3 {
		t.Fatalf("count = %d", got)
	}
	if err := b.Dispatch(ctx, "add", json.RawMessage(`"three"`)); !errors.Is(err, bridge.ErrInvalid) {
		t.Fatalf("a wrong argument returned %v", err)
	}
	if err := b.Dispatch(ctx, "missing", nil); !errors.Is(err, bridge.ErrNotFound) {
		t.Fatalf("an unknown action returned %v", err)
	}
}

func TestTheHandlersErrorReachesThePage(t *testing.T) {
	b := bridge.New(bridge.Options{})
	defer b.Close()
	refused := errors.New("refused")
	if _, err := bridge.Handle(b, "save", func(context.Context, struct{}) error { return refused }); err != nil {
		t.Fatal(err)
	}
	if err := b.Dispatch(context.Background(), "save", nil); !errors.Is(err, refused) {
		t.Fatalf("dispatch returned %v", err)
	}
}

func TestNamesAreTakenOnceAndFreedByTheirOwnStop(t *testing.T) {
	store := nodra.New(state{})
	defer func() { _ = store.Close() }()
	b := bridge.New(bridge.Options{})
	defer b.Close()
	noop := func(context.Context, int) error { return nil }

	stopState, err := bridge.Expose(b, "s", store)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bridge.Expose(b, "s", store); !errors.Is(err, bridge.ErrExists) {
		t.Fatalf("exposing a name twice returned %v", err)
	}
	stopAction, err := bridge.Handle(b, "a", noop)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bridge.Handle(b, "a", noop); !errors.Is(err, bridge.ErrExists) {
		t.Fatalf("registering a name twice returned %v", err)
	}
	// States and actions are apart: the same name may be both.
	if _, err := bridge.Handle(b, "s", noop); err != nil {
		t.Fatal(err)
	}

	stopState()
	stopAction()
	if _, err := b.Get(context.Background(), "s"); !errors.Is(err, bridge.ErrNotFound) {
		t.Fatalf("a stopped state returned %v", err)
	}
	// A new registration under the name is not undone by the old stop, called again.
	if _, err := bridge.Handle(b, "a", noop); err != nil {
		t.Fatal(err)
	}
	stopAction()
	if err := b.Dispatch(context.Background(), "a", nil); err != nil {
		t.Fatalf("the old stop removed the new action: %v", err)
	}
}

func TestInvalidArgumentsAndAClosedBridge(t *testing.T) {
	store := nodra.New(state{})
	defer func() { _ = store.Close() }()
	b := bridge.New(bridge.Options{})
	var nilStore *nodra.Store[state]
	noop := func(context.Context, int) error { return nil }
	for name, err := range map[string]error{
		"empty name":       second(bridge.Expose(b, "", store)),
		"padded name":      second(bridge.Expose(b, " s", store)),
		"nil source":       second(bridge.Expose[state](b, "s", nil)),
		"typed nil source": second(bridge.Expose[state](b, "s", nilStore)),
		"nil projection":   second(bridge.ExposeProjection[state, int](b, "s", store, nil)),
		"nil bridge":       second(bridge.Expose(nil, "s", store)),
		"nil handler":      second(bridge.Handle[int](b, "a", nil)),
	} {
		if !errors.Is(err, bridge.ErrInvalid) {
			t.Errorf("%s: %v, want ErrInvalid", name, err)
		}
	}
	b.Close()
	b.Close()
	if _, err := bridge.Expose(b, "s", store); !errors.Is(err, bridge.ErrClosed) {
		t.Fatalf("exposing on a closed bridge returned %v", err)
	}
	if _, err := bridge.Handle(b, "a", noop); !errors.Is(err, bridge.ErrClosed) {
		t.Fatalf("registering on a closed bridge returned %v", err)
	}
	if err := b.Dispatch(context.Background(), "a", nil); !errors.Is(err, bridge.ErrClosed) {
		t.Fatalf("dispatching on a closed bridge returned %v", err)
	}
}

func second[A any](_ A, err error) error { return err }

func TestOptionsDecidePerPage(t *testing.T) {
	store := nodra.New(state{})
	defer func() { _ = store.Close() }()
	type key struct{}
	trusted := func(ctx context.Context) bool { return ctx.Value(key{}) == "trusted" }
	b := bridge.New(bridge.Options{
		AllowRead:     func(ctx context.Context, name string) bool { return name == "public" || trusted(ctx) },
		AllowDispatch: func(ctx context.Context, _ string) bool { return trusted(ctx) },
	})
	defer b.Close()
	for _, name := range []string{"public", "private"} {
		if _, err := bridge.Expose(b, name, store); err != nil {
			t.Fatal(err)
		}
	}
	ran := false
	if _, err := bridge.Handle(b, "act", func(context.Context, int) error { ran = true; return nil }); err != nil {
		t.Fatal(err)
	}
	page := context.Background()
	if _, err := b.Get(page, "public"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Get(page, "private"); !errors.Is(err, bridge.ErrDenied) {
		t.Fatalf("reading a private state returned %v", err)
	}
	if err := b.Watch(page, "private", 0, func(bridge.Frame) error { return nil }); !errors.Is(err, bridge.ErrDenied) {
		t.Fatalf("watching a private state returned %v", err)
	}
	if err := b.Dispatch(page, "act", nil); !errors.Is(err, bridge.ErrDenied) || ran {
		t.Fatalf("dispatching from an untrusted page returned %v, ran %v", err, ran)
	}
	if err := b.Dispatch(context.WithValue(page, key{}, "trusted"), "act", nil); err != nil || !ran {
		t.Fatalf("dispatching from a trusted page returned %v, ran %v", err, ran)
	}
}

func TestWatchersSeeTheFinalStateUnderConcurrentWrites(t *testing.T) {
	store := nodra.New(state{})
	defer func() { _ = store.Close() }()
	b := bridge.New(bridge.Options{})
	defer b.Close()
	if _, err := bridge.ExposeProjection(b, "counter", store, project); err != nil {
		t.Fatal(err)
	}
	if _, err := bridge.Handle(b, "inc", func(context.Context, struct{}) error {
		return store.Update(func(s *state) { s.Count++ })
	}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var latest [4]struct {
		sync.Mutex
		count int
	}
	var watches sync.WaitGroup
	for i := range latest {
		watches.Go(func() {
			_ = b.Watch(ctx, "counter", 0, func(frame bridge.Frame) error {
				var v pageState
				_ = json.Unmarshal(frame.Value, &v)
				latest[i].Lock()
				latest[i].count = v.Count
				latest[i].Unlock()
				return nil
			})
		})
	}
	var writers sync.WaitGroup
	for range 8 {
		writers.Go(func() {
			for range 50 {
				if err := b.Dispatch(ctx, "inc", nil); err != nil {
					t.Error(err)
				}
			}
		})
	}
	writers.Wait()
	deadline := time.Now().Add(5 * time.Second)
	for i := range latest {
		for {
			latest[i].Lock()
			got := latest[i].count
			latest[i].Unlock()
			if got == 400 {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("watch %d ended at %d, want 400", i, got)
			}
			time.Sleep(time.Millisecond)
		}
	}
	cancel()
	watches.Wait()
}

// Versions count per source, so a version the page holds from an exposure says nothing about another source exposed later under the same name.
func TestAWatchOfANameExposedAgainSendsTheFirstFrame(t *testing.T) {
	b := bridge.New(bridge.Options{})
	defer b.Close()
	docA := nodra.New(state{})
	defer func() { _ = docA.Close() }()
	if err := docA.Update(func(s *state) { s.Count = 111 }); err != nil {
		t.Fatal(err)
	}
	stopA, err := bridge.ExposeProjection(b, "doc", docA, project)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := newFrames()
	done := watch(ctx, b, "doc", 0, f)
	held := f.wait(t, 1)[0]
	stopA()
	if err := result(t, done); !errors.Is(err, bridge.ErrClosed) {
		t.Fatalf("the watch of a stopped exposure returned %v", err)
	}

	docB := nodra.New(state{})
	defer func() { _ = docB.Close() }()
	if err := docB.Update(func(s *state) { s.Count = 222 }); err != nil {
		t.Fatal(err)
	}
	if docB.Version() != held.Version {
		t.Fatalf("the test needs both sources at one version: A at %d, B at %d", held.Version, docB.Version())
	}
	if _, err := bridge.ExposeProjection(b, "doc", docB, project); err != nil {
		t.Fatal(err)
	}
	// The page resumes as nodra.js does, from the version it holds.
	g := newFrames()
	done = watch(ctx, b, "doc", held.Version, g)
	if got := g.wait(t, 1)[0]; decode[pageState](t, got).Count != 222 || got.Version != docB.Version() {
		t.Fatalf("first frame of the name exposed again = %+v", got)
	}
	cancel()
	_ = result(t, done)
}

// A page that a policy refuses gets ErrDenied whether or not the name is exposed or registered, so it cannot tell which names exist.
func TestARefusedPageCannotTellWhichNamesExist(t *testing.T) {
	store := nodra.New(state{})
	defer func() { _ = store.Close() }()
	var mu sync.Mutex
	var asked []string
	deny := func(_ context.Context, name string) bool {
		mu.Lock()
		asked = append(asked, name)
		mu.Unlock()
		return false
	}
	b := bridge.New(bridge.Options{AllowRead: deny, AllowDispatch: deny})
	defer b.Close()
	if _, err := bridge.Expose(b, "admin/secrets", store); err != nil {
		t.Fatal(err)
	}
	ran := false
	if _, err := bridge.Handle(b, "admin/wipe", func(context.Context, int) error { ran = true; return nil }); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, name := range []string{"admin/secrets", "admin/other"} {
		if _, err := b.Get(ctx, name); !errors.Is(err, bridge.ErrDenied) || errors.Is(err, bridge.ErrNotFound) {
			t.Errorf("Get(%q) from a refused page returned %v, want ErrDenied", name, err)
		}
		if err := b.Watch(ctx, name, 0, func(bridge.Frame) error { return nil }); !errors.Is(err, bridge.ErrDenied) || errors.Is(err, bridge.ErrNotFound) {
			t.Errorf("Watch(%q) from a refused page returned %v, want ErrDenied", name, err)
		}
	}
	for _, name := range []string{"admin/wipe", "admin/nope"} {
		if err := b.Dispatch(ctx, name, nil); !errors.Is(err, bridge.ErrDenied) || errors.Is(err, bridge.ErrNotFound) {
			t.Errorf("Dispatch(%q) from a refused page returned %v, want ErrDenied", name, err)
		}
	}
	if ran {
		t.Error("a refused dispatch ran the action")
	}
	askedSoFar := func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), asked...)
	}
	if got := askedSoFar(); len(got) != 6 {
		t.Errorf("the policies were asked about %q, want every call", got)
	}
	// A closed bridge still answers ErrClosed without asking.
	b.Close()
	if _, err := b.Get(ctx, "admin/other"); !errors.Is(err, bridge.ErrClosed) {
		t.Errorf("Get on a closed bridge returned %v", err)
	}
	if got := askedSoFar(); len(got) != 6 {
		t.Errorf("a closed bridge asked the policy: %q", got)
	}
}
