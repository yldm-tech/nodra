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

func TestAsyncActionRetryExhaustionAndCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		want := errors.New("permanent")
		action, err := NewAsyncActionWithOptions(New(0), "retry", func(context.Context, int, func(float64)) (int, error) { calls.Add(1); return 0, want }, func(*int, int) error { return nil }, ActionOptions[int]{Retry: RetryOptions{MaxAttempts: 3, Backoff: time.Second, Idempotent: true}})
		requireNoError(t, err)
		defer func() { _ = action.Close(context.Background()) }()
		task, err := action.Run(context.Background(), 0)
		requireNoError(t, err)
		time.Sleep(500 * time.Millisecond)
		task.Cancel()
		if err := task.Wait(context.Background()); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel = %v", err)
		}
		if calls.Load() != 2 {
			t.Fatalf("calls=%d", calls.Load())
		}
		if task.Status().Get().Metrics.Attempts != 2 {
			t.Fatalf("attempts=%d", task.Status().Get().Metrics.Attempts)
		}
	})
}
