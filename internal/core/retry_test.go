package core

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func TestAsyncActionRetriesOnlyWhenExplicitlyIdempotent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store := New(0)
		var calls atomic.Int32
		want := errors.New("temporary")
		action, err := NewAsyncActionWithOptions(store, "retry", func(context.Context, int, func(float64)) (int, error) {
			if calls.Add(1) < 3 {
				return 0, want
			}
			return 7, nil
		}, func(s *int, value int) error { *s = value; return nil }, ActionOptions[int]{Retry: RetryOptions{MaxAttempts: 3, Backoff: 100 * time.Millisecond, MaxBackoff: 150 * time.Millisecond, Idempotent: true}})
		requireNoError(t, err)
		defer func() { _ = action.Close(context.Background()) }()
		task, err := action.Run(context.Background(), 0)
		requireNoError(t, err)
		requireNoError(t, task.Wait(context.Background()))
		if calls.Load() != 3 || store.Get() != 7 || task.Status().Get().Metrics.Attempts != 3 {
			t.Fatalf("calls=%d state=%d metrics=%+v", calls.Load(), store.Get(), task.Status().Get().Metrics)
		}
		metrics := action.Metrics()
		if metrics.Accepted != 1 || metrics.Completed != 1 || metrics.Succeeded != 1 || metrics.Retried != 2 || metrics.DurationSum <= 0 || metrics.DurationMax <= 0 {
			t.Fatalf("aggregate metrics: %+v", metrics)
		}
	})
	if _, err := NewAsyncActionWithOptions(New(0), "invalid", func(context.Context, int, func(float64)) (int, error) { return 0, nil }, func(*int, int) error { return nil }, ActionOptions[int]{Retry: RetryOptions{MaxAttempts: 2}}); !errors.Is(err, ErrInvalidAction) {
		t.Fatalf("non-idempotent retry accepted: %v", err)
	}
}

func TestAsyncActionRetryExhaustionReportsLastError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		want := errors.New("permanent")
		action, err := NewAsyncActionWithOptions(New(0), "retry", func(context.Context, int, func(float64)) (int, error) { calls.Add(1); return 0, want }, func(*int, int) error { return nil }, ActionOptions[int]{Retry: RetryOptions{MaxAttempts: 3, Backoff: time.Second, Idempotent: true}})
		requireNoError(t, err)
		defer func() { _ = action.Close(context.Background()) }()
		started := time.Now()
		task, err := action.Run(context.Background(), 0)
		requireNoError(t, err)
		if err := task.Wait(context.Background()); !errors.Is(err, want) {
			t.Fatalf("exhausted retry = %v", err)
		}
		// Without MaxBackoff the delays are 1s and 2s.
		if elapsed := time.Since(started); calls.Load() != 3 || task.Status().Get().Metrics.Attempts != 3 || elapsed != 3*time.Second {
			t.Fatalf("calls=%d metrics=%+v elapsed=%v", calls.Load(), task.Status().Get().Metrics, elapsed)
		}
		if metrics := action.Metrics(); metrics.Failed != 1 || metrics.Retried != 2 {
			t.Fatalf("aggregate metrics: %+v", metrics)
		}
	})
}

func TestAsyncActionRetryBackoffStaysCappedForLargeBudgets(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const attempts = 80
		var times []time.Time
		want := errors.New("down")
		// The work calls run one after another on the task goroutine, so the slice needs no lock.
		action, err := NewAsyncActionWithOptions(New(0), "retry", func(context.Context, int, func(float64)) (int, error) {
			times = append(times, time.Now())
			return 0, want
		}, func(*int, int) error { return nil }, ActionOptions[int]{Retry: RetryOptions{MaxAttempts: attempts, Backoff: time.Second, MaxBackoff: 30 * time.Second, Idempotent: true}})
		requireNoError(t, err)
		defer func() { _ = action.Close(context.Background()) }()
		task, err := action.Run(context.Background(), 0)
		requireNoError(t, err)
		if err := task.Wait(context.Background()); !errors.Is(err, want) {
			t.Fatalf("exhausted retry = %v", err)
		}
		if len(times) != attempts {
			t.Fatalf("work ran %d times, want %d", len(times), attempts)
		}
		for n := 1; n < len(times); n++ {
			want := 30 * time.Second
			if n <= 5 {
				want = time.Second << (n - 1)
			}
			if delay := times[n].Sub(times[n-1]); delay != want {
				t.Fatalf("delay before attempt %d = %v, want %v", n+1, delay, want)
			}
		}
		if metrics := action.Metrics(); metrics.Failed != 1 || metrics.Retried != attempts-1 {
			t.Fatalf("aggregate metrics: %+v", metrics)
		}
	})
}

func TestAsyncActionRetryStopsDuringBackoffWithoutAnotherAttempt(t *testing.T) {
	cases := []struct {
		name string
		stop func(t *testing.T, action *AsyncAction[int, int, int], task *Task)
		want error
	}{
		{name: "cancel", stop: func(_ *testing.T, _ *AsyncAction[int, int, int], task *Task) { task.Cancel() }, want: context.Canceled},
		{name: "close", stop: func(t *testing.T, action *AsyncAction[int, int, int], _ *Task) {
			requireNoError(t, action.Close(context.Background()))
		}, want: ErrClosed},
		{name: "supersede", stop: func(t *testing.T, action *AsyncAction[int, int, int], _ *Task) {
			next, err := action.Run(context.Background(), 2)
			requireNoError(t, err)
			requireNoError(t, next.Wait(context.Background()))
		}, want: ErrSuperseded},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var calls, deadCalls atomic.Int32
				action, err := NewAsyncActionWithOptions(New(0), "retry", func(ctx context.Context, input int, _ func(float64)) (int, error) {
					if input != 1 {
						return input, nil
					}
					calls.Add(1)
					if ctx.Err() != nil {
						deadCalls.Add(1)
					}
					return 0, errors.New("temporary")
				}, func(s *int, v int) error { *s = v; return nil }, ActionOptions[int]{Retry: RetryOptions{MaxAttempts: 5, Backoff: time.Hour, Idempotent: true}})
				requireNoError(t, err)
				defer func() { _ = action.Close(context.Background()) }()
				task, err := action.Run(context.Background(), 1)
				requireNoError(t, err)
				// The first attempt has failed and the task is blocked in its one-hour backoff.
				synctest.Wait()
				if calls.Load() != 1 {
					t.Fatalf("calls before stop = %d", calls.Load())
				}
				tc.stop(t, action, task)
				if err := task.Wait(context.Background()); !errors.Is(err, tc.want) {
					t.Fatalf("stopped retry = %v, want %v", err, tc.want)
				}
				if calls.Load() != 1 || deadCalls.Load() != 0 {
					t.Fatalf("work ran %d times (%d with a canceled context) after stopping during backoff", calls.Load(), deadCalls.Load())
				}
				if attempts := task.Status().Get().Metrics.Attempts; attempts != 1 {
					t.Fatalf("attempts=%d", attempts)
				}
				if retried := action.Metrics().Retried; retried != 0 {
					t.Fatalf("retried=%d", retried)
				}
			})
		})
	}
}
