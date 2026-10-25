# Persistent word counter

A console example combining `NewWithClone`, `VersionedJSONCodec`, a JSON migration, `FileStorage` backups and `AutoSaveAsync`. Each normal run increments `hello`, flushes the current version and drains the writer before closing the Store.

Run from the repository root with Go 1.27.3 or later. Choose a disposable destination outside the checkout; omitting `-file` writes `state.json` in the current directory.

```sh
go run ./examples/persistent -file /path/to/state.json
go run ./examples/persistent -file /path/to/state.json
```

The first successful run prints `hello=1`; the next prints `hello=2`. The primary file and its `.bak` copy contain a schema version 1 envelope. An existing unversioned word map, such as `{"hello":3}`, is migrated into the `words` field and saved with `hello=4`. A malformed primary aborts normal startup without replacing it.

## Inspect a backup

After a successful normal run has created the backup:

```sh
go run ./examples/persistent -file /path/to/state.json -recover-backup
```

This explicitly validates and loads `state.json.bak` into memory and prints the number of words. It does not increment the counter, repair the primary, or rewrite either file. A missing or invalid backup returns an error.

```sh
go test ./examples/persistent
```

Read [main.go](main.go), the [persistence contracts](../../docs/architecture.md) and [the workbench](../workbench/README.md) for a complete application. [All examples](../README.md).
