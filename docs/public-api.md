# Public API reference

The supported import path is `github.com/yldm-tech/nodra`. The root package re-exports the implementation types from `internal/core` so applications do not need repository layout knowledge. Go doc and pkg.go.dev do not list the methods of a type alias, so the root package shows none of the methods of the types it re-exports; this page is the stable method index for the v1 review. Each method list starts at a qualified name such as `History.Undo` and names the methods of that type only.

## Store

Create a store with `nodra.New`, `nodra.NewWithClone`, or `nodra.Create`. `Store[T]` provides:

| Group | Methods | Contract |
| --- | --- | --- |
| Read | `Store.Get`, `State`, `Snapshot`, `Version`, `ReadOnly` | Snapshots carry a monotonic version. `NewWithClone` isolates owned mutable data. |
| Write | `Store.Set`, `Update`, `UpdateErr`, `Transaction`, `Reset`, `Patch` | Failed mutation callbacks and panics do not publish a candidate. Callbacks run under the store's lock: they may read other stores but must not write to any store or call methods of their own store. Mutable references follow the ownership policy below. `ErrClosed` rejects writes after close. |
| Group notifications | `Store.Batch`, `BatchErr` | Successful writes remain committed when the batch callback fails; the outermost batch publishes once. An `Atomic2` or `AtomicGroup` commit into the store during the batch publishes the earlier writes with the transaction, and the batch then publishes only later ones. Use transactions for rollback. |
| Observe | `Store.Subscribe`, `SubscribeWith`, `SubscribeSnapshot`, `SubscribeMutations`, `Watch` | Callbacks run after the store lock is released. Unsubscribe functions are idempotent. |
| Actions | `Store.Do`, `Action`, `SubscribeActions`, `SubscribeActionHooks`, `SubscribeActionTrace`, `TaskSnapshot`, `ActiveTasks` | Action metadata is published after the mutation boundary. |
| Lifecycle | `Store.ID`, `Done`, `Close` | Close removes subscribers and closes active watch streams. The final value remains readable; repeated Close returns `ErrClosed`. |

`New` and `Create` copy values shallowly. Treat referenced data as immutable or copy it before writing; rollback cannot undo an in-place mutation through a shared map, slice, or pointer. Use `NewWithClone` with a complete clone function for isolated mutable candidates, snapshots, reset values, and callback payloads.

Use `nodra.Atomic2`, `nodra.AtomicGroup`, and `nodra.AtomicUpdate` when several stores must become visible together. Do not call store methods from an atomic candidate callback; mutate only the supplied candidate pointers. Callback errors and preparation panics leave every participant unchanged; preparation panics return `PanicError`. After commit, a synchronous notification panic propagates to the caller without rolling back any participant.

## Factories and field bindings

`nodra.Define(id, initial)` and `nodra.DefineWithClone(id, initial, clone)` return a `Definition[T]`, a reusable named store factory. `Definition.New` runs `initial` and returns a store with independent state whose `ID` is the definition's id, which `Definition.ID` also returns; a nil `initial` yields the zero value. `Define` stores copy values shallowly like `New`, so `initial` must return fresh maps, slices and pointers for the instances to stay independent. `DefineWithClone` stores follow the `NewWithClone` isolation policy.

`nodra.BindField(store, read, write)` returns a `Field[T, V]` that edits one field of shared state, or `ErrInvalidBinding` when an argument is nil. `Field.Get` selects the field from the current store value. `Field.Set` applies `write` to the latest store candidate, so unrelated concurrent changes survive; an error from `write` leaves the store unchanged. `Field.Edit` runs an editor on a local copy outside Store locks, so it can render MyGo controls, and commits only when the editor reports a change. `read` and `write` follow the store's ownership policy. Keep form drafts separate when edits need an explicit Save or Cancel.

## Read-only and derived values

`Store.ReadOnly()` returns `ReadOnly[T]`, which exposes observation without write or close capability. `ReadOnly.Get`, `State`, `Snapshot`, `Version`, `ID`, `Done`, `Subscribe`, `SubscribeWith`, `SubscribeSnapshot`, `SubscribeMutations`, `SubscribeActions`, `SubscribeActionTrace`, `TaskSnapshot`, `ActiveTasks`, and `Watch` behave like the `Store` methods of the same name; the zero `ReadOnly` is an empty, invalid source whose `Done` returns nil. `Reader[T]` is the sealed observation surface accepted by projection and DevTools APIs; applications cannot implement it. `Observable[T]` is the smaller versioned surface accepted by `Derive`, `Combine`, and invalidation helpers and can be implemented by applications. Transactions settle derived values in dependency order only for Stores, read-only views and Nodra derived values: an application `Observable` counts as a source of height zero, so a `Combine` over one that forwards a derived value may see a half-applied `Atomic2` or `AtomicGroup`. Over Stores, read-only views and Nodra derived values, `Combine` projects and publishes only input pairs that some committed state contained, also while concurrent transactions write the same stores and while it is being constructed; an application `Observable` cannot report whether it has caught up with the Stores behind it, so a pair that includes one is checked only against the other input.

