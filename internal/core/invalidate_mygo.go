//go:build !nodra_nomygo

package core

import "github.com/egoist/mygo"

// These helpers take a *mygo.Window, so they are the only part of the package that imports MyGo. Building with the nodra_nomygo tag leaves them out; InvalidateTarget and bindInvalidation stay available.

// InvalidateProjection requests frames only when an owned projection changes, without cloning the entire source. The projector and callbacks follow SubscribeProjection's ownership rules. The window owns the subscription lifetime.
func InvalidateProjection[T, V any](source Reader[T], window *mygo.Window, project func(T) V) func() {
	return InvalidateProjectionWith(source, window, project, SelectorOptions[V]{})
}

func InvalidateProjectionWith[T, V any](source Reader[T], window *mygo.Window, project func(T) V, options SelectorOptions[V]) func() {
	store := sourceStore(source)
	if store == nil || window == nil || project == nil {
		return func() {}
	}
	return bindInvalidation(window, func(redraw func()) func() {
		return SubscribeProjectionWith(store, project, func(_, _ V) { redraw() }, options)
	})
}

// Invalidate requests a new frame of window's native Content whenever store publishes a snapshot. The subscription stops when the window closes; the returned function stops it earlier and is safe to call repeatedly. Nil inputs and destroyed windows subscribe to nothing. A callback already running when stopped may still request a frame.
func Invalidate[T any](source Observable[T], window *mygo.Window) (stop func()) {
	if !validObservable(source) || window == nil {
		return func() {}
	}
	return bindInvalidation(window, func(redraw func()) func() {
		return source.SubscribeSnapshot(func(Snapshot[T]) { redraw() })
	})
}

// InvalidateSelector requests a frame only when the selected value changes, using SubscribeSelector's equality rules. The subscription stops when the window closes or the returned function is called. Selectors run on writer goroutines and must be concurrency-safe; selected reference values follow the store's ownership policy.
func InvalidateSelector[T any, V any](source Observable[T], window *mygo.Window, selector func(T) V) (stop func()) {
	return InvalidateSelectorWith(source, window, selector, SelectorOptions[V]{})
}

// InvalidateSelectorWith is InvalidateSelector with custom equality and optional immediate invalidation. Nil inputs and destroyed windows subscribe to nothing. As with Invalidate, stopping does not wait for callbacks already running.
func InvalidateSelectorWith[T any, V any](source Observable[T], window *mygo.Window, selector func(T) V, options SelectorOptions[V]) (stop func()) {
	if !validObservable(source) || window == nil || selector == nil {
		return func() {}
	}
	return bindInvalidation(window, func(redraw func()) func() {
		return SubscribeSelectorWith(source, selector, func(_, _ V) { redraw() }, options)
	})
}
