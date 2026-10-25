# Basic Store

A console example of `New`, `Subscribe`, `Update` and `Batch`. No window opens and no file is written.

Run from the repository root with Go 1.27.3 or later:

```sh
go run ./examples/basic
```

The subscriber prints two snapshots:

```text
version=1 state={Count:1 Filter:all}
version=2 state={Count:2 Filter:active}
```

`Batch` coalesces its two updates into one publication, advancing the version once when the outer batch ends. For transactional rollback and atomic changes across Stores, see the [transaction contracts](../../docs/architecture.md), rather than treating notification batching as a transaction.

Read [main.go](main.go), then try [the MyGo example](../mygo/README.md) for native views or [the complete workbench](../workbench/README.md). [All examples](../README.md).