Use `Derive`, `DeriveProjection`, `Combine`, `Select`, `SubscribeSelector(With)`, and `SubscribeProjection(With)` for computed values. `Derived[T, V]` provides `Derived.Get`, `Snapshot`, `Subscribe`, `SubscribeSnapshot`, `Done`, and `Close`. Equality callbacks must be pure and safe for concurrent calls. A derived value with a nil equals function notifies on every accepted source version; pass one to notify only on changes. Call `Close` on derived values when their owner is done; `Derived.Done` is closed when it is.

## History and persistence

`Track` and `TrackWith` provide undo/redo. `History.Undo`, `Redo`, `CanUndo`, `CanRedo`, `BeginGroup`, `Clear`, `Status`, and `Close` are the supported lifecycle. `Clear` drops both stacks and makes the current value the baseline, for a document replaced by Open, New or Revert; it returns `ErrHistoryGroup` while a group is open, `ErrHistoryBatch` inside an open `Batch` and `ErrClosed` after `Close`.

`NewPersistence(WithOptions)` separates state from bytes. `JSONCodec`, `VersionedJSONCodec`, `AESGCMCodec`, `FileStorage`, and `FileStorage.WithConflictDetection` compose the built-in persistence stack; conflict detection returns a `CheckedFileStorage`. The codecs implement `Codec` through `JSONCodec.Encode` and `Decode`, `VersionedJSONCodec.Encode` and `Decode`, and `AESGCMCodec.Encode` and `Decode`. The storages implement `Storage` through `FileStorage.Load`, `LoadBackup`, and `Save` and `CheckedFileStorage.Load`, `LoadBackup`, and `Save`; `LoadBackup` returns `ErrNoBackup` unless `FileStorage.Backup` is set, and a checked `Save` over content another writer changed since the last `Load` returns a `FileConflictError` matching `ErrFileConflict` and leaves the files untouched. `AESGCMCodec` takes its keys from a `KeyProvider`; `NewMemoryKeyProvider` is an in-memory provider for tests and local applications. `MemoryKeyProvider.Add`, `SetCurrent`, `Current`, and `Get` add a key under an id, make an added id the key for new writes, and read the current key or a key by id; empty or unknown ids and invalid AES keys return `ErrInvalidKey`. Rotate by adding the new key before making it current, and keep the old keys so earlier data stays readable. `AESGCMCodec.AssociatedData` binds ciphertext to a store identity such as its file path or application name, under every key: data opens only with the value it was sealed with, the empty value keeps the original format, and adding or changing it for existing data needs a migration that loads with the old value and saves with the new one. `Persistence.Load`, `LoadBackup`, `LoadWithRecovery`, and `Save` are explicit operations. `Persistence.AutoSave`, `AutoSaveWithErrors`, and `AutoSaveAsync` have distinct shutdown and error behavior. `AutoSaveAsync` returns an `AsyncPersistence`; `AsyncPersistence.Flush`, `Close`, `Errors`, and `Status` expose durability and shutdown results. Callers that need a durable result must handle `Flush` or `Close` errors.

## Async actions and ownership

`NewAsyncAction(WithOptions)` creates cancelable latest-result or queued work. `AsyncAction.Run` returns a `Task`; use `Task.Wait` to await that run and `Task.Cancel` to request cancellation. `Task.Done` and `Task.Status` expose completion and progress. `AsyncAction.Close` rejects new runs, cancels accepted work, and waits within its context; it does not flush successful results. `AsyncAction.Status`, `Metrics`, `ExportMetrics`, and `SubscribeMetrics` expose action state and aggregate outcomes.

`NewScope` creates a `Scope`. `Scope.Context`, `Done`, `Observe`, `Defer`, and `Close` own subscriptions, persistence writers, and actions. Register stores before persistence and actions so reverse-order shutdown stops work before draining persistence and closing stores. Close with a fresh deadline during shutdown. `PanicError` identifies asynchronous callback failures, atomic preparation failures, and cleanup failures; ordinary synchronous notification panics propagate unchanged.

## MyGo invalidation

`Invalidate`, `InvalidateSelector(With)`, and `InvalidateProjection(With)` request a frame of a MyGo window when a source publishes, when a selected value changes, or when an owned projection changes. The subscription stops when the window closes; the returned stop function is idempotent and does not wait for callbacks already running. Nil inputs and destroyed windows subscribe to nothing. `InvalidateTarget` binds the same redraw to any `InvalidationTarget`; the caller owns the target lifecycle and calls the returned stop function.

## Compatibility rule

The v1 contract is defined by exported root-package documentation, tests in the external `nodra_test` package, and [compatibility.md](compatibility.md). Internal packages, DevTools HTML, error text, and performance measurements are not public compatibility surfaces unless explicitly listed there.
