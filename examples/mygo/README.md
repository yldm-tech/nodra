# MyGo counter

A native counter and a separate preferences window share one Store. `InvalidateSelector` redraws each window only when its selected field changes, and `BindField.Edit` commits the checkbox against the latest state so a concurrent count update is preserved.

Run from the repository root with Go 1.27.3 or later and a graphical desktop session:

```sh
go run ./examples/mygo
```

Use **+**, **−** and **Reset**, then open **Preferences…** and toggle **Increment by two**. **Increment in background** demonstrates a goroutine updating shared state. Closing the preferences window releases its redraw subscription; reopening it creates a new window using the same Store.

```sh
go test ./examples/mygo
```

The tests exercise the counter and preference controls with MyGo's UI tester. Read [main.go](main.go) for the integration, [the workbench](../workbench/README.md) for tasks and persistence, and [the support matrix](../../docs/support.md) for native runtime requirements. [All examples](../README.md).
