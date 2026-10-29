# Examples

Start with [workbench](workbench/README.md) for a complete desktop application, or choose a smaller example to learn one feature.

| Example | What it demonstrates | Run from the repository root |
| --- | --- | --- |
| [basic](basic/README.md) | Store updates, versioned subscriptions and batched notifications | `go run ./examples/basic` |
| [mygo](mygo/README.md) | Two native windows, selective redraws and field bindings | `go run ./examples/mygo` |
| [persistent](persistent/README.md) | Cloned state, JSON migration, backup recovery and asynchronous persistence | `go run ./examples/persistent -file /path/to/state.json` |
| [devtools](devtools/README.md) | Local inspector, named actions and optional snapshot restoration | `go run ./examples/devtools` |
| [workbench](workbench/README.md) | Search, settings, localization, cancelable tasks and scoped shutdown | `go run ./examples/workbench` |

## Requirements

Use Go 1.27.3 or later. All examples belong to the root Go module and use its pinned dependencies, including MyGo v0.7.2; they do not require separate module initialization. The `mygo` and `workbench` examples open native windows and need a graphical desktop session. See the [supported platforms](../docs/support.md) for the tested runtime matrix.

Clone the repository and enter its root before running any command:

```sh
git clone https://github.com/yldm-tech/nodra.git
cd nodra
go run ./examples/basic
```

## Verification

```sh
go test ./examples/...
go vet ./examples/...
```

The focused tests cover MyGo interaction, persistence recovery and the workbench's search, validation and storage behavior. Passing them does not replace checking native windows in a graphical session. Refer to the [public API](../docs/public-api.md) and [architecture](../docs/architecture.md) for ownership and concurrency contracts beyond these examples.
