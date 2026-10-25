package nodra_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/yldm-tech/nodra"
)

func TestRootStoreObservationAndDerivedValues(t *testing.T) {
	store := nodra.New(1)
	defer func() { _ = store.Close() }()
	if got := store.Get(); got != 1 || store.Version() != 0 {
		t.Fatalf("initial store = %d at version %d", got, store.Version())
	}
	var seen []nodra.Snapshot[int]
	stop := store.SubscribeWith(nodra.SubscribeOptions[int]{Immediate: true}, func(snapshot nodra.Snapshot[int]) {
		seen = append(seen, snapshot)
	})
	defer stop()
	if err := store.Update(func(value *int) { *value = 2 }); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 2 || seen[1].Value != 2 || seen[1].Version != 1 {
		t.Fatalf("snapshots = %+v", seen)
	}

	readOnly := store.ReadOnly()
	var reader nodra.Reader[int] = readOnly
	var observable nodra.Observable[int] = reader
	doubled := nodra.Derive(observable, func(value int) int { return value * 2 }, func(a, b int) bool { return a == b })
	defer func() { _ = doubled.Close() }()
	combined := nodra.Combine(observable, doubled, func(a, b int) int { return a + b }, func(a, b int) bool { return a == b })
	defer func() { _ = combined.Close() }()
	if doubled.Get() != 4 || combined.Get() != 6 {
		t.Fatalf("derived values = %d, %d", doubled.Get(), combined.Get())
	}
	if got := nodra.Select(reader, func(value int) string { return string(rune('0' + value)) }); got.Value != "2" {
		t.Fatalf("selected value = %q", got.Value)
	}
}

func TestRootMutationHistoryFieldAndAtomicContracts(t *testing.T) {
	first, second := nodra.New(1), nodra.New(10)
	defer func() { _ = first.Close() }()
	defer func() { _ = second.Close() }()
	field, err := nodra.BindField(first, func(value int) int { return value }, func(state *int, value int) error {
		if value < 0 {
			return errors.New("negative")
		}
		*state = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := field.Set(3); err != nil || first.Get() != 3 {
		t.Fatalf("field set: %v, value %d", err, first.Get())
	}
	if err := field.Set(-1); err == nil || first.Get() != 3 {
		t.Fatalf("invalid field write: err=%v value=%d", err, first.Get())
	}
	history := nodra.Track(first, 4)
	defer func() { _ = history.Close() }()
	if err := first.Set(4); err != nil {
		t.Fatal(err)
	}
	if err := history.Undo(); err != nil || first.Get() != 3 {
		t.Fatalf("undo: %v value=%d", err, first.Get())
	}
	if err := history.Redo(); err != nil || first.Get() != 4 {
		t.Fatalf("redo: %v value=%d", err, first.Get())
	}
	if err := nodra.Atomic2(first, second, func(a *int, b *int) error { *a += 1; *b += 2; return nil }); err != nil {
		t.Fatal(err)
	}
	if first.Get() != 5 || second.Get() != 12 {
		t.Fatalf("atomic values = %d, %d", first.Get(), second.Get())
	}
}

func TestRootPersistenceAndAsyncActionContracts(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.json")
	store := nodra.New(7)
	defer func() { _ = store.Close() }()
	persistence := nodra.NewPersistence(store, nodra.FileStorage{Path: path, Journal: true, Backup: true}, nodra.JSONCodec[int]{})
	if err := persistence.Save(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.Set(9); err != nil {
		t.Fatal(err)
	}
	if err := persistence.Load(ctx); err != nil || store.Get() != 7 {
		t.Fatalf("load: %v value=%d", err, store.Get())
	}

	action, err := nodra.NewAsyncAction(store, "double", func(_ context.Context, value int, progress func(float64)) (int, error) {
		progress(1)
		return value * 2, nil
	}, func(state *int, value int) error { *state = value; return nil })
	if err != nil {
		t.Fatal(err)
	}
	task, err := action.Run(ctx, 5)
	if err != nil {
		t.Fatal(err)
	}
	if err := task.Wait(ctx); err != nil || store.Get() != 10 {
		t.Fatalf("action: %v value=%d", err, store.Get())
	}
	if task.Status().Get().Phase != nodra.TaskSucceeded {
		t.Fatalf("task phase = %v", task.Status().Get().Phase)
	}
	if err := action.Close(ctx); err != nil {
		t.Fatal(err)
	}
}
