# Public API reference

The supported import path is `github.com/yldm-tech/nodra`. The root package re-exports the implementation types from `internal/core` so applications do not need repository layout knowledge. Some Go documentation renderers do not expand methods on generic type aliases; this page is the stable method index for the v1 review and links each group to the source documentation.

## Store

Create a store with `nodra.New`, `nodra.NewWithClone`, or `nodra.Create`. `Store[T]` provides:

| Group | Methods | Contract |
| --- | --- | --- |
| Read | `Get`, `State`, `Snapshot`, `Version`, `ReadOnly` | Snapshots carry a monotonic version. `NewWithClone` isolates owned mutable data. |
| Write | `Set`, `Update`, `UpdateErr`, `Transaction`, `Reset`, `Patch` | Failed mutation callbacks and panics do not publish a candidate. Mutable references follow the ownership policy below. `ErrClosed` rejects writes after close. |
| Group notifications | `Batch`, `BatchErr` | Successful writes remain committed when the batch callback fails; the outermost batch publishes once. Use transactions for rollback. |
| Observe | `Subscribe`, `SubscribeWith`, `SubscribeSnapshot`, `SubscribeMutations`, `Watch` | Callbacks run after the store lock is released. Unsubscribe functions are idempotent. |
| Actions | `Do`, `Action`, `SubscribeActions`, `SubscribeActionHooks`, `SubscribeActionTrace`, `TaskSnapshot`, `ActiveTasks` | Action metadata is published after the mutation boundary. |
| Lifecycle | `ID`, `Done`, `Close` | Close removes subscribers and closes active watch streams. The final value remains readable; repeated Close returns `ErrClosed`. |

`New` and `Create` copy values shallowly. Treat referenced data as immutable or copy it before writing; rollback cannot undo an in-place mutation through a shared map, slice, or pointer. Use `NewWithClone` with a complete clone function for isolated mutable candidates, snapshots, reset values, and callback payloads.

Use `nodra.Atomic2`, `nodra.AtomicGroup`, and `nodra.AtomicUpdate` when several stores must become visible together. Do not call store methods from an atomic candidate callback; mutate only the supplied candidate pointers. Callback errors and preparation panics leave every participant unchanged; preparation panics return `PanicError`. After commit, a synchronous notification panic propagates to the caller without rolling back any participant.

## Read-only and derived values

`Store.ReadOnly()` returns `ReadOnly[T]`, which exposes observation without write or close capability. `Reader[T]` is the sealed observation surface accepted by projection and DevTools APIs; applications cannot implement it. `Observable[T]` is the smaller versioned surface accepted by `Derive`, `Combine`, and invalidation helpers and can be implemented by applications.

Use `Derive`, `DeriveProjection`, `Combine`, `Select`, `SubscribeSelector(With)`, and `SubscribeProjection(With)` for computed values. Equality callbacks must be pure and safe for concurrent calls. Call `Close` on derived values when their owner is done.

## History and persistence

`Track` and `TrackWith` provide undo/redo. `History.Undo`, `Redo`, `CanUndo`, `CanRedo`, `BeginGroup`, `Status`, and `Close` are the supported lifecycle.

`NewPersistence(WithOptions)` separates state from bytes. `JSONCodec`, `VersionedJSONCodec`, `AESGCMCodec`, `FileStorage`, and `FileStorage.WithConflictDetection` compose the built-in persistence stack; conflict detection returns a `CheckedFileStorage`. `Persistence.Load`, `LoadBackup`, `LoadWithRecovery`, and `Save` are explicit operations. `AutoSave`, `AutoSaveWithErrors`, and `AutoSaveAsync` have distinct shutdown and error behavior. `AutoSaveAsync` returns an `AsyncPersistence` whose `Flush`, `Close`, `Errors`, and `Status` expose durability and shutdown results. Callers that need a durable result must handle `Flush` or `Close` errors.

## Async actions and ownership

`NewAsyncAction(WithOptions)` creates cancelable latest-result or queued work. `AsyncAction.Run` returns a `Task`; use `Task.Wait` to await that run and `Task.Cancel` to request cancellation. `Task.Done` and `Task.Status` expose completion and progress. `AsyncAction.Close` rejects new runs, cancels accepted work, and waits within its context; it does not flush successful results. `AsyncAction.Status`, `Metrics`, `ExportMetrics`, and `SubscribeMetrics` expose action state and aggregate outcomes.

`Scope.Context`, `Done`, `Observe`, `Defer`, and `Close` own subscriptions, persistence writers, and actions. Register stores before persistence and actions so reverse-order shutdown stops work before draining persistence and closing stores. Close with a fresh deadline during shutdown. `PanicError` identifies asynchronous callback failures, atomic preparation failures, and cleanup failures; ordinary synchronous notification panics propagate unchanged.

## Compatibility rule

The v1 contract is defined by exported root-package documentation, tests in the external `nodra_test` package, and [compatibility.md](compatibility.md). Internal packages, DevTools HTML, error text, and performance measurements are not public compatibility surfaces unless explicitly listed there.
