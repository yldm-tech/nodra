//go:build !nodra_nomygo

package nodra

import (
	"github.com/egoist/mygo"
	core "github.com/yldm-tech/nodra/internal/core"
)

// The *mygo.Window helpers are the only reason the root package imports MyGo. Programs that use only the core and DevTools can build with -tags nodra_nomygo to leave them out and drop MyGo from the binary; InvalidateTarget remains available.

func Invalidate[T any](source Observable[T], window *mygo.Window) func() {
	return core.Invalidate(source, window)
}

func InvalidateSelector[T, V any](source Observable[T], window *mygo.Window, selector func(T) V) func() {
	return core.InvalidateSelector(source, window, selector)
}

func InvalidateSelectorWith[T, V any](source Observable[T], window *mygo.Window, selector func(T) V, options SelectorOptions[V]) func() {
	return core.InvalidateSelectorWith(source, window, selector, options)
}

func InvalidateProjection[T, V any](source Reader[T], window *mygo.Window, project func(T) V) func() {
	return core.InvalidateProjection(source, window, project)
}

func InvalidateProjectionWith[T, V any](source Reader[T], window *mygo.Window, project func(T) V, options SelectorOptions[V]) func() {
	return core.InvalidateProjectionWith(source, window, project, options)
}
