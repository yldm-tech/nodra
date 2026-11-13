package nodra

import (
	"context"

	core "github.com/yldm-tech/nodra/internal/core"
)

// The root package is the stable public entry point. Implementations live in internal/core so the repository can keep feature code together without changing the import path used by applications.

var (
	ErrNoUndo              = core.ErrNoUndo
	ErrNoRedo              = core.ErrNoRedo
	ErrSuperseded          = core.ErrSuperseded
	ErrInvalidAction       = core.ErrInvalidAction
	ErrQueueFull           = core.ErrQueueFull
	ErrUnsupportedVersion  = core.ErrUnsupportedVersion
	ErrMissingMigration    = core.ErrMissingMigration
	ErrClosed              = core.ErrClosed
	ErrFileConflict        = core.ErrFileConflict
	ErrHistoryBatch        = core.ErrHistoryBatch
	ErrHistoryGroup        = core.ErrHistoryGroup
	ErrInvalidBinding      = core.ErrInvalidBinding
	ErrInvalidHistory      = core.ErrInvalidHistory
	ErrInvalidKey          = core.ErrInvalidKey
	ErrInvalidPersistence  = core.ErrInvalidPersistence
	ErrNoBackup            = core.ErrNoBackup
	ErrPersistenceRecovery = core.ErrPersistenceRecovery
)

const (
	LatestWins ActionPolicy = core.LatestWins
	QueueAll   ActionPolicy = core.QueueAll

	TaskQueued     TaskPhase = core.TaskQueued
	TaskRunning    TaskPhase = core.TaskRunning
	TaskSucceeded  TaskPhase = core.TaskSucceeded
	TaskCanceled   TaskPhase = core.TaskCanceled
	TaskSuperseded TaskPhase = core.TaskSuperseded
	TaskFailed     TaskPhase = core.TaskFailed

	MutationDirect      MutationType = core.MutationDirect
	MutationPatch       MutationType = core.MutationPatch
	MutationAction      MutationType = core.MutationAction
	MutationReset       MutationType = core.MutationReset
	MutationTransaction MutationType = core.MutationTransaction
)

type (
	// AESGCMCodec encrypts another codec's bytes with authenticated AES-GCM.
	AESGCMCodec[T any]       = core.AESGCMCodec[T]
	ActionEvent              = core.ActionEvent
	ActionHooks              = core.ActionHooks
	ActionMetrics            = core.ActionMetrics
	ActionOptions[I any]     = core.ActionOptions[I]
	ActionPolicy             = core.ActionPolicy
	ActionTrace              = core.ActionTrace
	AsyncAction[T, I, R any] = core.AsyncAction[T, I, R]
	AsyncOptions             = core.AsyncOptions
	AsyncPersistence[T any]  = core.AsyncPersistence[T]
	AsyncState               = core.AsyncState
	AtomicOp                 = core.AtomicOp
	CheckedFileStorage       = core.CheckedFileStorage
	Codec[T any]             = core.Codec[T]
	Definition[T any]        = core.Definition[T]
	Derived[T, V any]        = core.Derived[T, V]
	Field[T, V any]          = core.Field[T, V]
	FileConflictError        = core.FileConflictError
	FileFingerprint          = core.FileFingerprint
	FileStorage              = core.FileStorage
	History[T any]           = core.History[T]
	HistoryOptions[T any]    = core.HistoryOptions[T]
	HistoryState             = core.HistoryState
	JSONCodec[T any]         = core.JSONCodec[T]
	JSONMigration            = core.JSONMigration
	KeyProvider              = core.KeyProvider
	LoadError                = core.LoadError
	MemoryKeyProvider        = core.MemoryKeyProvider
	MetricsSink              = core.MetricsSink
	Mutation[T any]          = core.Mutation[T]
	MutationType             = core.MutationType
	Observable[T any]        = core.Observable[T]
	Pair[A, B any]           = core.Pair[A, B]
	PanicError               = core.PanicError
	// Persistence connects a Store to a byte Storage and Codec.
	Persistence[T any]        = core.Persistence[T]
	PersistenceOptions[T any] = core.PersistenceOptions[T]
	PersistenceState          = core.PersistenceState
	ReadOnly[T any]           = core.ReadOnly[T]
	Reader[T any]             = core.Reader[T]
	RecoveryError             = core.RecoveryError
	RetryOptions              = core.RetryOptions
	Scope                     = core.Scope
	SelectorOptions[V any]    = core.SelectorOptions[V]
	Snapshot[T any]           = core.Snapshot[T]
	Storage                   = core.Storage
	// Store owns versioned state and synchronously publishes snapshots to its observers. See Store methods in docs/public-api.md when the Go documentation renderer omits methods for this root-package alias.
	Store[T any]              = core.Store[T]
	SubscribeOptions[T any]   = core.SubscribeOptions[T]
	Task                      = core.Task
	TaskMetrics               = core.TaskMetrics
	TaskPhase                 = core.TaskPhase
	TaskSnapshot              = core.TaskSnapshot
	VersionedJSONCodec[T any] = core.VersionedJSONCodec[T]
	InvalidationTarget        = core.InvalidationTarget
)

