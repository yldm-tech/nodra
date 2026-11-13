package core

import (
	"reflect"
	"sync"
)

// InvalidationTarget is the minimal redraw capability required by InvalidateTarget. MyGo Window satisfies it, as do test fakes and other native UI hosts. Implementations must make Invalidate safe to call from a Store subscriber goroutine.
type InvalidationTarget interface {
	Invalidate()
}

// InvalidateTarget requests a redraw from any InvalidationTarget whenever the source publishes. Nil and typed-nil targets are ignored. It accepts read-only and derived sources without exposing a concrete UI toolkit. The caller owns the target lifecycle and should call the returned stop function when the target is no longer valid.
func InvalidateTarget[T any](source Observable[T], target InvalidationTarget) func() {
	if !validObservable(source) || !validInvalidationTarget(target) {
		return func() {}
	}
	return source.SubscribeSnapshot(func(Snapshot[T]) { target.Invalidate() })
}

func validInvalidationTarget(target InvalidationTarget) bool {
	if target == nil {
		return false
	}
	value := reflect.ValueOf(target)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return !value.IsNil()
	default:
		return true
	}
}

// invalidationWindow is the native lifecycle boundary; state subscriptions need no running UI event loop.
type invalidationWindow interface {
	Invalidate()
	IsDestroyed() bool
	OnClosed(func()) func()
}

func bindInvalidation(window invalidationWindow, subscribe func(redraw func()) func()) func() {
	if window.IsDestroyed() {
		return func() {}
	}
	var mu sync.Mutex
	var stopped bool
	var cleanups []func()
	stop := func() {
		mu.Lock()
		if stopped {
			mu.Unlock()
			return
		}
		stopped = true
		pending := cleanups
		cleanups = nil
		mu.Unlock()
		for _, cleanup := range pending {
			cleanup()
		}
	}
	// Closing may race with either registration, including selector initialization. A registration completed after stop must be removed immediately.
	addCleanup := func(cleanup func()) {
		mu.Lock()
		if !stopped {
			cleanups = append(cleanups, cleanup)
			mu.Unlock()
			return
		}
		mu.Unlock()
		cleanup()
	}
	attached := false
	defer func() {
		if !attached {
			stop()
		}
	}()
	addCleanup(window.OnClosed(stop))
	// OnClosed does not replay a close that happened before registration.
	if window.IsDestroyed() {
		stop()
		return stop
	}
	addCleanup(subscribe(func() {
		mu.Lock()
		active := !stopped
		mu.Unlock()
		if active {
			window.Invalidate()
		}
	}))
	attached = true
	return stop
}
