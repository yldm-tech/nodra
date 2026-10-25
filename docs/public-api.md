# Public API reference

The supported import path is `github.com/yldm-tech/nodra`. The root package re-exports the implementation types from `internal/core` so applications do not need repository layout knowledge. Some Go documentation renderers do not expand methods on generic type aliases; this page is the stable method index for the v1 review and links each group to the source documentation.

## Store

Create a store with `nodra.New`, `nodra.NewWithClone`, or `nodra.Create`. `Store[T]` provides:

| Group | Methods | Contract |
| --- | --- | --- |
| Read | `Get`, `State`, `Snapshot`, `Version`, `ReadOnly` | Snapshots carry a monotonic version. `NewWithClone` isolates owned mutable data. |
| Write | `Set`, `Update`, `UpdateErr`, `Transaction`, `Reset`, `Patch` | Failed callbacks and panics do not publish a partial candidate. `ErrClosed` rejects writes after close. |
| Observe | `Subscribe`, `SubscribeWith`, `SubscribeSnapshot`, `SubscribeMutations`, `Watch` | Callbacks run after the store lock is released. Unsubscribe functions are idempotent. |
| Actions | `Do`, `Action`, `SubscribeActions`, `SubscribeActionHooks`, `SubscribeActionTrace`, `TaskSnapshot`, `ActiveTasks` | Action metadata is published after the mutation boundary. |
| Lifecycle | `Close` | Removes subscribers and closes active watch streams. Close is idempotent. |

Use `nodra.Atomic2`, `nodra.AtomicGroup`, and `nodra.AtomicUpdate` when several stores must become visible together. Do not call store methods from an atomic candidate callback; mutate only the supplied candidate pointers.

## Read-only and derived values

`Store.ReadOnly()` returns `ReadOnly[T]`, which exposes observation without write or close capability. `Reader[T]` is the observation surface accepted by projection and DevTools APIs. `Observable[T]` is the smaller versioned surface accepted by `Derive`, `Combine`, and invalidation helpers.

Use `Derive`, `DeriveProjection`, `Combine`, `Select`, `SubscribeSelector(With)`, and `SubscribeProjection(With)` for computed values. Equality callbacks must be pure and safe for concurrent calls. Call `Close` on derived values when their owner is done.

## History and persistence

`Track` and `TrackWith` provide undo/redo. `History.Undo`, `Redo`, `CanUndo`, `CanRedo`, `BeginGroup`, `Status`, and `Close` are the supported lifecycle.

`NewPersistence(WithOptions)` separates state from bytes. `JSONCodec`, `VersionedJSONCodec`, `AESGCMCodec`, `FileStorage`, and `CheckedFileStorage.WithConflictDetection` compose the built-in persistence stack. `Persistence.Load`, `LoadBackup`, `LoadWithRecovery`, and `Save` are explicit operations. `AutoSave`, `AutoSaveWithErrors`, and `AutoSaveAsync` have distinct shutdown and error behavior; callers that need a durable result must use `Flush` or `Close` and handle the returned error.

## Async actions and ownership

`NewAsyncAction(WithOptions)` creates cancelable latest-result or queued work. `AsyncAction.Run`, `Status`, `Flush`, and `Close` are safe public lifecycle operations. `PanicError` identifies panics at asynchronous user-code boundaries. `Scope` can own subscriptions, persistence writers, and actions; close it with a fresh deadline during shutdown.

## Compatibility rule

The v1 contract is defined by exported root-package documentation, tests in the external `nodra_test` package, and [compatibility.md](compatibility.md). Internal packages, DevTools HTML, error text, and performance measurements are not public compatibility surfaces unless explicitly listed there.
