// Package nodra is a concurrency-safe reactive state store for Go desktop applications.
//
// The root package is the stable public API; implementation details are kept in internal/core. The public method index and ownership/error contracts are in docs/public-api.md, and the v1 support matrix is in docs/support.md. Go's current generic type-alias documentation omits methods from the Store, History, Persistence, AsyncAction and other aliased entries, so this package comment intentionally keeps the discoverable method index here too.
//
// # Stores
//
// Start with New, Create or NewWithClone. New and Create copy values shallowly; NewWithClone isolates mutable references with a clone function. Store provides Get, State, Snapshot, Version, ReadOnly, Set, Update, UpdateErr, Patch, Reset, Transaction, Batch, BatchErr, Watch, Subscribe, SubscribeWith, SubscribeSnapshot, SubscribeMutations, Do, Action, SubscribeActions, SubscribeActionHooks, SubscribeActionTrace, TaskSnapshot, ActiveTasks, ID, Done and Close. ReadOnly exposes the observation-only surface.
//
// Define and DefineWithClone create named store factories: Definition.New returns a store with independent state and Definition.ID returns its id. BindField binds an editor to one field through Field.Get, Field.Set and Field.Edit. Atomic2, AtomicGroup and AtomicUpdate commit several stores as one visibility boundary.
//
// # Derived values
//
// Derive, DeriveProjection and Combine return memoized Derived values, whose Done channel is closed when they are closed. Select, SubscribeSelector, SubscribeSelectorWith, SubscribeProjection and SubscribeProjectionWith read or observe projections.
//
// # History
//
// Track and TrackWith record undo and redo. History provides Undo, Redo, CanUndo, CanRedo, BeginGroup, Status and Close. CanUndo and CanRedo are false while a group is open, and closing the tracked Store does not close its History.
//
// # Persistence
//
// NewPersistence and NewPersistenceWithOptions combine a Storage such as FileStorage or CheckedFileStorage with a Codec such as JSONCodec, VersionedJSONCodec or AESGCMCodec; NewMemoryKeyProvider supplies AESGCMCodec keys for tests and local applications. Persistence provides Load, LoadBackup, LoadWithRecovery, Save, AutoSave, AutoSaveWithErrors and AutoSaveAsync. The AsyncPersistence returned by AutoSaveAsync provides Flush, Errors, Status and Close.
//
// # Async work
//
// NewAsyncAction and NewAsyncActionWithOptions create an AsyncAction, which provides Run, Status, Metrics, ExportMetrics, SubscribeMetrics and Close. Run returns a Task, which provides Wait, Cancel, Done and Status. NewScope creates a Scope whose Context, Observe, Defer, Done and Close own subscriptions, persistence writers and actions.
//
// # MyGo
//
// MyGo redraws use Invalidate, InvalidateTarget, InvalidateSelector, InvalidateSelectorWith, InvalidateProjection and InvalidateProjectionWith. Programs that never use a MyGo window can build with the nodra_nomygo tag, which leaves out the helpers that take a *mygo.Window so MyGo is not linked; InvalidateTarget stays available.
package nodra
