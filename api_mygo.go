//go:build !nodra_nomygo

package nodra

import (
	"github.com/egoist/mygo"
	core "github.com/yldm-tech/nodra/internal/core"
)

// The *mygo.Window helpers are the only reason the root package imports MyGo. Programs that use only the core and DevTools can build with -tags nodra_nomygo to leave them out and drop MyGo from the binary; InvalidateTarget remains available.

// Invalidate requests a new frame of window's native Content whenever source publishes a snapshot. The request is made on the writing goroutine and Window.Invalidate posts it to MyGo's main thread. The subscription stops when the window closes; the returned function stops it earlier and is safe to call repeatedly. Nil inputs and destroyed windows subscribe to nothing. A callback already running when stopped may still request a frame.
func Invalidate[T any](source Observable[T], window *mygo.Window) func() {
	return core.Invalidate(source, window)
}

// InvalidateSelector is Invalidate that requests a frame only when the selected value changes, using SubscribeSelector's equality. selector runs on writer goroutines and must be safe for concurrent calls; selected reference values follow the store's ownership policy. The subscription stops when the window closes or the returned function is called.
func InvalidateSelector[T, V any](source Observable[T], window *mygo.Window, selector func(T) V) func() {
	return core.InvalidateSelector(source, window, selector)
}

// InvalidateSelectorWith is InvalidateSelector with custom equality and optional immediate invalidation from options. Nil inputs and destroyed windows subscribe to nothing. As with Invalidate, stopping does not wait for callbacks already running.
func InvalidateSelectorWith[T, V any](source Observable[T], window *mygo.Window, selector func(T) V, options SelectorOptions[V]) func() {
	return core.InvalidateSelectorWith(source, window, selector, options)
}

// InvalidateProjection is Invalidate that requests a frame only when an owned projection changes, without cloning the entire source. project follows SubscribeProjectionWith's ownership rules. The window owns the subscription lifetime.
func InvalidateProjection[T, V any](source Reader[T], window *mygo.Window, project func(T) V) func() {
	return core.InvalidateProjection(source, window, project)
}

// InvalidateProjectionWith is InvalidateProjection with custom equality and optional immediate invalidation from options. Nil inputs and destroyed windows subscribe to nothing. As with Invalidate, the subscription stops when the window closes, the returned function is idempotent, and stopping does not wait for callbacks already running.
func InvalidateProjectionWith[T, V any](source Reader[T], window *mygo.Window, project func(T) V, options SelectorOptions[V]) func() {
	return core.InvalidateProjectionWith(source, window, project, options)
}
