# Complete Nodra example

`workbench` is the complete runnable Nodra + MyGo example. It is a small workspace search desktop app rather than an isolated API snippet, and it is the recommended starting point for integrating Nodra into an application.

The example demonstrates a cloned settings Store, local form drafts with `BindField`-style commit behavior, validated JSON persistence with backups, asynchronous file-name search with debounce, cancellation and bounded results, asynchronous settings saves, `Scope` ownership and shutdown, MyGo invalidation, and optional authenticated DevTools inspection.

The settings panel includes English and Simplified Chinese. The selected language is stored with the validated settings, changes the application catalog immediately, and is restored on the next launch through MyGo's locale support.

Run it from the repository root:

```sh
go run ./examples/workbench
go run ./examples/workbench -devtools
go test ./examples/workbench
```

The default settings file is stored below the operating system user config directory. Use `-settings /path/to/settings.json` to select another destination. The app keeps unsaved form edits in the UI owner, validates the selected folder before committing, persists only successful settings, and cancels a search when its owner closes.

Use the example as a reference for ownership boundaries: shared business state belongs in Nodra Stores, temporary widget edits remain local to the view owner, work that can block belongs in `AsyncAction`, and every resource is registered with the same `Scope` that closes it.

The tests in `main_test.go` exercise real temporary directories, hidden-file filtering, cancellation, UI form validation, persistence reload, storage failures, and search results through MyGo's UI tester. The smaller examples in the sibling directories focus on one concept at a time; this workbench is the end-to-end project.
