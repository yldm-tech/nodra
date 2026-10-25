# Nodra

Nodra is a concurrency-safe reactive state store for Go desktop applications. It provides Zustand-like ergonomics for [MyGo](https://github.com/egoist/mygo) apps while keeping state independent from the renderer and persistence backend.

## Install

```sh
go get github.com/yldm-tech/nodra@latest
```

The core package is `github.com/yldm-tech/nodra`. The optional inspector is `github.com/yldm-tech/nodra/devtools`.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/yldm-tech/nodra"
)

type State struct{ Count int }

func main() {
	store := nodra.New(State{})
	defer store.Close()

	stop := store.Subscribe(func(s nodra.Snapshot[State]) {
		fmt.Println(s.Value.Count)
	})
	defer stop()

	_ = store.Update(func(s *State) { s.Count++ })
}
```

## Core capabilities

- Versioned `Store[T]` with synchronous subscriptions, bounded watches and read-only views.
- Derived state, selectors, projections, field bindings and atomic transactions.
- Undo/redo history with grouping, filtering and payload budgets.
- JSON, versioned migration, encrypted, backup, journaled and conflict-checked persistence.
- Debounced, cancelable and queued async actions with progress, retries and metrics.
- `Scope` for ownership and shutdown ordering.
- Optional MyGo invalidation helpers and a local authenticated DevTools inspector.

## MyGo

```go
stop := nodra.InvalidateSelector(store, window, func(s State) int {
	return s.Count
})
defer stop()
```

The integration targets MyGo `v0.6.3`. Keep temporary widget state in the view owner and commit only the edited field to the shared Store. See [`examples/mygo`](examples/mygo/main.go) for a minimal native window.

## Persistence

```go
persistence := nodra.NewPersistence(store, nodra.FileStorage{Path: "state.json", Backup: true}, nodra.JSONCodec[State]{})
if err := persistence.Load(ctx); err != nil {
	// Handle a missing or invalid startup file before starting autosave.
}
writer, _ := persistence.AutoSaveAsync(ctx, nodra.AsyncOptions{Debounce: 100 * time.Millisecond})
defer writer.Close(shutdownCtx)
```

Use `VersionedJSONCodec`, `AESGCMCodec`, `Journal`, `Durable` and `WithConflictDetection` when the application needs migrations, encryption or stronger file recovery guarantees. Details are in [`docs/architecture.md`](docs/architecture.md).

## Complete example

[`examples/workbench`](examples/workbench/README.md) is the full Nodra + MyGo desktop project. It combines cloned state, local drafts, validated backup persistence, debounced and cancelable async search, async saves, scoped shutdown, native invalidation, English/Simplified Chinese localization and optional DevTools.

```sh
go run ./examples/workbench
go run ./examples/workbench -devtools
```

The smaller examples cover one integration at a time:

- [`examples/basic`](examples/basic/main.go): Store, subscriptions and batches.
- [`examples/mygo`](examples/mygo/main.go): native windows and invalidation.
- [`examples/persistent`](examples/persistent/main.go): migrations, backups and write-behind persistence.
- [`examples/devtools`](examples/devtools/main.go): local inspector and restoration.

## DevTools

```go
inspector := devtools.New(devtools.Options{})
defer inspector.Close()
stop, err := devtools.Attach(inspector, "state", store.ReadOnly())
if err != nil {
	panic(err)
}
defer stop()
url, _ := inspector.Listen()
fmt.Println(url)
```

DevTools is read-only by default. `AllowRestore` explicitly enables typed JSON editing and snapshot restoration. See [`docs/devtools.md`](docs/devtools.md) for redaction, limits, remote read-only mode and the JSON compatibility contract.

## Documentation

- [`docs/public-api.md`](docs/public-api.md): root package API index.
- [`docs/architecture.md`](docs/architecture.md): ownership, concurrency and lifecycle semantics.
- [`docs/compatibility.md`](docs/compatibility.md): v0.x to v1 compatibility policy.
- [`docs/support.md`](docs/support.md): supported Go, MyGo, OS, architecture and filesystem matrix.
- [`CONTRIBUTING.md`](CONTRIBUTING.md): development and verification workflow.
- [`SECURITY.md`](SECURITY.md): vulnerability reporting and sensitive-data policy.

## Verification

```sh
go test ./...
go test -race ./...
go vet ./...
golangci-lint run ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Nodra is one Go module: `github.com/yldm-tech/nodra`. The project requires Go `1.27.3` or newer and uses MyGo `v0.6.3` for native integration.
