package bridge_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/yldm-tech/nodra"
	"github.com/yldm-tech/nodra/bridge"
)

// counts returns the counts the frames carry, in order.
func counts(t *testing.T, f *frames) []int {
	t.Helper()
	var out []int
	for _, frame := range f.all() {
		out = append(out, decode[pageState](t, frame).Count)
	}
	return out
}

// watchCounter exposes a counter on a bridge with options and starts a watch of it. It must run inside a synctest bubble.
func watchCounter(t *testing.T, options bridge.Options) (*nodra.Store[state], *frames, func()) {
	t.Helper()
	store := nodra.New(state{})
	b := bridge.New(options)
	if _, err := bridge.ExposeProjection(b, "counter", store, project); err != nil {
		t.Fatal(err)
	}
	f := newFrames()
	ctx, cancel := context.WithCancel(context.Background())
	done := watch(ctx, b, "counter", 0, f)
	synctest.Wait()
	return store, f, func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("watch: %v", err)
		}
		b.Close()
		_ = store.Close()
	}
}

func increment(t *testing.T, store *nodra.Store[state]) {
	t.Helper()
	if err := store.Update(func(s *state) { s.Count++ }); err != nil {
		t.Fatal(err)
	}
}

// drain takes the frame notifications so the frames' buffered channel never fills during a long burst.
func drain(f *frames) {
	for {
		select {
		case <-f.more:
		default:
			return
		}
	}
}

func TestFrameIntervalCoalescesABurstAndSendsTheLatest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store, f, stop := watchCounter(t, bridge.Options{FrameInterval: 100 * time.Millisecond})
		defer stop()
		// 50 changes 10 ms apart: 500 ms of changes at ten times the rate the interval allows.
		for range 50 {
			increment(t, store)
			synctest.Wait()
			drain(f)
			time.Sleep(10 * time.Millisecond)
		}
		time.Sleep(100 * time.Millisecond)
		synctest.Wait()
		got := counts(t, f)
		if last := got[len(got)-1]; last != 50 {
			t.Fatalf("the last frame carries %d, want the final 50; frames %v", last, got)
		}
		// The first frame, then at most one per 100 ms of the 500 ms burst and the trailing one.
		if len(got) > 7 {
			t.Fatalf("sent %d frames for a 500 ms burst with a 100 ms interval: %v", len(got), got)
		}
		for i := 1; i < len(got); i++ {
			if got[i] <= got[i-1] {
				t.Fatalf("frames go back: %v", got)
			}
		}
	})
}

func TestFrameIntervalSendsAChangeAfterAQuietSpellAtOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store, f, stop := watchCounter(t, bridge.Options{FrameInterval: 100 * time.Millisecond})
		defer stop()
		time.Sleep(time.Second)
		before := time.Now()
		increment(t, store)
		synctest.Wait()
		got := counts(t, f)
		if len(got) != 2 || got[1] != 1 {
			t.Fatalf("frames after a quiet change: %v, want [0 1]", got)
		}
		if elapsed := time.Since(before); elapsed != 0 {
			t.Fatalf("the change after a quiet spell waited %v", elapsed)
		}
	})
}

func TestWithoutFrameIntervalEachChangeIsSent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store, f, stop := watchCounter(t, bridge.Options{})
		defer stop()
		for range 20 {
			increment(t, store)
			synctest.Wait()
		}
		if got := counts(t, f); len(got) != 21 || got[20] != 20 {
			t.Fatalf("frames without an interval: %v, want every count from 0 to 20", got)
		}
	})
}

func TestAWatchWaitingOutItsIntervalEndsWhenTheStateGoesAway(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store := nodra.New(state{})
		b := bridge.New(bridge.Options{FrameInterval: time.Hour})
		defer b.Close()
		stopExposing, err := bridge.ExposeProjection(b, "counter", store, project)
		if err != nil {
			t.Fatal(err)
		}
		f := newFrames()
		done := watch(context.Background(), b, "counter", 0, f)
		synctest.Wait()
		stopExposing()
		if err := <-done; !errors.Is(err, bridge.ErrClosed) {
			t.Fatalf("a watch in its interval ended with %v when the exposure stopped, want ErrClosed", err)
		}
		_ = store.Close()
	})
}
