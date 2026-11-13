package core

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"
)

// replaceRetryBudget bounds how long a file replacement keeps retrying transient failures, measured from the first attempt. The Go toolchain's cmd/internal/robustio retries the same Windows rename failures for the same 2 s.
const replaceRetryBudget = 2 * time.Second

// replaceRetryMaxDelay caps one delay between replacement attempts.
const replaceRetryMaxDelay = 100 * time.Millisecond

// retryTransient calls op until it succeeds, fails with an error that transient rejects, budget has elapsed since the first call, or ctx is done, and returns the last error from op in the first three cases. Delays start at 1 ms and double up to replaceRetryMaxDelay; each is jittered to between half and all of its nominal length so writers that collide do not retry in lockstep. When ctx ends during a delay the result wraps both ctx.Err() and the last error from op.
func retryTransient(ctx context.Context, budget time.Duration, op func() error, transient func(error) bool) error {
	deadline := time.Now().Add(budget)
	delay := time.Millisecond
	for {
		err := op()
		if err == nil || !transient(err) {
			return err
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return err
		}
		timer := time.NewTimer(min(delay/2+rand.N(delay/2+1), remaining))
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("%w; last attempt: %w", ctx.Err(), err)
		case <-timer.C:
		}
		delay = min(delay*2, replaceRetryMaxDelay)
	}
}
