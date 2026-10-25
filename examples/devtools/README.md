# DevTools counter

A console counter with a local browser inspector. The Store uses a stable definition ID and named actions; an intentionally rejected action demonstrates error capture without committing state. A timer increments the counter once per second by default.

Run from the repository root with Go 1.27.3 or later:

```sh
go run ./examples/devtools
```

Open the loopback session URL printed by the process to inspect live state, action errors and snapshots. The inspector is read-only by default. Stop the process with **Ctrl+C** to close the inspector and ticker; no state file is written.

## Options

```sh
go run ./examples/devtools -interval 250ms
go run ./examples/devtools -restore
```

`-interval` must be positive. `-restore` explicitly enables JSON editing and snapshot restoration for this demo Store. The timer continues running after an edit or restore, so the count will increase again on the next tick. Treat the printed URL as a credential for this local development session.

Read [main.go](main.go) and the [DevTools documentation](../../docs/devtools.md) for redaction, resource limits and remote inspection. For task timelines in an application, run [the workbench](../workbench/README.md) with `-devtools`. [All examples](../README.md).