// New creates an open store at version zero that holds initial. New copies values shallowly: maps, slices and pointers reachable from initial, from values passed to Set and from values returned by Get, Snapshot and callbacks stay shared with the store. Do not mutate that data outside an update callback, copy it before writing inside one, and do not keep a mutable alias of a value passed to New or Set; rollback and Reset cannot undo an in-place mutation through a shared reference. Use NewWithClone for isolated mutable state.
func New[T any](initial T) *Store[T] { return core.New(initial) }

// Create is the Pinia/Zustand-style name for New and follows New's shallow ownership policy.
func Create[T any](initial T) *Store[T] { return core.Create(initial) }

// NewWithClone creates a store that isolates owned state, snapshots, subscriber values and rollback with clone. clone must copy all mutable references, be safe for concurrent calls, and must not reenter the store. A nil clone preserves New's behavior.
func NewWithClone[T any](initial T, clone func(T) T) *Store[T] {
	return core.NewWithClone(initial, clone)
}

// Atomic2 commits two different stores as one visibility boundary. Both candidates are prepared while both store locks are held; a callback error or panic leaves both stores unchanged, with preparation panics returned as PanicError. Readers and ordinary writers wait until both values are committed. Notifications run after the boundary is released, so listeners may safely reenter either store. A notification panic propagates unchanged without rolling back the committed stores. fn must not call Store methods; mutate only the provided pointers.
func Atomic2[A, B any](first *Store[A], second *Store[B], fn func(*A, *B) error) error {
	return core.Atomic2(first, second, fn)
}

// AtomicGroup commits any number of AtomicUpdate operations as one visibility boundary. Every callback is prepared while all participating store locks are held; an error or panic leaves every store unchanged, with preparation panics returned as PanicError, and a store that appears twice is rejected. Lock order is stable across calls, so groups built in opposite order cannot deadlock. Notifications run after all locks are released and may safely reenter their stores. A notification panic propagates unchanged without rolling back the committed stores.
func AtomicGroup(ops ...AtomicOp) error { return core.AtomicGroup(ops...) }

// AtomicUpdate prepares one store update for AtomicGroup. fn sees a private candidate and may return an error to abort the whole group. It must not call Store methods; mutate only the provided pointer.
func AtomicUpdate[T any](store *Store[T], fn func(*T) error) AtomicOp {
	return core.AtomicUpdate(store, fn)
}

// Combine derives a value from two stores or derived values. Atomic2 and AtomicGroup notifications are settled through the derived graph before downstream callbacks run, and over Stores, read-only views and Nodra derived values Combine projects and publishes only input pairs that some committed state contained, also while concurrent writers update the same stores and while Combine is being constructed: when one input has not yet caught up with a write the other already reflects, Combine reads both inputs again, recomputing derived inputs that are behind, instead of using the mixed pair. Independent writes are eventually consistent, and concurrent writers may see fewer intermediate values. Old source versions and obsolete computations cannot replace newer ones. project and equals must be pure and safe for concurrent calls, and source references must stay immutable. Close the returned value to detach both sources.
func Combine[A, B, V any](first Observable[A], second Observable[B], project func(A, B) V, equals func(V, V) bool) *Derived[Pair[A, B], V] {
	return core.Combine(first, second, project, equals)
}

