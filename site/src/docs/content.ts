import type { Lang } from '../lib/lang';
import { LOCALES } from '../lib/i18n';

// API reference for the root package and its optional subpackages devtools, bridge and nativeui. Signatures, contract notes and examples were checked against the v1.4.0 sources (api.go, internal/core, devtools, bridge, nativeui, docs/architecture.md, docs/bridge.md); every example compiles against the module.

type Pair = [zh: string, en: string];

export interface DocSection {
	id: string;
	group: keyof typeof GROUPS;
	title: Pair;
	lead: Pair;
	// [display name, Go signature, zh, en]
	methods?: [string, string, string, string][];
	file?: string;
	code?: string;
	contract?: Pair[];
	concurrency?: Pair;
}

export const UI = Object.fromEntries(Object.entries(LOCALES).map(([lang, locale]) => [lang, locale.docs])) as Record<Lang, (typeof LOCALES)['zh']['docs']>;

export const GROUPS = {
	start: ['开始', 'Getting started'],
	core: ['核心 Store', 'Core store'],
	derived: ['派生', 'Derived'],
	history: ['历史', 'History'],
	persist: ['持久化', 'Persistence'],
	async: ['异步与所有权', 'Async & ownership'],
	integ: ['集成', 'Integrations'],
	ref: ['参考', 'Reference'],
} satisfies Record<string, Pair>;

