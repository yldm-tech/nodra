package core

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

type asyncStorage struct {
	mu       sync.Mutex
	values   []int
	calls    int
	failures int
	err      error
	gate     <-chan struct{}
	entered  chan int
}

func (s *asyncStorage) Load(context.Context) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.values) == 0 {
		return nil, errors.New("empty storage")
	}
	return json.Marshal(s.values[len(s.values)-1])
}
func (s *asyncStorage) Save(ctx context.Context, data []byte) error {
	var value int
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if s.entered != nil {
		s.entered <- value
	}
	if s.gate != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.gate:
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.failures > 0 {
		s.failures--
		return s.err
	}
	s.values = append(s.values, value)
	return nil
}
func (s *asyncStorage) result() ([]int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]int(nil), s.values...), s.calls
}

func TestAsyncDebounceAndFlushCoalesceWrites(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store, storage := New(0), &asyncStorage{}
		writer, err := NewPersistence(store, storage, JSONCodec[int]{}).AutoSaveAsync(context.Background(), AsyncOptions{Debounce: time.Hour})
		requireNoError(t, err)
		for i := 1; i <= 100; i++ {
			requireNoError(t, store.Set(i))
		}
		synctest.Wait()
		if _, calls := storage.result(); calls != 0 {
			t.Fatal("debounce wrote before its deadline")
		}
		requireNoError(t, writer.Flush(context.Background()))
		if values, _ := storage.result(); !reflect.DeepEqual(values, []int{100}) {
			t.Fatalf("flush writes = %v", values)
		}
		requireNoError(t, store.Set(101))
		synctest.Wait()
		time.Sleep(30 * time.Minute)
		requireNoError(t, store.Set(102))
		synctest.Wait()
		time.Sleep(30 * time.Minute)
		if _, calls := storage.result(); calls != 1 {
			t.Fatal("debounce was not reset by the latest update")
		}
		time.Sleep(30 * time.Minute)
		synctest.Wait()
		requireNoError(t, writer.Close(context.Background()))
		if values, _ := storage.result(); !reflect.DeepEqual(values, []int{100, 102}) {
			t.Fatalf("writes = %v", values)
		}
	})
}

func TestAsyncWritersStayResponsiveAndFlushWaitsForLatest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		gate := make(chan struct{})
		storage := &asyncStorage{gate: gate, entered: make(chan int, 10)}
		store := New(0)
		writer, err := NewPersistence(store, storage, JSONCodec[int]{}).AutoSaveAsync(context.Background(), AsyncOptions{})
		requireNoError(t, err)
		if got := <-storage.entered; got != 0 {
			t.Fatalf("initial write = %d", got)
		}
		// These synchronous updates must finish while storage remains blocked.
		for i := 1; i <= 100; i++ {
			requireNoError(t, store.Set(i))
		}
		flushed := make(chan error, 1)
		go func() { flushed <- writer.Flush(context.Background()) }()
		synctest.Wait()
		select {
		case err := <-flushed:
			t.Fatalf("flush returned before storage completed: %v", err)
		default:
		}
		close(gate)
		requireNoError(t, <-flushed)
		requireNoError(t, writer.Close(context.Background()))
		if values, _ := storage.result(); !reflect.DeepEqual(values, []int{0, 100}) {
			t.Fatalf("writes during blocked storage = %v", values)
		}
	})
}

func TestAsyncFailureIsObservableAndFlushRetries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		want := errors.New("disk full")
		storage := &asyncStorage{failures: 2, err: want}
		writer, err := NewPersistence(New(7), storage, JSONCodec[int]{}).AutoSaveAsync(context.Background(), AsyncOptions{})
		requireNoError(t, err)
		synctest.Wait()
		if err := <-writer.Errors(); !errors.Is(err, want) {
			t.Fatalf("reported error = %v", err)
		}
		if err := writer.Flush(context.Background()); !errors.Is(err, want) {
			t.Fatalf("flush error = %v", err)
		}
		synctest.Wait()
		if _, calls := storage.result(); calls != 2 {
			t.Fatalf("failed writes retried without a request: calls = %d", calls)
		}
		requireNoError(t, writer.Flush(context.Background()))
		requireNoError(t, writer.Close(context.Background()))
		if values, calls := storage.result(); !reflect.DeepEqual(values, []int{7}) || calls != 3 {
			t.Fatalf("retry result = %v after %d calls", values, calls)
		}
	})
}

