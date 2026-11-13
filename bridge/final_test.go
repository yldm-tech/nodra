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

// These tests check that a page ends up with a source's final value when the source closes, and that stopping an exposure or closing the bridge sends nothing more.

func TestFrameIntervalSendsTheFinalValueWhenTheSourceCloses(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store := nodra.New(state{})
		b := bridge.New(bridge.Options{FrameInterval: 100 * time.Millisecond})
		defer b.Close()
		if _, err := bridge.ExposeProjection(b, "counter", store, project); err != nil {
			t.Fatal(err)
		}
		f := newFrames()
		done := watch(context.Background(), b, "counter", 0, f)
		synctest.Wait()
		start := time.Now()
		increment(t, store)
		increment(t, store)
		_ = store.Close()
		if err := <-done; !errors.Is(err, bridge.ErrClosed) {
			t.Fatalf("the watch of a closed source ended with %v, want ErrClosed", err)
		}
		if got := counts(t, f); len(got) != 2 || got[1] != 2 {
			t.Fatalf("frames %v, want [0 2]: the final value of the closed source", got)
		}
		// The final frame still keeps the interval after the first one.
		if elapsed := time.Since(start); elapsed != 100*time.Millisecond {
			t.Fatalf("the final frame went out %v after the first, want the 100ms interval", elapsed)
		}
	})
}

func TestFrameIntervalEndsAtOnceWhenTheSourceClosesWithNothingPending(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store := nodra.New(state{})
		b := bridge.New(bridge.Options{FrameInterval: time.Hour})
		defer b.Close()
		if _, err := bridge.ExposeProjection(b, "counter", store, project); err != nil {
			t.Fatal(err)
		}
		f := newFrames()
		done := watch(context.Background(), b, "counter", 0, f)
		synctest.Wait()
		start := time.Now()
		// A change the projection leaves out is not a value the page lacks.
		if err := store.Update(func(s *state) { s.Note = "hidden" }); err != nil {
			t.Fatal(err)
		}
		_ = store.Close()
		if err := <-done; !errors.Is(err, bridge.ErrClosed) {
			t.Fatalf("watch ended with %v, want ErrClosed", err)
		}
		if elapsed := time.Since(start); elapsed != 0 {
			t.Fatalf("a watch with nothing to send waited %v after its source closed", elapsed)
		}
		if got := counts(t, f); len(got) != 1 {
			t.Fatalf("frames %v, want only the first", got)
		}
	})
}

func TestTheFinalTaskStatusReachesThePage(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		app := nodra.New(state{})
		defer func() { _ = app.Close() }()
		gate := make(chan struct{})
		action, err := nodra.NewAsyncAction(app, "job", func(_ context.Context, _ int, report func(float64)) (int, error) {
			<-gate
			for i := 1; i <= 10; i++ {
				report(float64(i) / 10)
			}
			return 1, nil
		}, func(s *state, r int) error { s.Count += r; return nil })
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = action.Close(context.Background()) }()
		task, err := action.Run(context.Background(), 0)
		if err != nil {
			t.Fatal(err)
		}
		b := bridge.New(bridge.Options{FrameInterval: 16 * time.Millisecond})
		defer b.Close()
		if _, err := bridge.Expose(b, "task", task.Status()); err != nil {
			t.Fatal(err)
		}
		f := newFrames()
		done := watch(context.Background(), b, "task", 0, f)
		synctest.Wait()
		close(gate)
		if err := <-done; !errors.Is(err, bridge.ErrClosed) {
			t.Fatalf("the watch of a finished task ended with %v, want ErrClosed", err)
		}
		all := f.all()
		last := decode[nodra.AsyncState](t, all[len(all)-1])
		if last.Running || last.Phase != nodra.TaskSucceeded || last.Progress != 1 {
			t.Fatalf("the page's last task status is %+v, want the succeeded one", last)
		}
	})
}

// TestTheFinalValueIsSentWhenTheSourceClosesDuringASend covers a watch without an interval: when the source changes and closes while a send blocks, Watch finds a change and the closed source at once, and must not end before sending the final value. Go picks between ready select cases at random, so the test repeats.
func TestTheFinalValueIsSentWhenTheSourceClosesDuringASend(t *testing.T) {
	for range 64 {
		synctest.Test(t, func(t *testing.T) {
			store := nodra.New(state{})
			b := bridge.New(bridge.Options{})
			defer b.Close()
			if _, err := bridge.ExposeProjection(b, "counter", store, project); err != nil {
				t.Fatal(err)
			}
			f := newFrames()
			release := make(chan struct{})
			done := make(chan error, 1)
			go func() {
				done <- b.Watch(context.Background(), "counter", 0, func(frame bridge.Frame) error {
					if decode[pageState](t, frame).Count == 1 {
						<-release
					}
					return f.send(frame)
				})
			}()
			synctest.Wait()
			increment(t, store)
			synctest.Wait() // the send of count 1 blocks
			increment(t, store)
			_ = store.Close()
			close(release)
			if err := <-done; !errors.Is(err, bridge.ErrClosed) {
				t.Fatalf("watch ended with %v, want ErrClosed", err)
			}
			if got := counts(t, f); got[len(got)-1] != 2 {
				t.Fatalf("frames %v, want the final 2 last", got)
			}
		})
	}
}

// TestStoppingAnExposureDropsAHeldBackChange records the choice for a withdrawn state: the app stopped exposing it, so a watch ends at once and sends nothing more, not even a change it was holding back.
func TestStoppingAnExposureDropsAHeldBackChange(t *testing.T) {
	for _, tc := range []struct {
		name string
		end  func(stop func(), b *bridge.Bridge)
	}{
		{"exposure stopped", func(stop func(), _ *bridge.Bridge) { stop() }},
		{"bridge closed", func(_ func(), b *bridge.Bridge) { b.Close() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				store := nodra.New(state{})
				defer func() { _ = store.Close() }()
				b := bridge.New(bridge.Options{FrameInterval: time.Hour})
				defer b.Close()
				stop, err := bridge.ExposeProjection(b, "counter", store, project)
				if err != nil {
					t.Fatal(err)
				}
				f := newFrames()
				done := watch(context.Background(), b, "counter", 0, f)
				synctest.Wait()
				start := time.Now()
				increment(t, store)
				tc.end(stop, b)
				if err := <-done; !errors.Is(err, bridge.ErrClosed) {
					t.Fatalf("watch ended with %v, want ErrClosed", err)
				}
				if elapsed := time.Since(start); elapsed != 0 {
					t.Fatalf("the watch waited %v after the state went away", elapsed)
				}
				if got := counts(t, f); len(got) != 1 {
					t.Fatalf("frames %v, want only the first", got)
				}
			})
		})
	}
}
