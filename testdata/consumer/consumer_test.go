package consumer_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yldm-tech/nodra"
	"github.com/yldm-tech/nodra/devtools"
)

type appState struct {
	Count int
	Name  string
	Items []string
}

type invalidationTarget struct{ count atomic.Int64 }

func (t *invalidationTarget) Invalidate() { t.count.Add(1) }

type metricsSink struct{ called atomic.Int64 }

func (s *metricsSink) RecordActionMetrics(_ string, _ nodra.ActionMetrics) { s.called.Add(1) }

type memoryStorage struct {
	mu   sync.Mutex
	data []byte
}

func (s *memoryStorage) Load(context.Context) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]byte(nil), s.data...), nil
}

func (s *memoryStorage) Save(_ context.Context, data []byte) error {
	s.mu.Lock()
	s.data = append([]byte(nil), data...)
	s.mu.Unlock()
	return nil
}

func waitUntil(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition was not reached")
}

// TestPublicAPI exercises the root package from a separate module. It must not reach internal/core or rely on repository-local replace/workspace settings.
func TestPublicAPI(t *testing.T) {
	ctx := context.Background()
	store := nodra.New(appState{Name: "initial", Items: []string{"a"}})
	defer func() { _ = store.Close() }()

	if got := store.Get(); got.Name != "initial" || store.Version() != 0 {
		t.Fatalf("initial store = %+v at version %d", got, store.Version())
	}
	var snapshots atomic.Int64
	stopSnapshot := store.SubscribeWith(nodra.SubscribeOptions[appState]{Immediate: true}, func(nodra.Snapshot[appState]) {
		snapshots.Add(1)
	})
	defer stopSnapshot()
	if err := store.Update(func(value *appState) { value.Count = 1 }); err != nil {
		t.Fatal(err)
	}
	if store.Snapshot().Version != 1 || snapshots.Load() != 2 {
		t.Fatalf("snapshot/version = %d/%d", snapshots.Load(), store.Snapshot().Version)
	}
	if err := store.UpdateErr(func(value *appState) error {
		value.Count = 2
		return errors.New("reject")
	}); err == nil || store.Get().Count != 1 {
		t.Fatalf("UpdateErr did not roll back: err=%v state=%+v", err, store.Get())
	}
	var batches atomic.Int64
	stopMutation := store.Subscribe(func(nodra.Snapshot[appState]) { batches.Add(1) })
	defer stopMutation()
	if err := store.Batch(func() {
		_ = store.Set(appState{Count: 3})
		_ = store.Patch(func(value *appState) { value.Count = 4 })
	}); err != nil {
		t.Fatal(err)
	}
	if store.Get().Count != 4 || batches.Load() != 1 {
		t.Fatalf("batch notifications=%d state=%+v", batches.Load(), store.Get())
	}
	if err := store.Transaction(func(value *appState) error { value.Count = 5; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := store.Do("rename", func(value *appState) error { value.Name = "done"; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := store.Action("increment", func(value *appState) error { value.Count++; return nil })(); err != nil {
		t.Fatal(err)
	}
	watch := store.Watch(ctx, 2)
	if err := store.Set(appState{Count: 7}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-watch:
	case <-time.After(time.Second):
		t.Fatal("watch did not receive an update")
	}

	readOnly := store.ReadOnly()
	var reader nodra.Reader[appState] = readOnly
	var observable nodra.Observable[appState] = readOnly
	if selected := nodra.Select(reader, func(value appState) int { return value.Count }); selected.Value != 7 {
		t.Fatalf("selected value = %d", selected.Value)
	}
	doubled := nodra.Derive(observable, func(value appState) int { return value.Count * 2 }, func(a, b int) bool { return a == b })
	defer func() { _ = doubled.Close() }()
	combined := nodra.Combine(observable, doubled, func(value appState, doubled int) int { return value.Count + doubled }, func(a, b int) bool { return a == b })
	defer func() { _ = combined.Close() }()
	if doubled.Get() != 14 || combined.Get() != 21 {
		t.Fatalf("derived values = %d/%d", doubled.Get(), combined.Get())
	}
	projection := nodra.DeriveProjection(reader, func(value appState) []string {
		return append([]string(nil), value.Items...)
	}, func(a, b []string) bool { return len(a) == len(b) })
	defer func() { _ = projection.Close() }()
	stopSelector := nodra.SubscribeSelectorWith(observable, func(value appState) int { return value.Count }, func(int, int) {}, nodra.SelectorOptions[int]{Immediate: true})
	defer stopSelector()
	stopProjection := nodra.SubscribeProjectionWith(reader, func(value appState) []string {
		return append([]string(nil), value.Items...)
	}, func([]string, []string) {}, nodra.SelectorOptions[[]string]{})
	defer stopProjection()
	field, err := nodra.BindField(store, func(value appState) string { return value.Name }, func(value *appState, name string) error {
		value.Name = name
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := field.Set("field"); err != nil || field.Get() != "field" {
		t.Fatalf("field = %q, err=%v", field.Get(), err)
	}

	other := nodra.New(10)
	defer func() { _ = other.Close() }()
	if err := nodra.Atomic2(store, other, func(first *appState, second *int) error {
		first.Count = 8
		*second = 11
		return nil
	}); err != nil || store.Get().Count != 8 || other.Get() != 11 {
		t.Fatalf("Atomic2: err=%v state=%+v/%d", err, store.Get(), other.Get())
	}
	if err := nodra.AtomicGroup(
		nodra.AtomicUpdate(store, func(value *appState) error { value.Count = 9; return nil }),
		nodra.AtomicUpdate(other, func(value *int) error { *value = 12; return nil }),
	); err != nil || store.Get().Count != 9 || other.Get() != 12 {
		t.Fatalf("AtomicGroup: err=%v state=%+v/%d", err, store.Get(), other.Get())
	}

	history := nodra.TrackWith(store, nodra.HistoryOptions[appState]{
		Limit:    4,
		MaxBytes: 1 << 20,
		Size:     func(value appState) int64 { return int64(len(value.Name) + len(value.Items)) },
	})
	defer func() { _ = history.Close() }()
	if err := store.Set(appState{Count: 10, Name: "history"}); err != nil {
		t.Fatal(err)
	}
	if err := history.Undo(); err != nil {
		t.Fatal(err)
	}
	if err := history.Redo(); err != nil {
		t.Fatal(err)
	}
	end, err := history.BeginGroup()
	if err != nil {
		t.Fatal(err)
	}
	_ = store.Set(appState{Count: 11})
	_ = store.Set(appState{Count: 12})
	end()
	if !history.CanUndo() || history.Status().Get().Closed {
		t.Fatal("history status")
	}

	path := filepath.Join(t.TempDir(), "state.json")
	persisted := nodra.New(1)
	defer func() { _ = persisted.Close() }()
	file := nodra.FileStorage{Path: path, Backup: true, Journal: true, Durable: true}
	persistence := nodra.NewPersistence(persisted, file, nodra.VersionedJSONCodec[int]{Version: 1})
	if err := persistence.Save(ctx); err != nil {
		t.Fatal(err)
	}
	if err := persisted.Set(2); err != nil {
		t.Fatal(err)
	}
	if err := persistence.Load(ctx); err != nil || persisted.Get() != 1 {
		t.Fatalf("persistence load: err=%v value=%d", err, persisted.Get())
	}
	codec := nodra.JSONCodec[appState]{}
	raw, err := codec.Encode(store.Get())
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := codec.Decode(raw)
	if err != nil || decoded.Count != store.Get().Count {
		t.Fatalf("json codec: err=%v value=%+v", err, decoded)
	}
	keys, err := nodra.NewMemoryKeyProvider("key-1", []byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatal(err)
	}
	encrypted := nodra.AESGCMCodec[appState]{Inner: codec, Keys: keys}
	ciphertext, err := encrypted.Encode(store.Get())
	if err != nil {
		t.Fatal(err)
	}
	if decoded, err := encrypted.Decode(ciphertext); err != nil || decoded.Count != store.Get().Count {
		t.Fatalf("encrypted codec: err=%v value=%+v", err, decoded)
	}
	memory := &memoryStorage{}
	memoryPersistence := nodra.NewPersistence(persisted, memory, nodra.JSONCodec[int]{})
	if err := memoryPersistence.Save(ctx); err != nil {
		t.Fatal(err)
	}

	target := &invalidationTarget{}
	stopInvalidation := nodra.InvalidateTarget(observable, target)
	defer stopInvalidation()
	if err := store.Set(appState{Count: 13}); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, time.Second, func() bool { return target.count.Load() > 0 })

	asyncStore := nodra.New(0)
	defer func() { _ = asyncStore.Close() }()
	action, err := nodra.NewAsyncAction(asyncStore, "double", func(_ context.Context, value int, progress func(float64)) (int, error) {
		progress(0.5)
		return value * 2, nil
	}, func(value *int, result int) error {
		*value = result
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	task, err := action.Run(ctx, 4)
	if err != nil {
		t.Fatal(err)
	}
	if err := task.Wait(ctx); err != nil || asyncStore.Get() != 8 || task.Status().Get().Phase != nodra.TaskSucceeded {
		t.Fatalf("async action: err=%v value=%d status=%+v", err, asyncStore.Get(), task.Status().Get())
	}
	sink := &metricsSink{}
	if err := action.ExportMetrics(sink); err != nil || sink.called.Load() != 1 {
		t.Fatalf("metrics export: err=%v calls=%d", err, sink.called.Load())
	}
	if err := action.Close(ctx); err != nil {
		t.Fatal(err)
	}

	scope := nodra.NewScope()
	var stopped atomic.Bool
	if err := scope.Observe(func() { stopped.Store(true) }); err != nil {
		t.Fatal(err)
	}
	if err := scope.Defer(func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := scope.Close(ctx); err != nil || !stopped.Load() {
		t.Fatalf("scope close: err=%v stopped=%t", err, stopped.Load())
	}

	inspector := devtools.New(devtools.Options{AllowRestore: true})
	defer func() { _ = inspector.Close() }()
	detach, err := devtools.Attach(inspector, "state", store)
	if err != nil {
		t.Fatal(err)
	}
	defer detach()
	if len(inspector.Snapshot().Stores) != 1 {
		t.Fatal("DevTools did not attach the store")
	}
	if err := inspector.Apply("state", json.RawMessage(`{"Count":14,"Name":"restored","Items":["a"]}`)); err != nil || store.Get().Count != 14 {
		t.Fatalf("DevTools apply: err=%v state=%+v", err, store.Get())
	}
	_ = inspector.Search("restored", 10)
	_ = inspector.Changes(0, 0, 10)
	_ = inspector.EventsBefore(0, 10)
	if _, err := inspector.RotateToken(); err != nil {
		t.Fatal(err)
	}
	inspector.Clear()
}