func TestAsyncFlushDeadlineDoesNotCancelWorker(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		gate := make(chan struct{})
		storage := &asyncStorage{gate: gate, entered: make(chan int, 10)}
		store := New(0)
		writer, err := NewPersistence(store, storage, JSONCodec[int]{}).AutoSaveAsync(context.Background(), AsyncOptions{})
		requireNoError(t, err)
		<-storage.entered
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := writer.Flush(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("flush error = %v", err)
		}
		requireNoError(t, store.Set(1))
		close(gate)
		requireNoError(t, writer.Close(context.Background()))
		if values, _ := storage.result(); !reflect.DeepEqual(values, []int{0, 1}) {
			t.Fatalf("worker was canceled by one flush: %v", values)
		}
	})
}

func TestAsyncCloseFailureAndDeadlineReleaseWorker(t *testing.T) {
	for _, reason := range []string{"failure", "deadline"} {
		t.Run(reason, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				want := errors.New("disk full")
				storage := &asyncStorage{failures: 1, err: want}
				ctx := context.Background()
				cancel := func() {}
				if reason == "deadline" {
					storage.gate = make(chan struct{})
					want = context.DeadlineExceeded
					ctx, cancel = context.WithTimeout(ctx, time.Second)
				}
				defer cancel()
				store := New(3)
				writer, err := NewPersistence(store, storage, JSONCodec[int]{}).AutoSaveAsync(context.Background(), AsyncOptions{Debounce: time.Hour})
				requireNoError(t, err)
				if err := writer.Close(ctx); !errors.Is(err, want) {
					t.Fatalf("close error = %v, want %v", err, want)
				}
				synctest.Wait()
				for range writer.Errors() {
				}
				if err := writer.Flush(context.Background()); !errors.Is(err, ErrClosed) {
					t.Fatalf("flush after close = %v", err)
				}
				requireNoError(t, store.Set(4))
				if err := writer.Close(context.Background()); !errors.Is(err, want) {
					t.Fatalf("idempotent close error = %v", err)
				}
			})
		})
	}
}

func TestAsyncParentAndStoreShutdownReleaseSubscriptions(t *testing.T) {
	for _, reason := range []string{"context", "store"} {
		t.Run(reason, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				store := New(0)
				writer, err := NewPersistence(store, &asyncStorage{}, JSONCodec[int]{}).AutoSaveAsync(ctx, AsyncOptions{Debounce: time.Hour})
				requireNoError(t, err)
				want := context.Canceled
				if reason == "store" {
					want = ErrClosed
					requireNoError(t, store.Close())
				} else {
					cancel()
				}
				synctest.Wait()
				for range writer.Errors() {
				}
				if err := writer.Flush(context.Background()); !errors.Is(err, want) {
					t.Fatalf("flush after shutdown = %v", err)
				}
				if count := listenerCount(store); count != 0 {
					t.Fatalf("remaining subscriptions = %d", count)
				}
			})
		})
	}
}

func TestAsyncConcurrentFlushAndCloseAreIdempotent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store, storage := New(0), &asyncStorage{}
		writer, err := NewPersistence(store, storage, JSONCodec[int]{}).AutoSaveAsync(context.Background(), AsyncOptions{Debounce: time.Hour})
		requireNoError(t, err)
		requireNoError(t, store.Set(9))
		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() {
				if err := writer.Flush(context.Background()); err != nil {
					t.Error(err)
				}
			})
		}
		wg.Wait()
		for range 10 {
			wg.Go(func() {
				if err := writer.Close(context.Background()); err != nil {
					t.Error(err)
				}
			})
		}
		wg.Wait()
		if values, _ := storage.result(); !reflect.DeepEqual(values, []int{9}) {
			t.Fatalf("concurrent flush/close writes = %v", values)
		}
	})
}

func TestAsyncCloseOverlapsAnExistingFlush(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		gate := make(chan struct{})
		storage := &asyncStorage{gate: gate, entered: make(chan int, 10)}
		writer, err := NewPersistence(New(5), storage, JSONCodec[int]{}).AutoSaveAsync(context.Background(), AsyncOptions{})
		requireNoError(t, err)
		<-storage.entered
		flushed, closed := make(chan error, 1), make(chan error, 1)
		go func() { flushed <- writer.Flush(context.Background()) }()
		synctest.Wait()
		go func() { closed <- writer.Close(context.Background()) }()
		synctest.Wait()
		close(gate)
		requireNoError(t, <-flushed)
		requireNoError(t, <-closed)
	})
}

