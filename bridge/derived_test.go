package bridge_test

import (
	"context"
	"errors"
	"testing"

	"github.com/yldm-tech/nodra"
	"github.com/yldm-tech/nodra/bridge"
)

func TestAWatchOfADerivedValueEndsWhenItCloses(t *testing.T) {
	store := nodra.New(state{Count: 1})
	defer func() { _ = store.Close() }()
	derived := nodra.Derive(store, project, nil)
	b := bridge.New(bridge.Options{})
	defer b.Close()
	if _, err := bridge.Expose[pageState](b, "derived", derived); err != nil {
		t.Fatal(err)
	}
	f := newFrames()
	done := watch(context.Background(), b, "derived", 0, f)
	f.wait(t, 1)
	if err := derived.Close(); err != nil {
		t.Fatal(err)
	}
	if err := result(t, done); !errors.Is(err, bridge.ErrClosed) {
		t.Fatalf("watch returned %v, want ErrClosed", err)
	}
}

func TestAWatchOfAClosedDerivedValueEndsAtOnce(t *testing.T) {
	store := nodra.New(state{Count: 1})
	defer func() { _ = store.Close() }()
	derived := nodra.Derive(store, project, nil)
	if err := derived.Close(); err != nil {
		t.Fatal(err)
	}
	b := bridge.New(bridge.Options{})
	defer b.Close()
	if _, err := bridge.Expose[pageState](b, "derived", derived); err != nil {
		t.Fatal(err)
	}
	f := newFrames()
	done := watch(context.Background(), b, "derived", 0, f)
	if err := result(t, done); !errors.Is(err, bridge.ErrClosed) {
		t.Fatalf("watch returned %v, want ErrClosed", err)
	}
}
