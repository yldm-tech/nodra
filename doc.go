// Package nodra is a concurrency-safe reactive state store for Go desktop applications.
//
// The root package is the stable public API; implementation details are kept in internal/core. Start with New, Create or NewWithClone, then use Store.Get, Set, Update, UpdateErr, Patch, Reset, Transaction, Batch, BatchErr, Watch, Subscribe, SubscribeWith, SubscribeSnapshot, SubscribeMutations, Do, Action, TaskSnapshot, ActiveTasks and Close. ReadOnly exposes the observation-only surface. Derived values use Derive, DeriveProjection, Combine, Select, SubscribeSelector and SubscribeProjection. Persistence uses NewPersistence, JSONCodec, VersionedJSONCodec, FileStorage, CheckedFileStorage and AESGCMCodec. Async work uses NewAsyncAction, Task and Scope. MyGo redraws use Invalidate, InvalidateTarget, InvalidateSelector and InvalidateProjection.
//
// The public method index and ownership/error contracts are in docs/public-api.md. Go's current generic type-alias documentation omits methods from the Store, History, Persistence and AsyncAction entries, so this package comment intentionally keeps the discoverable method index here too. The v1 support matrix is in docs/support.md.
package nodra