// Derive returns a memoized value that selector computes from source. It recomputes when source publishes and notifies its listeners only when the selected value changes according to equals; a nil equals treats every recomputation as a change. selector and equals must be pure and safe for concurrent calls, and returned reference values and listener arguments are read-only; a Combine that reads the derived value before it has caught up with source may also call selector, without changing the derived value. An invalid source or a nil selector returns a closed value. Close the returned value when its owner is done.
func Derive[T, V any](source Observable[T], selector func(T) V, equals func(V, V) bool) *Derived[T, V] {
	return core.Derive(source, selector, equals)
}

// DeriveProjection is Derive over a borrowed projection: it memoizes an owned projection without cloning the whole source. project follows SubscribeProjectionWith's ownership rules, and returned reference values and listener arguments are read-only. Close the returned value when its owner is done.
func DeriveProjection[T, V any](source Reader[T], project func(T) V, equals func(V, V) bool) *Derived[T, V] {
	return core.DeriveProjection(source, project, equals)
}

// Select reads an owned projection and the source version without cloning the whole store. project borrows read-only state under the read lock: it must not mutate or retain source references, reenter the store, or block. Return scalars or copy every mutable reference into the projection, for example with slices.Clone.
func Select[T, V any](source Reader[T], project func(T) V) Snapshot[V] {
	return core.Select(source, project)
}

// SubscribeSelector calls fn with the selected value each time it changes, using the default equality of SubscribeSelectorWith. It returns an idempotent unsubscribe function; nil inputs subscribe to nothing.
func SubscribeSelector[T, S any](source Observable[T], selector func(T) S, fn func(S)) func() {
	return core.SubscribeSelector(source, selector, fn)
}

// SubscribeSelectorWith calls fn with the current and previous selected values each time the selection changes. options.Equals replaces the default equality, which uses == for comparable values and reflect.DeepEqual otherwise, and options.Immediate also calls fn once with the initial selection. It catches up after registration and ignores obsolete computations. selector, equality and fn run on writer goroutines and may overlap, as with Store.Subscribe. It returns an idempotent unsubscribe function; nil inputs subscribe to nothing.
func SubscribeSelectorWith[T, V any](source Observable[T], selector func(T) V, fn func(current, previous V), options SelectorOptions[V]) func() {
	return core.SubscribeSelectorWith(source, selector, fn, options)
}

// SubscribeProjection is SubscribeSelector over a borrowed projection and follows SubscribeProjectionWith's ownership rules.
func SubscribeProjection[T, V any](source Reader[T], project func(T) V, fn func(V)) func() {
	return core.SubscribeProjection(source, project, fn)
}

// SubscribeProjectionWith is an opt-in selector that borrows immutable source snapshots instead of cloning the entire store per notification. project must be pure and safe for concurrent calls, must not retain or mutate source references, and must return independently owned projections. fn must treat the current and previous projections as immutable. Equality, initialization and callback ordering match SubscribeSelectorWith.
func SubscribeProjectionWith[T, V any](source Reader[T], project func(T) V, fn func(current, previous V), options SelectorOptions[V]) func() {
	return core.SubscribeProjectionWith(source, project, fn, options)
}

// Define creates a named store factory. Each Definition.New call runs initial and returns a store with independent state and the definition's ID; a nil initial yields the zero value. The stores copy values shallowly as with New, so initial must return fresh maps, slices and pointers for the instances to stay independent. Use DefineWithClone for isolated mutable state.
func Define[T any](id string, initial func() T) *Definition[T] {
	return core.Define(id, initial)
}

// DefineWithClone creates a named store factory whose stores follow NewWithClone's isolation policy with clone.
func DefineWithClone[T any](id string, initial func() T, clone func(T) T) *Definition[T] {
	return core.DefineWithClone(id, initial, clone)
}

// BindField binds an immediate editor to one field of store's state. read selects the field from the store's current value, and write applies a new value to the latest store candidate, so unrelated concurrent changes survive; an error from write rejects the value and leaves the store unchanged. It returns ErrInvalidBinding when store, read or write is nil. read and write follow the store's ownership policy; keep form drafts separate when edits need an explicit Save or Cancel.
func BindField[T, V any](store *Store[T], read func(T) V, write func(*T, V) error) (Field[T, V], error) {
	return core.BindField(store, read, write)
}