func TestAsyncErrorStreamDoesNotBlockShutdownWhenFull(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		want := errors.New("unwritable storage")
		storage := &asyncStorage{failures: 100, err: want}
		writer, err := NewPersistence(New(0), storage, JSONCodec[int]{}).AutoSaveAsync(context.Background(), AsyncOptions{Debounce: time.Hour})
		requireNoError(t, err)
		for range 20 {
			if err := writer.Flush(context.Background()); !errors.Is(err, want) {
				t.Fatalf("flush error = %v", err)
			}
		}
		if err := writer.Close(context.Background()); !errors.Is(err, want) {
			t.Fatalf("close error = %v", err)
		}
		for range writer.Errors() {
		}
	})
}

func TestAsyncClonedVersionedStateRoundTripsThroughFile(t *testing.T) {
	store := NewWithClone(map[string]int{"count": 0}, func(v map[string]int) map[string]int { return maps.Clone(v) })
	storage := FileStorage{Path: filepath.Join(t.TempDir(), "state.json")}
	codec := VersionedJSONCodec[map[string]int]{Version: 1}
	p := NewPersistence(store, storage, codec)
	writer, err := p.AutoSaveAsync(context.Background(), AsyncOptions{})
	requireNoError(t, err)
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range 100 {
				if err := store.Update(func(v *map[string]int) { (*v)["count"]++ }); err != nil {
					t.Error(err)
				}
			}
		})
	}
	wg.Wait()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	requireNoError(t, writer.Close(ctx))
	restored := New(map[string]int{})
	requireNoError(t, NewPersistence(restored, storage, codec).Load(ctx))
	if count := restored.Get()["count"]; count != 400 {
		t.Fatalf("restored count = %d, want 400", count)
	}
}

func TestAsyncRejectsInvalidConfiguration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	valid := NewPersistence(New(0), &asyncStorage{}, JSONCodec[int]{})
	closed := New(0)
	requireNoError(t, closed.Close())
	for _, test := range []struct {
		name    string
		p       *Persistence[int]
		ctx     context.Context
		options AsyncOptions
		want    error
	}{
		{"nil", nil, nil, AsyncOptions{}, ErrInvalidPersistence},
		{"codec", NewPersistence[int](New(0), &asyncStorage{}, nil), nil, AsyncOptions{}, ErrInvalidPersistence},
		{"debounce", valid, nil, AsyncOptions{Debounce: -1}, ErrInvalidPersistence},
		{"canceled", valid, ctx, AsyncOptions{}, context.Canceled},
		{"closed", NewPersistence(closed, &asyncStorage{}, JSONCodec[int]{}), nil, AsyncOptions{}, ErrClosed},
	} {
		t.Run(test.name, func(t *testing.T) {
			writer, err := test.p.AutoSaveAsync(test.ctx, test.options)
			if writer != nil || !errors.Is(err, test.want) {
				t.Fatalf("writer, error = %v, %v", writer, err)
			}
		})
	}
}

type panickingStorage struct {
	asyncStorage
	panicking atomic.Bool
}

func (s *panickingStorage) Save(ctx context.Context, data []byte) error {
	if s.panicking.Load() {
		panic("storage bug")
	}
	return s.asyncStorage.Save(ctx, data)
}

type panickingCodec struct {
	JSONCodec[int]
	panicking *atomic.Bool
}

func (c panickingCodec) Encode(value int) ([]byte, error) {
	if c.panicking.Load() {
		panic("codec bug")
	}
	return c.JSONCodec.Encode(value)
}