export const SECTIONS: DocSection[] = [
	{
		id: 'overview',
		group: 'start',
		title: ['概览', 'Overview'],
		lead: ['Nodra 是面向 Go 桌面应用的并发安全响应式状态库。它为 MyGo 应用提供类似 Zustand 的使用体验，同时让状态独立于渲染器和持久化后端。', 'Nodra is a concurrency-safe reactive state store for Go desktop applications. It brings Zustand-like ergonomics to MyGo apps while keeping state independent from the renderer and persistence backend.'],
		contract: [
			['唯一受支持的导入路径是 github.com/yldm-tech/nodra；根包重新导出 internal/core 的实现类型，应用无需了解仓库布局。', 'The supported import path is github.com/yldm-tech/nodra. The root package re-exports implementation types from internal/core, so applications need no knowledge of the repository layout.'],
			['可选的子包：github.com/yldm-tech/nodra/devtools 是检查器，bridge 让 MyGo 应用的网页跟随 Store，nativeui 让原生 UI 的撤销与重做作用于 History。核心包不依赖它们。', 'Optional subpackages: github.com/yldm-tech/nodra/devtools is the inspector, bridge lets the web pages of a MyGo app follow a Store, and nativeui makes the Undo and Redo of native UI act on a History. The core imports none of them.'],
			['需要 Go 1.27.3 或更新版本；原生集成基于 MyGo v0.7.4。', 'Requires Go 1.27.3 or newer; native integration targets MyGo v0.7.4.'],
		],
	},
	{
		id: 'install',
		group: 'start',
		title: ['安装', 'Install'],
		lead: ['Nodra 是单个 Go 模块。添加依赖后即可在任何 Go 程序中使用 Store，MyGo 窗口不是必需的。', 'Nodra is a single Go module. Once added, a Store works in any Go program — a MyGo window is not required.'],
		file: 'shell',
		code: `go get github.com/yldm-tech/nodra@latest

# verification workflow
go test ./...
go test -race ./...
go vet ./...
golangci-lint run ./...`,
	},
	{
		id: 'quickstart',
		group: 'start',
		title: ['快速开始', 'Quick start'],
		lead: ['创建 Store、订阅变化、提交更新。Subscribe 返回取消函数，Close 释放所有订阅。', 'Create a store, subscribe to changes and commit an update. Subscribe returns a stop function; Close releases every subscription.'],
		file: 'main.go',
		code: `package main

import (
	"fmt"

	"github.com/yldm-tech/nodra"
)

type State struct{ Count int }

func main() {
	store := nodra.New(State{})
	defer store.Close()

	stop := store.Subscribe(func(s nodra.Snapshot[State]) {
		fmt.Println(s.Value.Count)
	})
	defer stop()

	_ = store.Update(func(s *State) { s.Count++ })
}`,
		contract: [
			['取消订阅函数是幂等的，可以安全地多次调用。', 'Unsubscribe functions are idempotent and safe to call more than once.'],
			['Close 之后的写入会被 ErrClosed 拒绝；重复调用 Close 也返回 ErrClosed，最后的值仍可读取。', 'Writes after Close are rejected with ErrClosed; a repeated Close also returns ErrClosed, and the final value stays readable.'],
		],
	},
	{
		id: 'store',
		group: 'core',
		title: ['创建 Store', 'Creating stores'],
		lead: ['New 创建一个版本号为 0 的已打开 Store。两种所有权策略可选：默认的零拷贝引用语义，或由 NewWithClone 提供的完全隔离。', 'New creates an open store at version zero. Two ownership policies exist: the default zero-copy reference semantics, or full isolation via NewWithClone.'],
		methods: [
			['New', 'func New[T any](initial T) *Store[T]', '默认策略。UpdateErr 失败时执行浅层值回滚，不会克隆映射、切片或指针。', 'Default policy. UpdateErr performs a shallow value rollback on failure and does not clone maps, slices or pointers.'],
			['NewWithClone', 'func NewWithClone[T any](initial T, clone func(T) T) *Store[T]', '初始值、更新候选、提交、读取和每次通知都各自克隆，保留候选也无法修改已拥有的状态。', 'Clones initial state, update candidates, commits, reads and every notification, so retaining a candidate cannot modify owned state.'],
			['Create', 'func Create[T any](initial T) *Store[T]', 'Zustand 风格的入口，状态值是显式的 Go 类型。', 'Zustand-style entry point; the state value is explicit Go state.'],
			['Define', 'func Define[T any](id string, initial func() T) *Definition[T]', '类似 Pinia defineStore 的工厂，.New() 创建相互独立的命名实例。', 'Pinia defineStore-like factory; .New() creates independent named instances.'],
			['DefineWithClone', 'func DefineWithClone[T any](id string, initial func() T, clone func(T) T) *Definition[T]', '把克隆隔离策略传播到每个命名实例。', 'Propagates clone isolation to each named instance.'],
		],
		file: 'update_err.go',
		code: `err := store.UpdateErr(func(state *State) error {
	items := maps.Clone(state.Items)
	items["word"] = updatedItem
	state.Items = items
	return validate(*state)
})`,
		contract: [
			['使用 New 时，修改引用数据前先复制它，这样错误、panic、Reset 和已保留的快照才能保持预期值。', 'With New, copy referenced data before mutating it, so errors, panics, Reset and retained snapshots keep their expected values.'],
			['不要在更新之外修改 Get 或 Snapshot 返回的数据，也不要保留传给 New 或 Set 的可变别名。', 'Do not mutate data returned by Get or Snapshot outside an update, and do not retain a mutable alias passed to New or Set.'],
		],
		concurrency: ['克隆函数由应用提供：必须复制所有可变的嵌套数据，必须并发安全，且不得重入 Store。即使克隆函数 panic，Store 的锁也会被释放。', 'The clone function is application-supplied: it must copy all mutable nested data, be concurrency-safe and never reenter the store. The store lock is released even if a clone panics.'],
	},
	{
		id: 'read-write',
		group: 'core',
		title: ['读取与写入', 'Reading & writing'],
		lead: ['快照携带单调递增的版本号。写入在发布前完整准备好候选值，失败时什么也不发布。', 'Snapshots carry a monotonic version. Writes fully prepare a candidate before publishing, and publish nothing on failure.'],
		methods: [
			['Get / State', 'func (s *Store[T]) Get() T', '返回当前值。', 'Returns the current value.'],
			['Snapshot', 'func (s *Store[T]) Snapshot() Snapshot[T]', '返回值及其版本号。', 'Returns the value together with its version.'],
			['Version', 'func (s *Store[T]) Version() uint64', '当前已提交的版本。', 'The currently committed version.'],
			['ReadOnly', 'func (s *Store[T]) ReadOnly() ReadOnly[T]', '只读视图：不提供写入、钩子或 Close，类型断言也无法取回可写的 Store。', 'Read-only view with no mutations, hooks or Close; a type assertion cannot recover the writable Store.'],
			['Set', 'func (s *Store[T]) Set(value T) error', '替换整个值并发布新快照。', 'Replaces the whole value and publishes a new snapshot.'],
			['Update', 'func (s *Store[T]) Update(fn func(*T)) error', '在候选指针上修改并提交。', 'Mutates a candidate pointer and commits it.'],
			['UpdateErr', 'func (s *Store[T]) UpdateErr(fn func(*T) error) error', '回调返回错误或 panic 时恢复原值，不发布任何内容。', 'Restores the previous value and publishes nothing when the callback errors or panics.'],
			['Transaction', 'func (s *Store[T]) Transaction(fn func(*T) error) error', '一次串行化的变更。', 'One serialized mutation.'],
			['Patch', 'func (s *Store[T]) Patch(fn func(*T)) error', '分组变更，只发布一次，对应 Pinia $patch。', 'Grouped change with one publication, like Pinia $patch.'],
			['Reset', 'func (s *Store[T]) Reset() error', '恢复初始值。', 'Restores the initial value.'],
		],
		contract: [
			['失败的回调和 panic 不会发布部分候选值。', 'Failed callbacks and panics do not publish a partial candidate.'],
			['关闭后的写入返回 ErrClosed。', 'ErrClosed rejects writes after close.'],
		],
		concurrency: ['Update 的回调，以及 UpdateErr、Transaction、Do 与 Field 更新的回调，都在 Store 加锁期间运行：可以读取其他 Store（前提是那些 Store 的更新不会同时读取本 Store），但不得写入任何 Store，也不得调用本 Store 的方法。读取只获取各 Store 自己的锁，因此回调中的读取不会等待其他 Store 的事务。', 'Update callbacks, like those of UpdateErr, Transaction, Do and Field updates, run while the store is locked: they may read other stores, as long as no update of those stores reads this one at the same time, but must not write to any store or call methods of their own store. Reads take only the store’s own lock, so a read inside a callback does not wait on transactions of other stores.'],
	},
	{
		id: 'observe',
		group: 'core',
		title: ['订阅与观察', 'Subscriptions'],
		lead: ['普通订阅、带选项的订阅、变更元数据订阅以及有界的 Watch 通道。', 'Plain subscriptions, subscriptions with options, mutation-metadata subscriptions and bounded Watch channels.'],
		methods: [
			['Subscribe', 'func (s *Store[T]) Subscribe(fn func(Snapshot[T])) func()', '每次提交后调用；返回幂等的取消函数。', 'Called after every commit; returns an idempotent stop function.'],
			['SubscribeWith', 'func (s *Store[T]) SubscribeWith(options SubscribeOptions[T], fn func(Snapshot[T])) func()', '可立即投递初始快照，并可按订阅者比较值。', 'Can deliver the initial snapshot and compare values per subscriber.'],
			['SubscribeSnapshot', 'func (s *Store[T]) SubscribeSnapshot(fn func(Snapshot[T])) func()', '与 Subscribe 相同，是 Store、ReadOnly 与派生值共享的 Observable 接口方法；快照带版本号与事务 ID。', 'Same as Subscribe; it is the Observable method shared by Store, ReadOnly and derived values. Snapshots carry the version and transaction ID.'],
			['SubscribeMutations', 'func (s *Store[T]) SubscribeMutations(fn func(Mutation[T])) func()', '变更元数据包含类型和之前的状态，对应 $subscribe。', 'Mutation metadata includes type and previous state, like $subscribe.'],
			['Watch', 'func (s *Store[T]) Watch(ctx context.Context, buffer int) <-chan Snapshot[T]', '容量为 buffer（至少为 1）的快照通道；满了时丢弃最旧的一条、保留最新快照；ctx 取消或 store 关闭时关闭。', 'Snapshot channel of capacity buffer (at least 1); when full it drops the oldest entry and keeps the newest; closed when ctx ends or the store closes.'],
			['Close', 'func (s *Store[T]) Close() error', '移除订阅者并关闭活动的 Watch 流；最后的快照仍可读取，重复调用返回 ErrClosed。', 'Removes subscribers and closes active watch streams; the final snapshot stays readable and repeated calls return ErrClosed.'],
		],
		file: 'subscribe.go',
		code: `stop := store.SubscribeWith(nodra.SubscribeOptions[State]{Immediate: true},
	func(s nodra.Snapshot[State]) {
		log.Printf("v%d count=%d", s.Version, s.Value.Count)
	})
defer stop()`,
		concurrency: ['监听器在 store 锁释放后、于写入者 goroutine 中同步运行，因此可以安全地调用 Get、Update、Close 或取消订阅。不同的并发写入者可能并发调用监听器；已开始的回调可能在较新版本的回调之后才结束。需要串行副作用时，请使用自己的队列或互斥锁。', 'Listeners run synchronously on the writer goroutine after the store lock is released, so they may safely call Get, Update, Close or unsubscribe. Different concurrent writers may invoke listeners concurrently, and a callback already running can finish after one for a newer version. Use your own queue or mutex for serialized side effects.'],
	},
	{
		id: 'atomic',
		group: 'core',
		title: ['批量与原子事务', 'Batches & atomic'],
		lead: ['Batch 会把嵌套的通知合并，在最外层结束时统一发布一次。需要多个 Store 同时生效时，使用 Atomic2 或 AtomicGroup。', 'Batch groups nested notifications into one publication at the outermost boundary. Use Atomic2 or AtomicGroup when several stores must become visible together.'],
		methods: [
			['Batch', 'func (s *Store[T]) Batch(fn func()) error', '分组嵌套通知，在最外层发布一次。并发写入者可以加入打开的批次。', 'Groups nested notifications and publishes once at the outermost boundary. Concurrent writers may join an open batch.'],
			['BatchErr', 'func (s *Store[T]) BatchErr(fn func() error) error', '同样的分组语义；错误之前成功的更新仍然保留。', 'Same grouping semantics; successful updates before the error remain committed.'],
			['Atomic2', 'func Atomic2[A, B any](first *Store[A], second *Store[B], fn func(*A, *B) error) error', '两个不同类型的 store 一起提交，按地址排序加锁。', 'Commits two stores of different types together, locking by address.'],
			['AtomicGroup', 'func AtomicGroup(ops ...AtomicOp) error', '任意数量 store 的原子提交；重复的 store 会被拒绝。', 'Atomic commit across any number of stores; duplicate stores are rejected.'],
			['AtomicUpdate', 'func AtomicUpdate[T any](store *Store[T], fn func(*T) error) AtomicOp', '为 AtomicGroup 构造类型化操作。', 'Builds a typed operation for AtomicGroup.'],
		],
		file: 'transfer.go',
		code: `err := nodra.AtomicGroup(
	nodra.AtomicUpdate(accounts, func(a *Accounts) error {
		a.Balance -= 100
		return nil
	}),
	nodra.AtomicUpdate(ledger, func(l *Ledger) error {
		l.Entries = append(slices.Clone(l.Entries), Entry{Amount: -100})
		return nil
	}),
)`,
		contract: [
			['回调错误或 panic 时，所有 store 的值和版本都保持不变。', 'A callback error or panic leaves every value and version unchanged.'],
			['成功的变更事件共享同一个事务 ID；通知在所有赋值完成后才运行。', 'Successful mutation events share one transaction ID; notifications run after all assignments.'],
			['在打开的 Batch 中读取会看到批次的写入，但版本号仍是最后一次发布的版本；批次发布时它们才获得新版本。', 'Reads inside an open batch see its writes under the version published last; the batch’s publication gives them a new version.'],
			['Batch 打开期间，Atomic2 或 AtomicGroup 向该 store 的提交会与事务一起发布批次之前的写入，批次随后只发布之后的写入；每个变更事件的 Previous 是订阅者最后收到的值。', 'An Atomic2 or AtomicGroup commit into a store with an open batch publishes the batch’s earlier writes with the transaction, and the batch then publishes only the writes made after it; each mutation’s Previous is the value subscribers last received.'],
			['在打开的 Batch 中关闭 store 会放弃批次的发布，但最终值仍获得自己的新版本，不会沿用一个已随旧值发布过的版本号。', 'Closing the store inside an open batch drops the batch’s publication but still gives the final value a version of its own, never one already published with an older value.'],
		],
		concurrency: ['在原子候选回调中不要调用任何 Store 方法——只修改传入的候选指针。', 'Never call Store methods from an atomic candidate callback — mutate only the supplied candidate pointers.'],
	},
	{
		id: 'derived',
		group: 'derived',
		title: ['派生值与选择器', 'Derived values & selectors'],
		lead: ['Reader[T] 是投影与 DevTools 接受的观察接口；更小的 Observable[T] 被 Derive、Combine 和失效辅助函数接受。', 'Reader[T] is the observation surface accepted by projections and DevTools; the smaller Observable[T] is accepted by Derive, Combine and invalidation helpers.'],
		methods: [
			['Derive', 'func Derive[T, V any](source Observable[T], selector func(T) V, equals func(V, V) bool) *Derived[T, V]', '带订阅的记忆化派生值，对应 getters / computed。', 'Memoized derived value with subscriptions, like getters / computed.'],
			['DeriveProjection', 'func DeriveProjection[T, V any](source Reader[T], project func(T) V, equals func(V, V) bool) *Derived[T, V]', '借用已提交快照计算派生值，避免整状态克隆。', 'Derives from borrowed committed snapshots without a full-state clone.'],
			['Combine', 'func Combine[A, B, V any](first Observable[A], second Observable[B], project func(A, B) V, equals func(V, V) bool) *Derived[Pair[A, B], V]', '组合两个来源；原子事务在下游回调前结算，避免菱形中间值。', 'Combines two sources; atomic transactions settle before downstream callbacks, preventing diamond intermediates.'],
			['Select', 'func Select[T, V any](source Reader[T], project func(T) V) Snapshot[V]', '显式借用读取：在源读锁下执行，必须返回自有数据。', 'Explicit borrowed read under the source read lock; must return owned data.'],
			['SubscribeSelector', 'func SubscribeSelector[T, S any](source Observable[T], selector func(T) S, fn func(S)) func()', '只在选中值变化时回调。', 'Calls back only when the selected value changes.'],
			['SubscribeSelectorWith', 'func SubscribeSelectorWith[T, V any](source Observable[T], selector func(T) V, fn func(current, previous V), options SelectorOptions[V]) func()', '同时提供当前值与上一个值；Equals 可替换默认比较。', 'Delivers current and previous values; Equals replaces the default comparison.'],
			['SubscribeProjection(With)', 'func SubscribeProjectionWith[T, V any](source Reader[T], project func(T) V, fn func(current, previous V), options SelectorOptions[V]) func()', '借用不可变快照，复用选择器的相等性与版本排序。', 'Borrows immutable snapshots, reusing selector equality and version ordering.'],
			['Derived.Done', 'func (d *Derived[T, V]) Done() <-chan struct{}', '在 Close 时关闭；由无效来源或 nil 选择器构造的派生值一开始就已关闭。关闭来源不会关闭派生值，它保留最后的值。bridge 据此在派生值关闭时以 ErrClosed 结束网页对它的 watch。', 'Closed by Close, and already closed for a value built from an invalid source or a nil selector. Closing a source does not close the derived value, which keeps its last value. The bridge uses it to end the pages’ watches of a closed derived value with ErrClosed.'],
		],
		file: 'derived.go',
		code: `doubled := nodra.Derive(store.ReadOnly(),
	func(s State) int { return s.Count * 2 },
	func(a, b int) bool { return a == b },
)
defer func() { _ = doubled.Close() }()

stop := nodra.SubscribeSelector(store, func(s State) string { return s.Filter },
	func(filter string) { log.Println("filter →", filter) })
defer stop()`,
		contract: [
			['默认比较：两个动态值都可比较时用 ==（保留指针身份），否则用 reflect.DeepEqual；它区分 nil 与空集合。', 'Default comparison uses == when both dynamic values are comparable (preserving pointer identity), otherwise reflect.DeepEqual, which distinguishes nil from empty collections.'],
			['大集合请提供合适的比较器，例如 slices.Equal 或 maps.Equal。', 'Large collections should use a suitable comparator such as slices.Equal or maps.Equal.'],
			['equals 为 nil 时，派生值在每个被接受的源版本上替换值并通知；只有传入相等性函数才会过滤未变化的值，例如派生值用于 Invalidate 或 bridge 时。', 'With a nil equals, a derived value replaces its value and notifies on every accepted source version; only an equality function filters unchanged values, for example when the derived value feeds Invalidate or the bridge.'],
			['Atomic2 与 AtomicGroup 按高度结算派生值（派生值比它最深的来源深一层），因此深度不同的分支先于组合它们的值结算，派生值以它在该事务中收到的最新快照结算。只有 Nodra 自己的派生值带高度：应用实现的 Observable 视为高度 0，即使它转发某个派生值也是如此，所以基于这种包装和该派生值其他输入的 Combine 可能看到只应用了一半的事务。', 'Atomic2 and AtomicGroup settle derived values by height (a derived value is one deeper than its deepest source), so branches of unequal depth settle before the value that combines them, and a derived value settles on the newest snapshot it received for the transaction. Only Nodra’s own derived values carry a height: an application Observable counts as height zero even when it forwards a derived value, so a Combine over such a wrapper and the derived value’s other inputs may see a half-applied transaction.'],
			['所有者使用完毕后对派生值调用 Close。', 'Call Close on derived values when their owner is done.'],
		],
		concurrency: ['相等性函数在订阅与派生缓存锁之外运行，可能并发执行并针对更新的基线重试——必须是纯函数且并发安全。比较失败时保留之前的投影及其版本。', 'Equality functions run outside subscription and derived-cache locks, may run concurrently and retry against a newer baseline — keep them pure and concurrency-safe. A failed comparison leaves the previous projection and version unchanged.'],
	},
	{
		id: 'history',
		group: 'history',
		title: ['撤销与重做', 'Undo & redo'],
		lead: ['Track 为 store 提供撤销/重做。TrackWith 可只比较相关字段，并在时间旅行时合并记录值。', 'Track adds undo/redo to a store. TrackWith can compare only relevant fields and merge recorded values on travel.'],
		methods: [
			['Track', 'func Track[T any](store *Store[T], limit int) *History[T]', '注册后再次检查源，忽略过时版本。', 'Rechecks the source after registering and ignores obsolete versions.'],
			['TrackWith', 'func TrackWith[T any](store *Store[T], options HistoryOptions[T]) *History[T]', '字段过滤、合并策略与负载预算。', 'Field filtering, merge policy and payload budgets.'],
			['Undo / Redo', 'func (h *History[T]) Undo() error', '选择目标前先追上已提交的源状态。', 'Catches up with committed source state before selecting a target.'],
			['CanUndo / CanRedo', 'func (h *History[T]) CanUndo() bool', '现在是否有可以恢复的步骤。与 HistoryState 的 CanUndo、CanRedo 一致，分组打开期间或 History 关闭后为 false；它不感知打开的 Batch（此时 Undo 返回 ErrHistoryBatch）或已关闭的源 Store（此时返回 ErrClosed）。', 'Whether Undo or Redo has a step to restore now. Like HistoryState’s CanUndo and CanRedo, false while a group is open or after History is closed; it does not observe an open Batch, where Undo returns ErrHistoryBatch, or a closed source Store, where it returns ErrClosed.'],
			['BeginGroup', 'func (h *History[T]) BeginGroup() (func(), error)', '只记录第一个相关基线；分组可嵌套，可跨帧。返回的函数结束分组。', 'Records only the first relevant baseline; groups nest and may span frames. The returned function ends the group.'],
			['Status', 'func (h *History[T]) Status() ReadOnly[HistoryState]', '可用性发布到独立的只读 status Store。', 'Availability is published to a separate read-only status Store.'],
			['Close', 'func (h *History[T]) Close() error', '释放订阅，不关闭 store。', 'Releases the subscription without closing the store.'],
		],
		file: 'history.go',
		code: `history := nodra.Track(store, 100)
defer func() { _ = history.Close() }()

_ = store.Set(State{Count: 4})
if err := history.Undo(); err != nil { // ErrNoUndo when empty
	log.Println(err)
}
_ = history.Redo()`,
		contract: [
			['在打开的 Batch 中撤销/重做返回 ErrHistoryBatch；在分组中返回 ErrHistoryGroup，状态和栈都不变。', 'Undo/redo inside an open Batch returns ErrHistoryBatch, and inside a group ErrHistoryGroup, without changing state or stacks.'],
			['被忽略的变化推进观察到的源版本，但不新增步骤，也不使重做失效。', 'Ignored changes advance the observed source version without adding a step or invalidating redo.'],
			['被淘汰和弹出的条目会清零，切片的空余容量不会继续持有旧引用。', 'Evicted and popped entries are zeroed so unused slice capacity does not retain old references.'],
			['关闭被追踪的 Store 不会关闭 History：此后 Undo 与 Redo 返回 ErrClosed 并保留各自的栈，CanUndo、CanRedo 与 Status 继续反映这些栈，Status 保持打开。Store 的所有者用完后请关闭 History。', 'Closing the tracked store does not close History: Undo and Redo then return ErrClosed and keep their stacks, CanUndo, CanRedo and Status keep reporting those stacks, and Status stays open. Close History when the store’s owner is done with it.'],
		],
		concurrency: ['策略回调必须是纯函数且不可重入：历史记录在持有自身锁（撤销/重做时还持有 Store 锁）的情况下同步状态。', 'Policy callbacks must be pure and non-reentrant: history catches up while holding its lock, and during travel the Store lock.'],
	},
	{
		id: 'persistence',
		group: 'persist',
		title: ['持久化', 'Persistence'],
		lead: ['持久化是显式的：Storage 拥有字节，Codec[T] 拥有编码。Load、Save 是显式操作；三种 AutoSave 的关闭与错误行为各不相同。', 'Persistence is explicit: Storage owns bytes and Codec[T] owns encoding. Load and Save are explicit operations; the three AutoSave variants differ in shutdown and error behavior.'],
		methods: [
			['NewPersistence', 'func NewPersistence[T any](store *Store[T], storage Storage, codec Codec[T]) *Persistence[T]', '每个存储目标使用一个实例。', 'Use one instance per storage destination.'],
			['Load', 'func (p *Persistence[T]) Load(ctx context.Context) error', '启动操作，不与并发更新构成事务。解码后发布加载的值。', 'A startup operation, not a transaction with concurrent updates. Publishes the loaded value after decoding.'],
			['LoadBackup', 'func (p *Persistence[T]) LoadBackup(ctx context.Context) error', '读取并验证备份后替换内存，不修复或删除主文件。', 'Reads and validates the backup, then replaces memory; does not repair or delete the primary.'],
			['LoadWithRecovery', 'func (p *Persistence[T]) LoadWithRecovery(ctx context.Context) error', '显式的先主后备策略；两者都失败时返回带双重原因的 RecoveryError。主文件读取、解码或校验失败时改读备份，包括包装了 ErrInvalidPersistence 并附带上下文的解码或校验错误，例如损坏的 VersionedJSONCodec 信封。', 'Explicit primary-then-backup policy; returns RecoveryError with both causes when neither works. A primary that fails to read, decode or validate falls back to the backup, including a decode or validate error that wraps ErrInvalidPersistence with more context, such as a damaged VersionedJSONCodec envelope.'],
			['Save', 'func (p *Persistence[T]) Save(ctx context.Context) error', '串行化执行，获取保存锁后才读取当前状态。', 'Serialized; takes current state only after acquiring the save lock.'],
			['AutoSave', 'func (p *Persistence[T]) AutoSave(ctx context.Context) func()', '有意的尽力而为保存。', 'Intentional best-effort saving.'],
			['AutoSaveWithErrors', 'func (p *Persistence[T]) AutoSaveWithErrors(ctx context.Context) (<-chan error, func())', '通过有界的错误通道报告失败；调用停止函数不会等待挂起的写入完成。', 'Exposes failures through a bounded channel; stop does not drain pending writes.'],
			['AutoSaveAsync', 'func (p *Persistence[T]) AutoSaveAsync(ctx context.Context, options AsyncOptions) (*AsyncPersistence[T], error)', '单个后台写入协程的延迟写入，详见下方「异步写入」。', 'Single-worker write-behind; see Async write-behind.'],
		],
		file: 'persistence.go',
		code: `persistence := nodra.NewPersistence(store,
	nodra.FileStorage{Path: "state.json", Backup: true},
	nodra.JSONCodec[State]{},
)
if err := persistence.LoadWithRecovery(ctx); err != nil {
	var rec *nodra.RecoveryError
	if errors.As(err, &rec) {
		// neither primary nor backup could be loaded
	}
}`,
		contract: [
			['启动时必须先决定如何处理主文件加载失败，再启用自动保存。库不会静默选择恢复策略。', 'Startup must decide how to handle a failed primary load before enabling autosave. The library never silently chooses a recovery policy.'],
			['备份是主文件之后的第二次原子替换，因此备份失败可能报告部分成功的 Save。', 'Backup is a second atomic replacement after the primary, so a backup failure can report a partially successful Save.'],
			['LoadWithRecovery 遇到取消、已关闭的 Store 或无效配置时不尝试备份，直接返回主文件的错误，而不是 RecoveryError。无效配置指 Persistence 自身、存储读取（例如空的 FileStorage 路径，或已变成符号链接的 CheckedFileStorage 主文件）或编解码器直接返回的 ErrInvalidPersistence（例如缺少 Inner 或 Keys 的 AESGCMCodec，或 Version 小于 1 的 VersionedJSONCodec）。', 'LoadWithRecovery stops before the backup on cancellation, a closed Store or an invalid configuration, and returns the primary error rather than a RecoveryError. An invalid configuration is ErrInvalidPersistence returned by Persistence itself, by a storage read (such as an empty FileStorage path or a CheckedFileStorage primary that became a symlink), or unwrapped by a codec (such as an AESGCMCodec without Inner or Keys, or a VersionedJSONCodec with Version below 1).'],
		],
		concurrency: ['同一 Persistence 实例串行化 Save，避免慢的旧写入覆盖新的保存。普通 FileStorage 的多个实例之间不协调，除非所有写入者都使用 CheckedFileStorage。', 'One Persistence instance serializes Save so a slow older write cannot replace a newer one. Separate instances over plain FileStorage are not coordinated unless every writer uses CheckedFileStorage.'],
	},
	{
		id: 'codecs',
		group: 'persist',
		title: ['编解码与存储', 'Codecs & storage'],
		lead: ['内置持久化栈可自由组合：迁移、加密与文件保证各自独立。', 'The built-in persistence stack composes freely: migration, encryption and file guarantees are independent layers.'],
		methods: [
			['JSONCodec[T]', 'type JSONCodec[T any] struct{}', '标准 JSON 编码。', 'Plain JSON encoding.'],
			['VersionedJSONCodec[T]', 'type VersionedJSONCodec[T any] struct{ Version int; Migrations map[int]JSONMigration; AllowUnversioned bool }', '写入 schemaVersion 与 state 信封；迁移 n 只负责 n → n+1。', 'Writes a schemaVersion + state envelope; migration n upgrades only n → n+1.'],
			['AESGCMCodec[T]', 'type AESGCMCodec[T any] struct{ Inner Codec[T]; Keys KeyProvider }', '认证加密，只保存密钥 ID、随机数和密文；通过 KeyProvider 轮换密钥。', 'Authenticated encryption storing only key ID / nonce / ciphertext; rotates through a KeyProvider.'],
			['FileStorage', 'type FileStorage struct{ Path string; Perm os.FileMode; Backup, Durable, Journal bool }', '原子替换文件；Journal 提供两阶段崩溃恢复记录；Durable 在重命名后同步父目录。Windows 没有目录 fsync，Durable 在那里不起作用，也不会使保存失败。', 'Atomic file replacement; Journal adds a two-phase crash record; Durable syncs the parent directory after rename. Windows has no directory fsync, so Durable adds nothing there and does not fail the save.'],
			['WithConflictDetection', 'func (f FileStorage) WithConflictDetection(ctx context.Context) (*CheckedFileStorage, error)', '基于 SHA-256 内容指纹与 gofrs/flock 文件锁，实现跨进程的比较后替换。', 'SHA-256 baseline fingerprint plus gofrs/flock cross-process compare-and-replace.'],
		],
		file: 'codec.go',
		code: `codec := nodra.VersionedJSONCodec[wordState]{
	Version: 1, AllowUnversioned: true,
	Migrations: map[int]nodra.JSONMigration{
		0: func(raw json.RawMessage) (json.RawMessage, error) {
			var words map[string]int
			if err := json.Unmarshal(raw, &words); err != nil {
				return nil, err
			}
			return json.Marshal(wordState{Words: words})
		},
	},
}
persistence := nodra.NewPersistence(store,
	nodra.FileStorage{Path: path, Backup: true, Journal: true}, codec)`,
		contract: [
			['未来版本和负数版本返回 ErrUnsupportedVersion，缺失步骤返回 ErrMissingMigration。', 'Future and negative versions fail with ErrUnsupportedVersion; missing steps with ErrMissingMigration.'],
			['所有迁移步骤和最终类型化解码都在 Load 调用 Set 之前完成；失败的解码不改变状态、不发布、不写入存储。', 'All steps and final typed decoding complete before Load calls Set; a failed decode changes no state, emits nothing and writes nothing.'],
			['冲突时返回期望/实际指纹，不替换主文件或备份。符号链接与非常规文件会被拒绝。', 'A conflict returns expected/actual fingerprints without replacing primary or backup. Symlinked and non-regular files are rejected.'],
			['Journal 的清理先删除提交标记，再删除日志。遗留的、没有匹配日志的提交标记属于一次已经完成的保存，恢复时会丢弃它并读取主文件。', 'Journal cleanup removes the commit marker before the journal. A leftover marker without a matching journal belongs to a save that already finished, so recovery discards it and reads the primary.'],
		],
	},
	{
		id: 'write-behind',
		group: 'persist',
		title: ['异步写入', 'Async write-behind'],
		lead: ['AutoSaveAsync 持有一个后台写入协程。Store 的通知只登记目标版本，并通过有界信号通道唤醒它，从不在通知中编码或等待存储。尾沿防抖合并突发写入。', 'AutoSaveAsync owns one worker. Store notifications only request a version and wake the worker through a bounded signal channel — they never encode or wait on storage. A trailing-edge debounce merges bursts.'],
		methods: [
			['AsyncOptions', 'type AsyncOptions struct{ Debounce, MaxWait time.Duration }', 'Debounce 是尾沿延迟，每次更新都会重新计时，因此比它更频繁地持续到来的更新会把写入推迟到更新停下，或推迟到 Flush 或 Close。MaxWait 限制这种推迟：后台协程开始等待挂起的变化后，最多 MaxWait 就开始写入最新状态，即使更新仍在继续；MaxWait 比 Debounce 短时也限制延迟本身。零值保持不设上限，负值使 AutoSaveAsync 返回 ErrInvalidPersistence，Debounce 为零时 MaxWait 不起作用。', 'Debounce is a trailing-edge delay that every update restarts, so updates arriving more often than Debounce postpone the write until they pause, or until Flush or Close. MaxWait bounds that postponement: the worker starts writing the latest state at most MaxWait after it begins waiting for pending changes, even if updates continue, and a MaxWait shorter than Debounce caps the delay itself. Zero keeps it unbounded, a negative value makes AutoSaveAsync return ErrInvalidPersistence, and MaxWait has no effect when Debounce is zero.'],
			['Flush', 'func (a *AsyncPersistence[T]) Flush(ctx context.Context) error', '捕获目标版本、绕过防抖，等待到达或超过该版本的确认。重叠的 Flush 共享写入。', 'Captures a target version, bypasses debounce and waits for an acknowledgement at or beyond it. Overlapping flushes share writes.'],
			['Errors', 'func (a *AsyncPersistence[T]) Errors() <-chan error', '容量为 8，超出时丢弃。', 'Capacity eight; excess entries are dropped.'],
			['Close', 'func (a *AsyncPersistence[T]) Close(ctx context.Context) error', '停止订阅、刷新最终版本、结束后台协程；重复或并发调用都返回第一次关闭的结果。', 'Stops subscriptions, flushes the final target and cancels the worker; repeated and concurrent closes return the first result.'],
		],
		file: 'shutdown.go',
		code: `shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := writer.Close(shutdownCtx); err != nil {
	// the store still owns the data: retry with a fresh context
	_ = persistence.Save(context.Background())
}`,
		contract: [
			['不会积累完整状态快照的队列；写入期间到来的变化会触发对最新快照的后续写入。', 'No queue of full snapshots accumulates; changes during a write trigger a follow-up write of the latest snapshot.'],
			['写入失败后状态保持挂起，后续变化或 Flush 会重试；没有无限的自动重试循环。', 'After a write failure the state stays pending and a later change or Flush retries it; there is no unbounded retry loop.'],
			['需要可靠结果的调用方必须使用 Flush 或 Close 并处理返回的错误。', 'Callers that need a reliable result must use Flush or Close and handle the returned error.'],
			['后台写入时 PersistenceOptions.Validate、Codec.Encode 或 Storage.Save 的 panic 会转为 PanicError 写入失败，经 Errors、Flush 与 Close 报告，状态保持挂起，进程不会崩溃。同步的 Save 与 AutoSave 仍把 panic 传给调用方。', 'A panic from PersistenceOptions.Validate, Codec.Encode or Storage.Save during a background write becomes a PanicError write failure, reported through Errors, Flush and Close with the state left pending, instead of crashing the process. Synchronous Save and AutoSave still propagate panics to their caller.'],
		],
		concurrency: ['调用 Close 前请先完成对 Store 的修改。如果先取消父上下文再关闭 Store，只会停止后台协程，不保证写完所有数据。', 'Finish mutating the store before Close. Canceling the parent and closing the store stops the worker without guaranteeing a drain.'],
	},
	{
		id: 'async',
		group: 'async',
		title: ['异步动作', 'Async actions'],
		lead: ['NewAsyncAction 创建可取消的「最新结果」或排队工作。工作在不持有任何锁的情况下执行，然后获取源写锁和代际锁检查所有权并应用结果。', 'NewAsyncAction creates cancelable latest-result or queued work. Work runs without locks, then takes the source write lock and generation lock to check ownership and apply its result.'],
		methods: [
			['NewAsyncAction', 'func NewAsyncAction[T, I, R any](store *Store[T], name string, work func(context.Context, I, func(float64)) (R, error), apply func(*T, R) error) (*AsyncAction[T, I, R], error)', '默认 LatestWins：新的运行以 ErrSuperseded 取消同一 Key 的上一个运行（未设置 Key 时共用一个 Key）。', 'Defaults to LatestWins: a new run cancels the previous run with the same Key with ErrSuperseded (a nil Key means one shared key).'],
			['NewAsyncActionWithOptions', 'func NewAsyncActionWithOptions[T, I, R any](store *Store[T], name string, work func(context.Context, I, func(float64)) (R, error), apply func(*T, R) error, options ActionOptions[I]) (*AsyncAction[T, I, R], error)', 'ActionOptions：Policy（LatestWins / QueueAll）、Key、MaxConcurrent、QueueLimit、Debounce，以及 Retry（MaxAttempts、Backoff、MaxBackoff、Idempotent）。', 'ActionOptions: Policy (LatestWins / QueueAll), Key, MaxConcurrent, QueueLimit, Debounce and Retry (MaxAttempts, Backoff, MaxBackoff, Idempotent).'],
			['Run', 'func (a *AsyncAction[T, I, R]) Run(ctx context.Context, input I) (*Task, error)', '每次运行拥有自己的 context 与完成通道。', 'Each run owns its context and completion channel.'],
			['Status', 'func (a *AsyncAction[T, I, R]) Status() ReadOnly[AsyncState]', '独立的只读状态 Store。', 'A separate read-only status Store.'],
			['Close', 'func (a *AsyncAction[T, I, R]) Close(ctx context.Context) error', 'Close 阻止新运行、取消所有运行并在 context 内等待。', 'Close prevents new runs, cancels all runs and waits within the context.'],
			['Metrics / ExportMetrics', 'func (a *AsyncAction[T, I, R]) ExportMetrics(sink MetricsSink) error', '通过小型 MetricsSink 接口导出聚合计数；核心不依赖 Prometheus/OTel。', 'Exports aggregate counters through a small MetricsSink; the core imports neither Prometheus nor OTel.'],
			['ActionMetrics', 'type ActionMetrics struct{ Accepted, Completed, Succeeded, Failed, Canceled, Retried uint64; QueuePeak int; DurationSum, DurationMax time.Duration; Superseded uint64 }', '聚合终态结果，不保留已完成的 Task。每个完成的运行恰好计入 Succeeded、Failed、Canceled、Superseded 之一，与其 Task 发布的终态 TaskPhase 一致。Superseded 统计被同一 Key 的较新运行取代的 LatestWins 运行，无论它们在等待、执行还是即将提交；1.2 及更早版本把它们计入 Failed。Retried 统计额外的工作尝试。', 'Aggregates terminal outcomes without retaining completed Task values. Each completed run counts in exactly one of Succeeded, Failed, Canceled or Superseded, matching the terminal TaskPhase its Task publishes. Superseded counts LatestWins runs replaced by a newer run with the same key, whether they were waiting, working or about to commit; releases up to 1.2 counted them as Failed. Retried counts additional work attempts.'],
			['PanicError', 'type PanicError struct{ Value any }', '标识异步回调、原子事务准备阶段与 Scope 清理中的 panic；普通同步通知的 panic 原样向上传播。', 'Identifies panics in asynchronous callbacks, atomic preparation and Scope cleanup; ordinary synchronous notification panics propagate unchanged.'],
		],
		file: 'search.go',
		code: `search, err := nodra.NewAsyncAction(store, "search",
	func(ctx context.Context, q string, progress func(float64)) ([]Result, error) {
		return api.Search(ctx, q)
	},
	func(s *State, r []Result) error { s.Results = r; return nil },
)
if err != nil {
	return err
}
defer func() { _ = search.Close(shutdownCtx) }()

task, err := search.Run(ctx, "nodra")
if err == nil {
	_ = task.Wait(ctx)
}`,
		contract: [
			['重试需显式声明幂等；MaxAttempts 包含第一次调用。退避从 Backoff 开始逐次翻倍，接近溢出时饱和而不回绕，因此即使 MaxAttempts 很大，每次延迟仍受 MaxBackoff 限制。在退避期间取消、被取代或关闭会结束这次运行，不再调用 work。重试时不会重复执行应用结果的回调。', 'Retry requires an explicit idempotent declaration; MaxAttempts includes the first call. Backoff doubles from Backoff between attempts and saturates instead of overflowing, so MaxBackoff bounds every delay even when MaxAttempts is large. Canceling, superseding or closing during a backoff delay ends the run without another work call. A retry never repeats apply.'],
			['SubscribeMetrics 的回调在任务结束后运行，不在 Task.Wait 或 AsyncAction.Close 等待的范围内，取消订阅后仍可能被调用，并发任务的快照也可能乱序到达；需要最终聚合值时，在 Close 之后读取 Metrics 或调用 ExportMetrics。', 'SubscribeMetrics callbacks run after the task is done, outside what Task.Wait and AsyncAction.Close wait for: they can still run after unsubscribing, and snapshots from concurrent tasks can arrive out of order. For the final aggregate, read Metrics or call ExportMetrics after Close.'],
			['QueueAll 拒绝防抖，而不是静默丢弃已接受的输入。', 'QueueAll rejects debounce instead of silently dropping accepted inputs.'],
		],
		concurrency: ['开始较新的运行不会越过较旧运行已接受的提交边界；较新的结果永远不会被过时的运行覆盖。取消是协作式的，外部副作用不在回滚边界之内。', 'A newer run cannot pass an older run’s accepted commit boundary, and newer results are never overwritten by an obsolete run. Cancellation is cooperative; external effects are outside the rollback boundary.'],
	},
	{
		id: 'scope',
		group: 'async',
		title: ['Scope 所有权', 'Scope'],
		lead: ['Scope 持有订阅、持久化写入器和动作。关闭时先取消上下文、停止观察，再按后进先出的顺序依次清理。', 'A Scope owns subscriptions, persistence writers and actions. Close cancels its context, closes observations, then drains in LIFO order.'],
		methods: [
			['NewScope', 'func NewScope() *Scope', '没有隐式的全局注册表，也不拥有共享 store。', 'No implicit global registry and no ownership of shared stores.'],
			['Context', 'func (s *Scope) Context() context.Context', '关闭开始时被取消。', 'Canceled at the start of shutdown.'],
			['Observe', 'func (s *Scope) Observe(stop func()) error', '登记一个取消订阅函数，Close 时在所有清理之前按后进先出的顺序调用；不等待正在执行的回调。关闭开始后登记的函数会立即调用并返回 ErrClosed，函数 panic 时 ErrClosed 再与一个 PanicError 合并；nil 会被忽略。', 'Owns an unsubscribe function, which Close calls in LIFO order before any drain; callbacks already running are not awaited. One registered after shutdown begins is called immediately and returns ErrClosed, joined with a PanicError if it panics; nil is ignored.'],
			['Defer', 'func (s *Scope) Defer(cleanup func(context.Context) error) error', '登记一个带上下文的清理，Close 时以第一次 Close 的上下文按后进先出的顺序运行，panic 转为 PanicError。关闭开始后登记的清理立即以同一上下文运行，返回 ErrClosed 与清理自身的错误。清理必须响应取消，且不得同步关闭自己所属的 Scope。', 'Registers a context-aware drain, which Close runs in LIFO order with the first Close’s context; a panic becomes a PanicError. One registered after shutdown begins runs immediately with that context and returns ErrClosed joined with its own error. Drains must respect cancellation and must not synchronously Close their own Scope.'],
			['Done', 'func (s *Scope) Done() <-chan struct{}', '所有观察停止、所有清理运行完毕后关闭。', 'Closed once every observation has stopped and every drain has run.'],
			['Close', 'func (s *Scope) Close(ctx context.Context) error', '第一次 Close 只运行一次清理；其他调用方等待同一完成通道或各自的 context。', 'The first Close runs cleanup once; other callers wait on one completion channel or their own context.'],
		],
		file: 'scope.go',
		code: `scope := nodra.NewScope()
store := nodra.New(State{})
if err := scope.Defer(func(context.Context) error { return store.Close() }); err != nil {
	return err
}

persistence := nodra.NewPersistence(store,
	nodra.FileStorage{Path: "state.json"},
	nodra.JSONCodec[State]{},
)
// Load first: autosave writes the current state at once. A missing file is a first start; stop on any other error.
if err := persistence.Load(context.Background()); err != nil && !errors.Is(err, fs.ErrNotExist) {
	return err
}
writer, err := persistence.AutoSaveAsync(context.Background(), nodra.AsyncOptions{Debounce: 200 * time.Millisecond})
if err != nil {
	return err
}
_ = scope.Defer(writer.Close) // LIFO: flushes before the store closes

_ = scope.Observe(store.Subscribe(func(s nodra.Snapshot[State]) {
	log.Printf("v%d count=%d", s.Version, s.Value.Count)
}))

shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
return scope.Close(shutdownCtx)`,
		contract: [
			['先登记 store，再登记持久化，最后登记动作，这样后进先出的关闭会先停下工作，再排空持久化，最后关闭 store。', 'Register stores first, persistence second and actions last, so reverse-order shutdown stops work before persistence drains and stores close.'],
			['关闭时请使用新的截止时间；需要刷新数据的持久化写入应使用独立于 Scope.Context 的上下文。', 'Close with a fresh deadline; persistence workers that must flush should use a context independent of Scope.Context.'],
			['关闭后注册的观察会立即停止；迟到的排空收到第一次的关闭上下文并返回 ErrClosed。', 'Late observations stop immediately; late drains receive the first shutdown context and return ErrClosed.'],
		],
	},
	{
		id: 'mygo',
		group: 'integ',
		title: ['MyGo 集成', 'MyGo integration'],
		lead: ['把 store 订阅连接到 Window.Invalidate。选择器版本只在投影变化时请求原生帧；MyGo 仍然负责渲染，不会自动追踪字段读取。', 'Connects a store subscription to Window.Invalidate. Selector variants request native frames only when the projection changes; MyGo still owns rendering and there is no automatic field-read tracking.'],
		methods: [
			['Invalidate', 'func Invalidate[T any](source Observable[T], window *mygo.Window) func()', '任何提交都请求重绘。', 'Any commit requests a redraw.'],
			['InvalidateSelector', 'func InvalidateSelector[T, V any](source Observable[T], window *mygo.Window, selector func(T) V) func()', '选择器必须包含该窗口显示的所有业务状态。', 'The selector must include every business-state value the window displays.'],
			['InvalidateSelectorWith', 'func InvalidateSelectorWith[T, V any](source Observable[T], window *mygo.Window, selector func(T) V, options SelectorOptions[V]) func()', '自定义相等性。', 'Custom equality.'],
			['InvalidateProjection(With)', 'func InvalidateProjectionWith[T, V any](source Reader[T], window *mygo.Window, project func(T) V, options SelectorOptions[V]) func()', '只在自有投影变化时请求帧，不克隆整个来源；project 遵循 SubscribeProjectionWith 的所有权规则。', 'Requests a frame only when an owned projection changes, without cloning the entire source; project follows SubscribeProjectionWith’s ownership rules.'],
			['InvalidateTarget', 'func InvalidateTarget[T any](source Observable[T], target InvalidationTarget) func()', '最小接口，供自定义原生宿主与测试替身使用，无需导入 MyGo。', 'Minimal interface for custom native hosts and test fakes, without importing MyGo.'],
		],
		file: 'window.go',
		code: `mygo.App.WhenReady(func() {
	window := mygo.NewWindow(mygo.WindowOptions{Title: "Counter", Content: ui.View(view)})
	// Redraws stop when the window closes. Keep the returned stop only to end them sooner: a defer here would end them as soon as WhenReady returns.
	nodra.InvalidateSelector(store, window, func(s State) int {
		return s.Count
	})
})`,
		contract: [
			['辅助函数会注册窗口关闭监听，并返回可重复调用的停止函数。隐藏窗口保留订阅；关闭某个窗口不会关闭 Store 或影响其他窗口。', 'Helpers register a window-close listener and return an idempotent stop. Hiding preserves subscriptions; closing one window neither closes the Store nor affects others.'],
			['临时草稿和控件状态留在界面层自己管理，只用 Store.Update 提交真正编辑过的字段。', 'Keep transient drafts and widget state in the UI owner; commit only edited fields with Store.Update.'],
			['不打开 MyGo 窗口的服务、命令行工具与测试可以用 -tags nodra_nomygo 构建：它去掉接收 *mygo.Window 的五个辅助函数（Invalidate、InvalidateSelector、InvalidateSelectorWith、InvalidateProjection、InvalidateProjectionWith），二进制不再链接 MyGo，也就没有 MyGo 在 init 中把主 goroutine 锁定到 OS 线程的行为，CGO_ENABLED=0 的 Linux 构建成为静态可执行文件。InvalidateTarget 仍然可用；不加该标签时构建与 API 不变。', 'Services, CLIs and tests that never open a MyGo window can build with -tags nodra_nomygo. It leaves out the five helpers that take a *mygo.Window (Invalidate, InvalidateSelector, InvalidateSelectorWith, InvalidateProjection and InvalidateProjectionWith), so MyGo is not linked: its init no longer locks the main goroutine to its OS thread, and a CGO_ENABLED=0 Linux build is a static executable. InvalidateTarget stays available; without the tag the build and API are unchanged.'],
			['构建标签作用于整个程序，只有依赖图中没有任何包调用这五个函数时才能使用：nativeui 以及调用它们的第三方库在该标签下无法编译；bridge 仍可编译，但它自己导入 MyGo，程序因此重新链接 MyGo，CGO_ENABLED=0 的 Linux 构建也不再是静态的。', 'Build tags apply to the whole program, so the tag only works when nothing in the dependency graph calls those five helpers: nativeui, and any third-party library that calls them, does not compile with it; bridge compiles but imports MyGo itself, so the program links MyGo again and its CGO_ENABLED=0 Linux build is not static.'],
		],
		concurrency: ['订阅和选择器运行在写入者 goroutine 中，Window.Invalidate 会安全地把帧请求投递到主线程。其他由界面自身产生的变化仍需 Window.Update。', 'Subscriptions and selectors run on writer goroutines; Window.Invalidate safely posts the frame request to the main thread. Other UI-owned changes still require Window.Update.'],
	},
	{
		id: 'nativeui',
		group: 'integ',
		title: ['原生撤销与重做', 'Native undo & redo'],
		lead: ['可选的 nativeui 子包让 MyGo 原生 UI 窗口的撤销、重做快捷键与「编辑」菜单作用于 History。它与核心包分开，不使用原生 UI 的应用不会链接 MyGo 的 ui 包。', 'The optional nativeui subpackage makes the Undo and Redo shortcuts and Edit menu of a MyGo native UI window act on a History. It is separate from the core, so apps without native UI do not link MyGo’s ui package.'],
		methods: [
			['nativeui.UndoShortcuts', 'func UndoShortcuts[T any](c *ui.Context, history *nodra.History[T])', '在正在构建的视图中注册快捷键：可以撤销时 Cmd+Z（Linux 与 Windows 上为 Ctrl+Z）撤销；可以重做时 Shift+Cmd+Z 重做，Windows 上 Ctrl+Y 也可以。请在窗口视图的顶层、子元素的构建函数之外调用，使快捷键作用于整个窗口。', 'Registers shortcuts in the view being built: while history can undo, Cmd+Z (Ctrl+Z on Linux and Windows) undoes; while it can redo, Shift+Cmd+Z redoes, and Ctrl+Y too on Windows. Call it at the top level of the window’s view, outside the builders of child elements, so the shortcuts serve the whole window.'],
			['nativeui.InvalidateHistory', 'func InvalidateHistory[T any](history *nodra.History[T], window *mygo.Window) (stop func())', '只在 history 开始或不再能够撤销、重做时重绘窗口，使 UndoShortcuts 注册的快捷键与「编辑」菜单跟上；history 的其他变化不触发重绘。窗口关闭或调用返回的函数时停止，可重复调用。', 'Redraws window only when history starts or stops being able to undo or redo, so the shortcuts UndoShortcuts registers and the Edit menu follow; other changes of history’s state draw nothing. It stops when the window closes or the returned function is called, and is safe to call more than once.'],
		],
		file: 'undo.go',
		code: `// History lives as long as the app: close it after App.Run returns.
history := nodra.Track(store, 100)
defer func() { _ = history.Close() }()

mygo.App.WhenReady(func() {
	window := mygo.NewWindow(mygo.WindowOptions{
		Title: "Editor",
		Content: ui.View(func(c *ui.Context) {
			nativeui.UndoShortcuts(c, history) // Cmd+Z, Shift+Cmd+Z and the Edit menu
			// ... the rest of the view
		}),
	})
	// Stops when the window closes; a defer here would stop it as soon as WhenReady returns.
	nativeui.InvalidateHistory(history, window)
})
if err := mygo.App.Run(); err != nil {
	log.Fatal(err)
}`,
		contract: [
			['获得焦点的文本输入框保留自己的撤销：快捷键与「编辑」菜单作用于它的文本，只在没有文本输入框获得焦点时才作用于 history。', 'A focused text input keeps its own undo: the keys and the Edit menu act on its text, and on history only while no text input has the focus.'],
			['在 macOS 上使用 MyGo 0.7.4 时，「编辑」菜单只在撤销、重做确实会生效时启用；在 Linux 与 Windows 上快捷键先收到按键，菜单项作用于文本输入框。', 'With MyGo 0.7.4 on macOS, the Edit menu enables Undo and Redo only while they would act; on Linux and Windows the keys reach the shortcuts first, and the menu items act on text inputs.'],
			['被 history 拒绝的撤销或重做（例如帧构建之后才开始的分组）什么也不做。', 'An Undo or Redo that history refuses, because a group began after the frame was built, does nothing.'],
		],
		concurrency: ['快捷键在主线程上运行，撤销与重做因此在主线程上发布：做慢操作的 Store 订阅者应把工作移到其他 goroutine。', 'The shortcuts run on the main thread, so Undo and Redo publish there: a store subscriber that does slow work should move it to another goroutine.'],
	},
	{
		id: 'bridge',
		group: 'integ',
		title: ['网页桥接', 'Web pages'],
		lead: ['可选的 bridge 子包让 MyGo 应用的网页跟随 Store，并通过 Go 注册的动作修改它，Go 中的同一个 Store 因此是原生窗口与网页共同的唯一数据源。网页只看到 Go 暴露的值，从不直接写入状态。', 'The optional bridge subpackage lets the web pages of a MyGo app follow a Store and change it through actions Go registers, so one Store in Go is the source of truth for native windows and web pages alike. Pages see only what Go exposes and never write state directly.'],
		methods: [
			['bridge.New', 'func New(options Options) *Bridge', '返回一个尚未暴露任何内容的已打开 Bridge。Options.AllowRead 与 AllowDispatch 按状态名和动作名进一步限制可以调用的页面；为 nil 时允许 MyGo 允许的所有页面。Options.FrameInterval 是同一 watch 两帧之间的最短间隔：间隔内的变化合并为最新值，在间隔结束时发出，安静一段时间后的变化立即发出；为零时（默认）每次变化都尽快发送。', 'Returns an open Bridge with nothing exposed. Options.AllowRead and AllowDispatch narrow, per state and action name, which pages may call; nil allows every page MyGo lets call. Options.FrameInterval is the least time between two frames of one watch: changes within it coalesce into the latest, sent when it ends, and a change after a quiet spell goes at once; zero, the default, sends each change as soon as possible.'],
			['bridge.Expose', 'func Expose[T any](b *Bridge, name string, source nodra.Observable[T]) (stop func(), err error)', '原样暴露 source：值必须能编码为 JSON，编码包含的每个字段都会到达网页。返回的停止函数结束网页对它的 watch，可重复调用。', 'Exposes source as it is: its value must encode as JSON, and every field the encoding includes reaches the pages. The returned stop ends the pages’ watches of it and is safe to call more than once.'],
			['bridge.ExposeProjection', 'func ExposeProjection[T, V any](b *Bridge, name string, source nodra.Observable[T], project func(T) V) (stop func(), err error)', '网页只看到 project 的返回值，密钥和大型内部数据留在 Go 中；只有编码后的投影变化时网页才收到新帧。', 'Pages see only what project returns, so secrets and large internals stay in Go; a page receives a frame only when the encoded projection changes.'],
			['bridge.Handle', 'func Handle[A any](b *Bridge, name string, handle func(ctx context.Context, arg A) error) (stop func(), err error)', '注册网页可以派发的动作：JSON 参数解码为 A，缺失或为 null 时取零值；ctx 在网页离开或窗口关闭时取消，handle 返回的错误使网页的 promise 被拒绝。', 'Registers an action pages may dispatch: the JSON argument decodes into an A, a missing or null one into its zero value; ctx is canceled when the page navigates away or its window closes, and the error handle returns rejects the page’s promise.'],
			['Bridge.Bind / BindAs', 'func (b *Bridge) Bind()', '以 DefaultService（Nodra）的名称绑定服务；BindAs(name) 使用其他名称。与 mygo.Bind 一样必须在 App.Run 之前调用，名称已被占用时 panic。', 'Binds the service under DefaultService (Nodra); BindAs(name) uses another name. Like mygo.Bind it must run before App.Run, and it panics when the name is taken.'],
			['Bridge.Get / Watch / Dispatch', 'func (b *Bridge) Get(ctx context.Context, name string) (Frame, error)', '与绑定的服务为网页所做的相同，但不需要 MyGo，供测试和 Go 代码使用。', 'Do what the bound service does for a page, without MyGo, for tests and Go code.'],
			['Bridge.Close', 'func (b *Bridge) Close()', '停止所有暴露，以 ErrClosed 结束网页的 watch，之后的调用返回 ErrClosed。source 保持打开：它们归应用所有。可重复调用。', 'Stops every exposure, ending the pages’ watches with ErrClosed, and makes later calls fail with ErrClosed. Sources stay open: the app owns them. Safe to call more than once.'],
			['Frame', 'type Frame struct{ Name string; Version uint64; Value json.RawMessage }', '网页收到的状态：名称、取值时 source 的版本号，以及 JSON 编码的投影值。版本号属于 source，在一次 watch 内只增不减；再次暴露的名称可能换成另一个 source，其版本号单独计数。', 'A state as pages receive it: its name, the source version it was taken at and the projected value as JSON. Versions are the source’s and only grow within a watch; a name exposed again may hold another source, whose versions count apart.'],
			['bridge.Client', 'var Client fs.FS', 'JavaScript 客户端 nodra.js 及其类型 nodra.d.ts，供从 Go 提供页面的应用使用；使用打包工具的应用把这两个文件复制到前端。', 'The JavaScript client nodra.js and its types nodra.d.ts, for apps that serve their pages from Go; apps with a bundler copy the two files into their frontend.'],
		],
		file: 'bridge.go',
		code: `b := bridge.New(bridge.Options{})
defer b.Close()

// Pages see the count, not the token.
stop, err := bridge.ExposeProjection(b, "counter", store, func(s State) PageState {
	return PageState{Count: s.Count}
})
if err != nil {
	return err
}
defer stop()

stopAdd, err := bridge.Handle(b, "counter/add", func(ctx context.Context, by int) error {
	return store.Update(func(s *State) { s.Count += by })
})
if err != nil {
	return err
}
defer stopAdd()

b.Bind() // before App.Run`,
		contract: [
			['一次 watch 先发送当前帧，之后只在编码后的投影变化时发送。当前帧的版本号等于网页已持有的 after 时跳过，但同一名称在这个 Bridge 上曾被暴露过时（它可能换成了另一个 source）总是发送；为此 Bridge 在关闭前记住每个暴露过的状态名，每个名称占一个小条目。', 'A watch sends the current frame first, then a frame only when the encoded projection changes. The first frame is skipped when its version is the after the page already holds, unless the name was exposed before on this Bridge and may hold another source; for that, a Bridge remembers every name a state was exposed under until it closes, a small entry per name.'],
			['默认情况下，帧只在 MyGo 的通道阻止发送时才合并为最新的一帧，即某个 watch 中网页尚未取走的帧累积到约 1 MiB 时；低于这个量时，处理较慢的网页仍会收到每一次中间变化。状态变化很快时（进度、播放位置、流式结果），把 Options.FrameInterval 设为一帧（约 16 毫秒），网页就不必渲染每个中间值，最终值仍一定送达。', 'By default frames coalesce into the latest only while MyGo’s channel holds back sends, once about 1 MiB of frames the page has not taken is queued for the watch; below that, a slow page still receives each intermediate change. For state that changes fast, such as progress, playback position or streamed results, set Options.FrameInterval to a display frame (about 16 ms) so pages do not render every intermediate value; the final value still arrives.'],
			['MyGo 只允许应用自己的页面调用绑定的服务。AllowRead 与 AllowDispatch 在查找名称之前被询问，因此也会收到没有暴露或注册的名称；被它们拒绝的调用无论名称是否存在都返回 ErrDenied，被拒绝的页面因此无法得知哪些名称存在。被允许的调用遇到没有暴露或注册的名称时返回 ErrNotFound，重复使用的名称返回 ErrExists。', 'MyGo lets only the app’s own pages call bound services. AllowRead and AllowDispatch are asked before the name is looked up, so they may be called with a name nothing is exposed or registered under, and a call they refuse fails with ErrDenied either way, so a refused page cannot tell which names exist. An allowed call to a name nothing is exposed or registered under fails with ErrNotFound, and a name used twice with ErrExists.'],
			['Expose 与 ExposeProjection 对空名称、nil 的 source 或 project，以及没有底层 store 的 Store 或 ReadOnly（例如零值 ReadOnly）返回 ErrInvalid，而不是通过一个永不结束的 watch 提供零值。', 'Expose and ExposeProjection return ErrInvalid for an empty name, a nil source or project, and a Store or ReadOnly without a store, such as a zero ReadOnly, instead of serving its zero value through a watch that never ends.'],
			['JavaScript 客户端 bridge/client/nodra.js 没有依赖：remoteState(name) 的 subscribe 与 getSnapshot 适配 React 的 useSyncExternalStore，dispatch(name, arg) 派发动作。', 'The JavaScript client bridge/client/nodra.js has no dependencies: remoteState(name) has subscribe and getSnapshot for React’s useSyncExternalStore, and dispatch(name, arg) dispatches an action.'],
			['bridge 的 Go API、服务的方法名与参数以及 Frame 的 JSON 字段属于 v1 契约，可以新增字段。', 'The bridge Go API, the service’s method names and arguments, and the Frame JSON fields are part of the v1 contract; fields may be added.'],
		],
		concurrency: ['project 在每个网页的 watch goroutine 上运行，可能同时为多个网页运行：它必须是纯函数、并发安全，并把 source 的引用值视为只读。动作在派发调用的 goroutine 上运行，通过应用的 Store 修改状态，因此历史、持久化与 DevTools 照常看到变化。AllowRead 与 AllowDispatch 必须并发安全。', 'project runs on the goroutine of each page’s watch, possibly for several pages at once: it must be pure, concurrency-safe and treat the source’s reference values as read-only. Handlers run on the dispatching call’s goroutine and change state through the app’s stores, so history, persistence and DevTools see the change as usual. AllowRead and AllowDispatch must be concurrency-safe.'],
	},
	{
		id: 'devtools',
		group: 'integ',
		title: ['DevTools', 'DevTools'],
		lead: ['可选的 devtools 子包通过公开的快照、变更与动作接口观察 Store。本地浏览器界面与带鉴权的接口共用同一套数据模型。', 'The optional devtools subpackage observes stores through exported snapshot, mutation and action APIs. The loopback browser UI and authenticated API share one model.'],
		methods: [
			['devtools.New', 'func New(options Options) *Inspector', '默认只读。', 'Read-only by default.'],
			['devtools.Attach', 'func Attach[T any](i *Inspector, id string, source nodra.Reader[T]) (func(), error)', '显式附加 ID 支持异构 store；代际 ID 防止恢复到旧实例。', 'Explicit IDs allow heterogeneous stores; generation IDs prevent restoring to a previous instance.'],
			['Inspector.Listen', 'func (i *Inspector) Listen() (string, error)', '启动本地带鉴权的检查器，并返回访问地址。', 'Starts the local authenticated inspector and returns its URL.'],
			['Options.AllowRestore', 'AllowRestore bool', '显式开启类型化 JSON 编辑与快照恢复。', 'Explicitly enables typed JSON editing and snapshot restoration.'],
			['Options.Redact', 'Redact func(storeID string, state json.RawMessage) (json.RawMessage, error)', '在保留之前过滤每个编码后的状态，设置后检查器始终只读。返回错误或 panic 都不会影响触发它的 Store 写入或 Attach：该状态不被保留，store 不可恢复；panic 时检查错误是固定文本「state redaction failed」，panic 的值被丢弃，因为它可能包含正在脱敏的状态。之后一次成功的脱敏会恢复正常。', 'Filters each encoded state before retention, and makes the inspector read-only. Neither an error nor a panic reaches the Store write or Attach that triggered it: the state is not retained and the store is not restorable; for a panic the inspection error is the fixed text “state redaction failed”, and the panic value is discarded because it may contain the state being redacted. A later successful redaction recovers.'],
			['Options.MaxStateBytes', 'MaxStateBytes int', '限制脱敏后每个编码状态的大小，默认 1 MiB。超出的状态成为可见的检查错误且不能恢复，但不阻止源更新。浏览器界面的恢复请求可以携带上限以内的任何状态。', 'Caps each encoded state after redaction, 1 MiB by default. An oversize state becomes a visible inspection error and cannot be restored, but does not prevent the source update. A restore request from the browser UI may carry any state within the cap.'],
		],
		file: 'inspect.go',
		code: `inspector := devtools.New(devtools.Options{})
defer inspector.Close()

stop, err := devtools.Attach(inspector, "state", store.ReadOnly())
if err != nil {
	panic(err)
}
defer stop()

url, _ := inspector.Listen()
fmt.Println(url)`,
		contract: [
			['脱敏发生在保留之前，并会禁用对被掩码状态的恢复。', 'Redaction occurs before retention and disables restoring masked state.'],
			['状态恢复严格解码，并在检查器锁之外调用 Set，普通订阅与持久化照常运行。', 'Restoration strictly decodes and calls Set outside inspector locks, so normal subscriptions and persistence run.'],
		],
	},
	{
		id: 'mapping',
		group: 'ref',
		title: ['从 Pinia / Zustand 迁移', 'From Pinia / Zustand'],
		lead: ['Nodra 有意映射概念而不是框架语法。', 'Nodra intentionally maps concepts, not framework syntax.'],
		methods: [
			['create / createStore', 'Create(initial)', '泛型 store 的值是显式的 Go 状态。', 'The generic store value is explicit Go state.'],
			['defineStore(id, opts)', 'Define(id, func() T { … }).New()', '工厂创建相互独立的命名实例。', 'Factory creates independent named instances.'],
			['immutable helpers', 'NewWithClone(initial, clone)', '可选的引用隔离与回滚。', 'Optional isolated references and rollback.'],
			['$patch(fn)', 'Patch(fn) / Batch(fn)', '分组变更只发布一次。', 'One publication for grouped changes.'],
			['getters / computed', 'Derive(store, selector, equals)', '带订阅的记忆化派生值。', 'Memoized derived value with subscriptions.'],
			['$onAction', 'SubscribeActionHooks', '执行前、执行后与出错时的回调。', 'Before, after and error callbacks.'],
			['persist middleware', 'Persistence, JSONCodec, FileStorage', '存储与编码保持可替换。', 'Storage and encoding remain replaceable.'],
			['debounced persistence', 'AutoSaveAsync, Flush, Close', '单个合并写入的后台协程，提供显式的落盘屏障。', 'One coalescing worker with explicit durability barriers.'],
		],
	},
	{
		id: 'compatibility',
		group: 'ref',
		title: ['兼容性', 'Compatibility'],
		lead: ['v1 契约由导出的根包文档、外部 nodra_test 包中的测试以及 compatibility.md 定义。', 'The v1 contract is defined by exported root-package documentation, tests in the external nodra_test package, and compatibility.md.'],
		contract: [
			['内部包、DevTools HTML、错误文本与性能测量不属于公共兼容面，除非在 compatibility.md 中明确列出。', 'Internal packages, DevTools HTML, error text and performance measurements are not public compatibility surfaces unless explicitly listed.'],
			['部分 Go 文档渲染器不会展开泛型类型别名上的方法；本页即为稳定的方法索引。', 'Some Go doc renderers do not expand methods on generic type aliases; this page serves as the stable method index.'],
		],
	},
];
