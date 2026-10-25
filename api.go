package nodra

import (
	"context"

	"github.com/egoist/mygo"
	core "github.com/yldm-tech/nodra/internal/core"
)

// The root package is the stable public entry point. Implementations live in
// internal/core so the repository can keep feature code together without
// changing the import path used by applications.

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
	AESGCMCodec[T any]        = core.AESGCMCodec[T]
	ActionEvent               = core.ActionEvent
	ActionHooks               = core.ActionHooks
	ActionMetrics             = core.ActionMetrics
	ActionOptions[I any]      = core.ActionOptions[I]
	ActionPolicy              = core.ActionPolicy
	ActionTrace               = core.ActionTrace
	AsyncAction[T, I, R any]  = core.AsyncAction[T, I, R]
	AsyncOptions              = core.AsyncOptions
	AsyncPersistence[T any]   = core.AsyncPersistence[T]
	AsyncState                = core.AsyncState
	AtomicOp                  = core.AtomicOp
	CheckedFileStorage        = core.CheckedFileStorage
	Codec[T any]              = core.Codec[T]
	Definition[T any]         = core.Definition[T]
	Derived[T, V any]         = core.Derived[T, V]
	Field[T, V any]           = core.Field[T, V]
	FileConflictError         = core.FileConflictError
	FileFingerprint           = core.FileFingerprint
	FileStorage               = core.FileStorage
	History[T any]            = core.History[T]
	HistoryOptions[T any]     = core.HistoryOptions[T]
	HistoryState              = core.HistoryState
	JSONCodec[T any]          = core.JSONCodec[T]
	JSONMigration             = core.JSONMigration
	KeyProvider               = core.KeyProvider
	LoadError                 = core.LoadError
	MemoryKeyProvider         = core.MemoryKeyProvider
	MetricsSink               = core.MetricsSink
	Mutation[T any]           = core.Mutation[T]
	MutationType              = core.MutationType
	Observable[T any]         = core.Observable[T]
	Pair[A, B any]            = core.Pair[A, B]
	PanicError                = core.PanicError
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
	Store[T any]              = core.Store[T]
	SubscribeOptions[T any]   = core.SubscribeOptions[T]
	Task                      = core.Task
	TaskMetrics               = core.TaskMetrics
	TaskPhase                 = core.TaskPhase
	TaskSnapshot              = core.TaskSnapshot
	VersionedJSONCodec[T any] = core.VersionedJSONCodec[T]
	InvalidationTarget        = core.InvalidationTarget
)

func New[T any](initial T) *Store[T] { return core.New(initial) }

func Create[T any](initial T) *Store[T] { return core.Create(initial) }

func NewWithClone[T any](initial T, clone func(T) T) *Store[T] {
	return core.NewWithClone(initial, clone)
}

func Atomic2[A, B any](first *Store[A], second *Store[B], fn func(*A, *B) error) error {
	return core.Atomic2(first, second, fn)
}

func AtomicGroup(ops ...AtomicOp) error { return core.AtomicGroup(ops...) }

func AtomicUpdate[T any](store *Store[T], fn func(*T) error) AtomicOp {
	return core.AtomicUpdate(store, fn)
}

func Combine[A, B, V any](first Observable[A], second Observable[B], project func(A, B) V, equals func(V, V) bool) *Derived[Pair[A, B], V] {
	return core.Combine(first, second, project, equals)
}

func Derive[T, V any](source Observable[T], selector func(T) V, equals func(V, V) bool) *Derived[T, V] {
	return core.Derive(source, selector, equals)
}

func DeriveProjection[T, V any](source Reader[T], project func(T) V, equals func(V, V) bool) *Derived[T, V] {
	return core.DeriveProjection(source, project, equals)
}

func Select[T, V any](source Reader[T], project func(T) V) Snapshot[V] {
	return core.Select(source, project)
}

func SubscribeSelector[T, S any](source Observable[T], selector func(T) S, fn func(S)) func() {
	return core.SubscribeSelector(source, selector, fn)
}

func SubscribeSelectorWith[T, V any](source Observable[T], selector func(T) V, fn func(current, previous V), options SelectorOptions[V]) func() {
	return core.SubscribeSelectorWith(source, selector, fn, options)
}

func SubscribeProjection[T, V any](source Reader[T], project func(T) V, fn func(V)) func() {
	return core.SubscribeProjection(source, project, fn)
}

func SubscribeProjectionWith[T, V any](source Reader[T], project func(T) V, fn func(current, previous V), options SelectorOptions[V]) func() {
	return core.SubscribeProjectionWith(source, project, fn, options)
}

func Define[T any](id string, initial func() T) *Definition[T] {
	return core.Define(id, initial)
}

func DefineWithClone[T any](id string, initial func() T, clone func(T) T) *Definition[T] {
	return core.DefineWithClone(id, initial, clone)
}

func BindField[T, V any](store *Store[T], read func(T) V, write func(*T, V) error) (Field[T, V], error) {
	return core.BindField(store, read, write)
}

func Track[T any](store *Store[T], limit int) *History[T] { return core.Track(store, limit) }

func TrackWith[T any](store *Store[T], options HistoryOptions[T]) *History[T] {
	return core.TrackWith(store, options)
}

func NewMemoryKeyProvider(id string, key []byte) (*MemoryKeyProvider, error) {
	return core.NewMemoryKeyProvider(id, key)
}

func NewPersistence[T any](store *Store[T], storage Storage, codec Codec[T]) *Persistence[T] {
	return core.NewPersistence(store, storage, codec)
}

func NewPersistenceWithOptions[T any](store *Store[T], storage Storage, codec Codec[T], options PersistenceOptions[T]) *Persistence[T] {
	return core.NewPersistenceWithOptions(store, storage, codec, options)
}

func NewAsyncAction[T, I, R any](store *Store[T], name string, work func(context.Context, I, func(float64)) (R, error), apply func(*T, R) error) (*AsyncAction[T, I, R], error) {
	return core.NewAsyncAction(store, name, work, apply)
}

func NewAsyncActionWithOptions[T, I, R any](store *Store[T], name string, work func(context.Context, I, func(float64)) (R, error), apply func(*T, R) error, options ActionOptions[I]) (*AsyncAction[T, I, R], error) {
	return core.NewAsyncActionWithOptions(store, name, work, apply, options)
}

func NewScope() *Scope { return core.NewScope() }

func Invalidate[T any](source Observable[T], window *mygo.Window) func() {
	return core.Invalidate(source, window)
}

func InvalidateTarget[T any](source Observable[T], target InvalidationTarget) func() {
	return core.InvalidateTarget(source, target)
}

func InvalidateSelector[T, V any](source Observable[T], window *mygo.Window, selector func(T) V) func() {
	return core.InvalidateSelector(source, window, selector)
}

func InvalidateSelectorWith[T, V any](source Observable[T], window *mygo.Window, selector func(T) V, options SelectorOptions[V]) func() {
	return core.InvalidateSelectorWith(source, window, selector, options)
}

func InvalidateProjection[T, V any](source Reader[T], window *mygo.Window, project func(T) V) func() {
	return core.InvalidateProjection(source, window, project)
}

func InvalidateProjectionWith[T, V any](source Reader[T], window *mygo.Window, project func(T) V, options SelectorOptions[V]) func() {
	return core.InvalidateProjectionWith(source, window, project, options)
}