// The write-behind worker runs Storage.Save, Codec.Encode and Validate on a goroutine the library owns, so a panic there must become a PanicError write failure instead of killing the process.
func TestAsyncPersistenceContainsWorkerPanics(t *testing.T) {
	for _, boundary := range []string{"storage", "codec", "validate"} {
		t.Run(boundary, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				storage := &panickingStorage{}
				panicking := &storage.panicking
				var codec Codec[int] = JSONCodec[int]{}
				var options PersistenceOptions[int]
				want := boundary + " bug"
				switch boundary {
				case "codec":
					panicking = &atomic.Bool{}
					codec = panickingCodec{panicking: panicking}
				case "validate":
					panicking = &atomic.Bool{}
					options.Validate = func(int) error {
						if panicking.Load() {
							panic("validate bug")
						}
						return nil
					}
				}
				panicking.Store(true)
				store := New(5)
				writer, err := NewPersistenceWithOptions(store, storage, codec, options).AutoSaveAsync(context.Background(), AsyncOptions{})
				requireNoError(t, err)
				var panicErr *PanicError
				if err := writer.Flush(context.Background()); !errors.As(err, &panicErr) || panicErr.Value != want {
					t.Fatalf("flush error = %v, want PanicError(%q)", err, want)
				}
				// Let a retry forced by the failed Flush settle before inspecting the worker.
				synctest.Wait()
				select {
				case err := <-writer.Errors():
					if !errors.As(err, &panicErr) || panicErr.Value != want {
						t.Fatalf("reported error = %v", err)
					}
				default:
					t.Fatal("panic was not reported on Errors")
				}
				if status := writer.Status().Get(); !status.Pending || status.Saving || status.Closed || status.Error == "" {
					t.Fatalf("status after panic = %+v", status)
				}
				panicking.Store(false)
				requireNoError(t, writer.Flush(context.Background()))
				requireNoError(t, store.Set(6))
				requireNoError(t, writer.Close(context.Background()))
				if values, _ := storage.result(); !reflect.DeepEqual(values, []int{5, 6}) {
					t.Fatalf("writes after recovery = %v", values)
				}
			})
		})
	}
}

// Updates arriving faster than Debounce restart the trailing delay forever. MaxWait bounds that postponement from the start of each wait, and zero keeps the unbounded behaviour.
func TestAsyncMaxWaitBoundsDebouncePostponement(t *testing.T) {
	for _, tc := range []struct {
		name          string
		maxWait       time.Duration
		before, after []int
	}{
		{name: "unbounded", before: nil, after: []int{100}},
		// Waits start at 0 ms and 1020 ms, so the writes at 1000 ms and 2020 ms capture the updates made at 990 ms and 2010 ms; the third wait has not expired when updates stop at 3000 ms.
		{name: "bounded", maxWait: time.Second, before: []int{33, 67}, after: []int{33, 67, 100}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				store, storage := New(0), &asyncStorage{}
				writer, err := NewPersistence(store, storage, JSONCodec[int]{}).AutoSaveAsync(context.Background(), AsyncOptions{Debounce: 100 * time.Millisecond, MaxWait: tc.maxWait})
				requireNoError(t, err)
				for i := 1; i <= 100; i++ {
					time.Sleep(30 * time.Millisecond)
					requireNoError(t, store.Set(i))
				}
				synctest.Wait()
				if values, _ := storage.result(); !reflect.DeepEqual(values, tc.before) {
					t.Fatalf("writes during continuous updates = %v, want %v", values, tc.before)
				}
				requireNoError(t, writer.Close(context.Background()))
				if values, _ := storage.result(); !reflect.DeepEqual(values, tc.after) {
					t.Fatalf("writes after close = %v, want %v", values, tc.after)
				}
			})
		})
	}
}

func TestAsyncMaxWaitShorterThanDebounceAndValidation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store, storage := New(1), &asyncStorage{}
		persistence := NewPersistence(store, storage, JSONCodec[int]{})
		if _, err := persistence.AutoSaveAsync(context.Background(), AsyncOptions{Debounce: time.Second, MaxWait: -time.Second}); !errors.Is(err, ErrInvalidPersistence) {
			t.Fatalf("negative MaxWait = %v", err)
		}
		writer, err := persistence.AutoSaveAsync(context.Background(), AsyncOptions{Debounce: time.Hour, MaxWait: time.Minute})
		requireNoError(t, err)
		time.Sleep(time.Minute - time.Millisecond)
		synctest.Wait()
		if _, calls := storage.result(); calls != 0 {
			t.Fatal("MaxWait wrote before its deadline")
		}
		time.Sleep(time.Millisecond)
		synctest.Wait()
		if values, _ := storage.result(); !reflect.DeepEqual(values, []int{1}) {
			t.Fatalf("writes at MaxWait = %v", values)
		}
		requireNoError(t, writer.Close(context.Background()))
	})
}

