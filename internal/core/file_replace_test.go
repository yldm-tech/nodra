package core

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

var errContention = errors.New("contention")

func isContention(err error) bool { return errors.Is(err, errContention) }

func TestRetryTransientSucceedsOnceContentionEnds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		calls := 0
		start := time.Now()
		err := retryTransient(context.Background(), replaceRetryBudget, func() error {
			calls++
			if calls < 4 {
				return errContention
			}
			return nil
		}, isContention)
		if err != nil {
			t.Fatalf("retryTransient = %v, want nil", err)
		}
		if calls != 4 {
			t.Fatalf("calls = %d, want 4", calls)
		}
		// Three delays of at most 1, 2 and 4 ms.
		if elapsed := time.Since(start); elapsed > 7*time.Millisecond {
			t.Fatalf("elapsed = %v, want at most 7ms", elapsed)
		}
	})
}

func TestRetryTransientReturnsOtherErrorsAtOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		other := errors.New("other")
		calls := 0
		start := time.Now()
		err := retryTransient(context.Background(), replaceRetryBudget, func() error {
			calls++
			return other
		}, isContention)
		if err != other || calls != 1 || time.Since(start) != 0 {
			t.Fatalf("retryTransient = %v after %d calls and %v, want the error after one call and no delay", err, calls, time.Since(start))
		}
	})
}

func TestRetryTransientStopsAtBudget(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var attempts []time.Time
		start := time.Now()
		err := retryTransient(context.Background(), replaceRetryBudget, func() error {
			attempts = append(attempts, time.Now())
			return errContention
		}, isContention)
		if !errors.Is(err, errContention) {
			t.Fatalf("retryTransient = %v, want the last contention error", err)
		}
		if elapsed := time.Since(start); elapsed != replaceRetryBudget {
			t.Fatalf("elapsed = %v, want the budget %v", elapsed, replaceRetryBudget)
		}
		// Delays grow, are capped, and the budget allows far fewer attempts than a fixed 1 ms delay would.
		if len(attempts) < 20 || len(attempts) > 50 {
			t.Fatalf("attempts = %d, want between 20 and 50 within %v", len(attempts), replaceRetryBudget)
		}
		for i := 1; i < len(attempts); i++ {
			if gap := attempts[i].Sub(attempts[i-1]); gap <= 0 || gap > replaceRetryMaxDelay {
				t.Fatalf("delay before attempt %d = %v, want within (0, %v]", i, gap, replaceRetryMaxDelay)
			}
		}
	})
}

func TestRetryTransientStopsWhenContextEnds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		calls := 0
		start := time.Now()
		err := retryTransient(ctx, replaceRetryBudget, func() error {
			calls++
			if calls == 2 {
				cancel()
			}
			return errContention
		}, isContention)
		if !errors.Is(err, context.Canceled) || !errors.Is(err, errContention) {
			t.Fatalf("retryTransient = %v, want both context.Canceled and the last contention error", err)
		}
		if calls != 2 {
			t.Fatalf("calls = %d, want 2", calls)
		}
		if elapsed := time.Since(start); elapsed > 2*time.Millisecond {
			t.Fatalf("elapsed = %v, want the cancellation to end the wait", elapsed)
		}
	})
}
