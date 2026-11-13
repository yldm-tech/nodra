// Package nodra is a concurrency-safe reactive state store for Go desktop applications.
//
// The root package is the stable public API; implementation details are kept in internal/core. The public method index and ownership/error contracts are in docs/public-api.md, and the v1 support matrix is in docs/support.md. Go doc does not list the methods of a type alias, so the root package shows none of the methods of the types it re-exports from internal/core. This package comment therefore keeps the method index of every re-exported type here too; the error types add only Error and Unwrap.
//
// # Stores
//
// Start with New, Create or NewWithClone. New and Create copy values shallowly; NewWithClone isolates mutable references with a clone function. Store provides Get, State, Snapshot, Version, ReadOnly, Set, Update, UpdateErr, Patch, Reset, Transaction, Batch, BatchErr, Watch, Subscribe, SubscribeWith, SubscribeSnapshot, SubscribeMutations, Do, Action, SubscribeActions, SubscribeActionHooks, SubscribeActionTrace, TaskSnapshot, ActiveTasks, ID, Done and Close. ReadOnly exposes the observation-only surface: ReadOnly provides Get, State, Snapshot, Version, ID, Done, Subscribe, SubscribeWith, SubscribeSnapshot, SubscribeMutations, SubscribeActions, SubscribeActionTrace, TaskSnapshot, ActiveTasks and Watch.
//
// Define and DefineWithClone create named store factories: Definition.New returns a store with independent state and Definition.ID returns its id. BindField binds an editor to one field through Field.Get, Field.Set and Field.Edit. Atomic2, AtomicGroup and AtomicUpdate commit several stores as one visibility boundary.
//
// # Derived values
//
// Derive, DeriveProjection and Combine return memoized Derived values, and Derived provides Get, Snapshot, Subscribe, SubscribeSnapshot, Done and Close; Done is closed when the value is closed. Select, SubscribeSelector, SubscribeSelectorWith, SubscribeProjection and SubscribeProjectionWith read or observe projections.
//
// # History
//
// Track and TrackWith record undo and redo. History provides Undo, Redo, CanUndo, CanRedo, BeginGroup, Status and Close. CanUndo and CanRedo are false while a group is open, and closing the tracked Store does not close its History.
//
// # Persistence
//
// NewPersistence and NewPersistenceWithOptions combine a Storage such as FileStorage or CheckedFileStorage with a Codec such as JSONCodec, VersionedJSONCodec or AESGCMCodec. FileStorage provides Load, LoadBackup, Save and WithConflictDetection, which returns a CheckedFileStorage; CheckedFileStorage provides Load, LoadBackup and Save and rejects a save over content another writer changed. The codecs provide JSONCodec.Encode and Decode, VersionedJSONCodec.Encode and Decode, and AESGCMCodec.Encode and Decode. NewMemoryKeyProvider supplies AESGCMCodec keys for tests and local applications, and MemoryKeyProvider provides Add, SetCurrent, Current and Get to add keys, rotate the key for new writes and read keys. Persistence provides Load, LoadBackup, LoadWithRecovery, Save, AutoSave, AutoSaveWithErrors and AutoSaveAsync. AutoSaveAsync returns an AsyncPersistence, and AsyncPersistence provides Flush, Errors, Status and Close.
//
// # Async work
//
// NewAsyncAction and NewAsyncActionWithOptions create an AsyncAction, and AsyncAction provides Run, Status, Metrics, ExportMetrics, SubscribeMetrics and Close. Run returns a Task, and Task provides Wait, Cancel, Done and Status. NewScope creates a Scope that owns subscriptions, persistence writers and actions, and Scope provides Context, Observe, Defer, Done and Close.
//
// # MyGo
//
// MyGo redraws use Invalidate, InvalidateTarget, InvalidateSelector, InvalidateSelectorWith, InvalidateProjection and InvalidateProjectionWith. Programs that never use a MyGo window can build with the nodra_nomygo tag, which leaves out the helpers that take a *mygo.Window so MyGo is not linked; InvalidateTarget stays available.
package nodra