// Track starts recording changes to store for undo and redo, keeping at most limit undo entries; values below one disable recording. It is TrackWith with only Limit set. Closing store does not close History, and closing the History releases its subscription without closing the store, so close History when the store's owner is done with it.
func Track[T any](store *Store[T], limit int) *History[T] { return core.Track(store, limit) }

// TrackWith is Track with HistoryOptions for filtering, restoring and payload budgets. A negative MaxBytes, or a positive MaxBytes without Size, panics with ErrInvalidHistory. A nil store or a Limit below one returns a closed History that records nothing. Closing store does not close History: Undo and Redo then return ErrClosed and keep their stacks, CanUndo, CanRedo and Status keep reporting those stacks, and Status stays open, so close History when the store's owner is done with it.
func TrackWith[T any](store *Store[T], options HistoryOptions[T]) *History[T] {
	return core.TrackWith(store, options)
}

// NewMemoryKeyProvider returns a concurrency-safe in-memory KeyProvider whose current key is key under id, for tests and local applications. It copies key and returns ErrInvalidKey when id is empty or key is not a valid AES key. Production applications should implement KeyProvider over their OS keychain or secret manager.
func NewMemoryKeyProvider(id string, key []byte) (*MemoryKeyProvider, error) {
	return core.NewMemoryKeyProvider(id, key)
}

// NewPersistence connects store to storage and codec. Persistence is explicit: call Load at startup, then Save after successful actions or start AutoSave, AutoSaveWithErrors or AutoSaveAsync. Save calls on one instance are serialized, but separate instances sharing a storage destination are not coordinated, so use one instance per destination. A nil store, storage or codec makes Load, Save and AutoSaveAsync return ErrInvalidPersistence.
func NewPersistence[T any](store *Store[T], storage Storage, codec Codec[T]) *Persistence[T] {
	return core.NewPersistence(store, storage, codec)
}

// NewPersistenceWithOptions is NewPersistence with options. options.Validate checks decoded state before Load replaces the store value and saved state before it is written; it must be pure and safe for concurrent calls, and must not mutate its input.
func NewPersistenceWithOptions[T any](store *Store[T], storage Storage, codec Codec[T], options PersistenceOptions[T]) *Persistence[T] {
	return core.NewPersistenceWithOptions(store, storage, codec, options)
}

// NewAsyncAction creates a cancelable action on store with the default LatestWins policy and one shared key. Each Run calls work outside every lock with its context and a progress callback, then apply commits the result to the latest store candidate; canceled and superseded results never commit, and panics in work or apply become task errors. It returns ErrInvalidAction when store, work or apply is nil. Closing store cancels the action's runs; Close rejects new runs, cancels accepted work and waits within its context.
func NewAsyncAction[T, I, R any](store *Store[T], name string, work func(context.Context, I, func(float64)) (R, error), apply func(*T, R) error) (*AsyncAction[T, I, R], error) {
	return core.NewAsyncAction(store, name, work, apply)
}

// NewAsyncActionWithOptions is NewAsyncAction with the scheduling, debounce and retry policy in options. It returns ErrInvalidAction for negative limits or durations, an unknown policy, a positive Debounce with QueueAll, or retries that are not declared Idempotent.
func NewAsyncActionWithOptions[T, I, R any](store *Store[T], name string, work func(context.Context, I, func(float64)) (R, error), apply func(*T, R) error, options ActionOptions[I]) (*AsyncAction[T, I, R], error) {
	return core.NewAsyncActionWithOptions(store, name, work, apply, options)
}

// NewScope returns an open Scope. Close cancels Context, stops observations, then runs deferred drains in reverse registration order: register stores first, persistence second and actions last so work finishes before persistence drains and stores close.
func NewScope() *Scope { return core.NewScope() }

// InvalidateTarget requests a redraw from any InvalidationTarget whenever source publishes. Nil and typed-nil targets are ignored. It accepts read-only and derived sources without depending on a UI toolkit. The caller owns the target lifecycle: unlike Invalidate it does not watch for a window closing, so call the returned stop function when the target is no longer valid.
func InvalidateTarget[T any](source Observable[T], target InvalidationTarget) func() {
	return core.InvalidateTarget(source, target)
}
