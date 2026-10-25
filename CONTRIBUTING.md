# Contributing

Changes should preserve the root import path and the contracts in
[docs/compatibility.md](docs/compatibility.md). Public behavior belongs in
the root package; implementation-only helpers stay under `internal/`.

Before opening a pull request, run:

```sh
go test ./...
go test -race ./...
go vet ./...
go mod tidy -diff
```

For API changes, run `scripts/check-api.sh` against the pull request base and
include the exact incompatible report under `.github/api-changes/` when the
change is intentional. Add or update an external-package test when changing
the root façade, and update the public API or support documentation when a
contract changes.

Use Conventional Commits. Keep commits focused, explain persistence or
concurrency trade-offs in the pull request, and do not commit generated
credentials, local state files, or DevTools exports containing sensitive data.
