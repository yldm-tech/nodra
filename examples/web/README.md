# Native window and web page

A window of native UI and a window that shows a web page follow one Store. Either one's **+1** and **Reset** change the count, and the other shows the change at once. The page sees the count through a projection that leaves the token in Go, and changes it only through the `counter/add` and `counter/reset` actions Go registers with the bridge.

Run from the repository root with Go 1.27.3 or later and a graphical desktop session:

```sh
go run ./examples/web
```

The page in [page/index.html](page/index.html) has no build step: Go serves it and the bridge's JavaScript client, `bridge.Client`, under `/nodra/`. Read [main.go](main.go) for the Go side and [docs/bridge.md](../../docs/bridge.md) for the bridge. [All examples](../README.md).
