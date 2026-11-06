// Package nativeui connects Nodra to MyGo's native UI beyond redrawing: the Edit menu's Undo and Redo, and their shortcuts, act on a History. It is separate from the core so that apps without native UI, such as those whose windows show web pages through the bridge, do not link MyGo's ui package.
package nativeui

import (
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/yldm-tech/nodra"
)

// UndoShortcuts makes the window's Undo and Redo act on history, in the view being built. While history can undo, Cmd+Z (Ctrl+Z on Linux and Windows) undoes; while it can redo, Shift+Cmd+Z redoes, and Ctrl+Y too on Windows, where the Edit menu shows it. Call it at the top level of the window's view, outside the builders of child elements, so the shortcuts serve the whole window.
//
// A focused text input keeps its own undo: the keys and the Edit menu act on its text, and on history only while no text input has the focus. With MyGo 0.7.4 on macOS, the Edit menu enables Undo and Redo only while they would act, so they follow history as the view is rebuilt; InvalidateHistory rebuilds it when history's state changes. On Linux and Windows the keys reach the shortcuts first; the menu's items act on text inputs.
//
// The shortcuts run on the main thread, as Undo and Redo then publish: a store subscriber that does slow work should move it to another goroutine. An Undo or Redo that history refuses, because a group began after the frame was built, does nothing.
func UndoShortcuts[T any](c *ui.Context, history *nodra.History[T]) {
	if c == nil || history == nil {
		return
	}
	status := history.Status().Get()
	if status.CanUndo {
		c.OnShortcut(ui.Cmd, ui.KeyZ, func() { _ = history.Undo() })
	}
	if status.CanRedo {
		redo := func() { _ = history.Redo() }
		c.OnShortcut(ui.Cmd|ui.Shift, ui.KeyZ, redo)
		if runtime.GOOS == "windows" {
			c.OnShortcut(ui.Ctrl, ui.KeyY, redo)
		}
	}
}

// InvalidateHistory redraws window when history starts or stops being able to undo or redo, so a view that calls UndoShortcuts registers the shortcuts history can serve and the Edit menu follows. Other changes of history's state draw nothing. The subscription stops when the window closes or the returned function is called; it is safe to call more than once.
func InvalidateHistory[T any](history *nodra.History[T], window *mygo.Window) (stop func()) {
	if history == nil {
		return func() {}
	}
	return nodra.InvalidateSelector(history.Status(), window, func(s nodra.HistoryState) [2]bool {
		return [2]bool{s.CanUndo, s.CanRedo}
	})
}