// An update made while a save is running is written after Debounce, like any other update, instead of in a follow-up write that starts as soon as the save returns.
func TestAsyncUpdateDuringSaveWaitsForDebounce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		gate := make(chan struct{})
		storage := &asyncStorage{gate: gate, entered: make(chan int, 10)}
		store := New(0)
		writer, err := NewPersistence(store, storage, JSONCodec[int]{}).AutoSaveAsync(context.Background(), AsyncOptions{Debounce: time.Hour})
		requireNoError(t, err)
		time.Sleep(time.Hour)
		if got := <-storage.entered; got != 0 {
			t.Fatalf("initial write = %d", got)
		}
		requireNoError(t, store.Set(1))
		close(gate)
		synctest.Wait()
		time.Sleep(time.Hour - time.Millisecond)
		synctest.Wait()
		select {
		case got := <-storage.entered:
			t.Errorf("value %d made during a save was written before Debounce elapsed", got)
		default:
		}
		time.Sleep(time.Millisecond)
		synctest.Wait()
		select {
		case got := <-storage.entered:
			if got != 1 {
				t.Errorf("debounced write = %d, want 1", got)
			}
		default:
			t.Error("value made during a save was not written after Debounce")
		}
		requireNoError(t, writer.Close(context.Background()))
		if values, _ := storage.result(); !reflect.DeepEqual(values, []int{0, 1}) {
			t.Fatalf("writes = %v", values)
		}
	})
}

// A zero Debounce, or a Flush waiting on the running save, still writes an update made during that save as soon as the save returns.
func TestAsyncUpdateDuringSaveWritesAtOnceWithoutDebounceOrWithFlush(t *testing.T) {
	for _, tc := range []struct {
		name     string
		debounce time.Duration
		flush    bool
	}{
		{name: "zero debounce", debounce: 0},
		{name: "flush", debounce: time.Hour, flush: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				gate := make(chan struct{})
				storage := &asyncStorage{gate: gate, entered: make(chan int, 10)}
				store := New(0)
				writer, err := NewPersistence(store, storage, JSONCodec[int]{}).AutoSaveAsync(context.Background(), AsyncOptions{Debounce: tc.debounce})
				requireNoError(t, err)
				time.Sleep(tc.debounce)
				if got := <-storage.entered; got != 0 {
					t.Fatalf("initial write = %d", got)
				}
				requireNoError(t, store.Set(1))
				flushed := make(chan error, 1)
				if tc.flush {
					go func() { flushed <- writer.Flush(context.Background()) }()
					synctest.Wait()
				}
				close(gate)
				synctest.Wait()
				select {
				case got := <-storage.entered:
					if got != 1 {
						t.Errorf("follow-up write = %d, want 1", got)
					}
				default:
					t.Error("update made during a save was not written when the save returned")
				}
				if tc.flush {
					requireNoError(t, <-flushed)
				}
				requireNoError(t, writer.Close(context.Background()))
			})
		})
	}
}

type slowStorage struct {
	asyncStorage
	delay time.Duration
}

func (s *slowStorage) Save(ctx context.Context, data []byte) error {
	time.Sleep(s.delay)
	return s.asyncStorage.Save(ctx, data)
}

// Storage slower than the update rate must not turn a debounced writer into back-to-back writes: Debounce and MaxWait still bound how often updates that arrive during a save are written.
func TestAsyncSlowStorageKeepsDebounceAndMaxWait(t *testing.T) {
	for _, tc := range []struct {
		name          string
		maxWait       time.Duration
		before, after []int
	}{
		{name: "unbounded", before: []int{0}, after: []int{0, 1000}},
		// The initial save runs from 1.0 s to 1.1 s; each later wait starts when the previous save returns and ends at MaxWait, at 3.1 s, 5.2 s, 7.3 s and 9.4 s, capturing the updates made 5 ms earlier. The next deadline, 11.5 s, falls after the updates stop.
		{name: "bounded", maxWait: 2 * time.Second, before: []int{0, 210, 420, 630, 840}, after: []int{0, 210, 420, 630, 840, 1000}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				store, storage := New(0), &slowStorage{delay: 100 * time.Millisecond}
				writer, err := NewPersistence(store, storage, JSONCodec[int]{}).AutoSaveAsync(context.Background(), AsyncOptions{Debounce: time.Second, MaxWait: tc.maxWait})
				requireNoError(t, err)
				// Updates every 10 ms from 1.005 s to 10.995 s, starting while the initial save runs and never at the same instant as a timer.
				time.Sleep(1005 * time.Millisecond)
				for i := 1; i <= 1000; i++ {
					requireNoError(t, store.Set(i))
					time.Sleep(10 * time.Millisecond)
				}
				synctest.Wait()
				if values, _ := storage.result(); !reflect.DeepEqual(values, tc.before) {
					t.Errorf("writes during continuous updates = %v, want %v", values, tc.before)
				}
				requireNoError(t, writer.Close(context.Background()))
				if values, _ := storage.result(); !reflect.DeepEqual(values, tc.after) {
					t.Fatalf("writes after close = %v, want %v", values, tc.after)
				}
			})
		})
	}
}
