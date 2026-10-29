// API reference for the root package. Signatures, contract notes and examples were checked against the v1.1.0 sources (api.go, internal/core, devtools, docs/architecture.md); every example compiles against the module.

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

export const UI = {
	zh: {
		kicker: 'API 参考 · github.com/yldm-tech/nodra',
		seeK: '还没看懂？',
		seeT: '看动画演示：失败回滚、批量合并与异步竞争',
		ftChange: '更新日志',
		ftContrib: '参与贡献',
		ftSec: '安全策略',
		search: '搜索方法或章节…',
		noResults: '没找到匹配的方法或章节',
		toc: '目录',
		title: 'Nodra 文档',
		intro: '根包 github.com/yldm-tech/nodra 的完整方法索引与语义约定。每个章节都包含方法签名、行为约定、并发说明和可直接运行的示例。',
		contract: '行为约定',
		concurrency: '并发说明',
		copy: '复制',
		copied: '已复制',
		footnote: '签名、示例代码与行为说明均已对照 v1.1.0 源码核对，示例可直接编译；以 pkg.go.dev 上的文档为准。',
		metaTitle: 'Nodra 文档 · API 参考',
		metaDesc: 'Nodra 根包的完整方法索引与语义约定：Store、选择器、原子事务、撤销重做、持久化、异步动作与 MyGo 集成。',
		ogImageAlt: 'Nodra 文档：根包 API 参考，含方法签名、行为约定与并发说明。',
	},
	en: {
		kicker: 'API REFERENCE · github.com/yldm-tech/nodra',
		seeK: 'PREFER VISUALS?',
		seeT: 'Watch rollback, batching and async races, step by step',
		ftChange: 'Changelog',
		ftContrib: 'Contributing',
		ftSec: 'Security',
		search: 'Search methods or sections…',
		noResults: 'Nothing matches that search',
		toc: 'Contents',
		title: 'Nodra documentation',
		intro: 'The method index and semantic contract for the root package github.com/yldm-tech/nodra. Each group lists signatures, contract points, concurrency notes and a runnable example.',
		contract: 'CONTRACT',
		concurrency: 'CONCURRENCY',
		copy: 'Copy',
		copied: 'Copied',
		footnote: 'Signatures, examples and contract notes are checked against the v1.1.0 sources, and every example compiles; pkg.go.dev is authoritative.',
		metaTitle: 'Nodra docs · API reference',
		metaDesc: 'The method index and semantic contract of the Nodra root package: Store, selectors, atomic transactions, undo/redo, persistence, async actions and MyGo integration.',
		ogImageAlt: 'Nodra docs: the root package API reference with signatures, contracts and concurrency notes.',
	},
};

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
			['可选的检查器位于 github.com/yldm-tech/nodra/devtools，核心包不依赖它。', 'The optional inspector lives in github.com/yldm-tech/nodra/devtools; the core does not import it.'],
			['需要 Go 1.27.3 或更新版本；原生集成基于 MyGo v0.7.2。', 'Requires Go 1.27.3 or newer; native integration targets MyGo v0.7.2.'],
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
			['CanUndo / CanRedo', 'func (h *History[T]) CanUndo() bool', '当前是否可以撤销或重做。', 'Whether undo or redo is currently available.'],
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
			['LoadWithRecovery', 'func (p *Persistence[T]) LoadWithRecovery(ctx context.Context) error', '显式的先主后备策略；两者都失败时返回带双重原因的 RecoveryError。', 'Explicit primary-then-backup policy; returns RecoveryError with both causes when neither works.'],
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
			['FileStorage', 'type FileStorage struct{ Path string; Perm os.FileMode; Backup, Durable, Journal bool }', '原子替换文件；Journal 提供两阶段崩溃恢复记录；Durable 在重命名后同步父目录。', 'Atomic file replacement; Journal adds a two-phase crash record; Durable syncs the parent directory after rename.'],
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
		],
	},
	{
		id: 'write-behind',
		group: 'persist',
		title: ['异步写入', 'Async write-behind'],
		lead: ['AutoSaveAsync 持有一个后台写入协程。Store 的通知只登记目标版本，并通过有界信号通道唤醒它，从不在通知中编码或等待存储。尾沿防抖合并突发写入。', 'AutoSaveAsync owns one worker. Store notifications only request a version and wake the worker through a bounded signal channel — they never encode or wait on storage. A trailing-edge debounce merges bursts.'],
		methods: [
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
			['重试需显式声明幂等；MaxAttempts 包含第一次调用，指数退避受 MaxBackoff 限制。重试时不会重复执行应用结果的回调。', 'Retry requires an explicit idempotent declaration; MaxAttempts includes the first call and backoff is bounded by MaxBackoff. A retry never repeats apply.'],
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
			['Close', 'func (s *Scope) Close(ctx context.Context) error', '第一次 Close 只运行一次清理；其他调用方等待同一完成通道或各自的 context。', 'The first Close runs cleanup once; other callers wait on one completion channel or their own context.'],
		],
		contract: [
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
			['InvalidateTarget', 'func InvalidateTarget[T any](source Observable[T], target InvalidationTarget) func()', '最小接口，供自定义原生宿主与测试替身使用，无需导入 MyGo。', 'Minimal interface for custom native hosts and test fakes, without importing MyGo.'],
		],
		file: 'window.go',
		code: `stop := nodra.InvalidateSelector(store, window, func(s State) int {
	return s.Count
})
defer stop()`,
		contract: [
			['辅助函数会注册窗口关闭监听，并返回可重复调用的停止函数。隐藏窗口保留订阅；关闭某个窗口不会关闭 Store 或影响其他窗口。', 'Helpers register a window-close listener and return an idempotent stop. Hiding preserves subscriptions; closing one window neither closes the Store nor affects others.'],
			['临时草稿和控件状态留在界面层自己管理，只用 Store.Update 提交真正编辑过的字段。', 'Keep transient drafts and widget state in the UI owner; commit only edited fields with Store.Update.'],
		],
		concurrency: ['订阅和选择器运行在写入者 goroutine 中，Window.Invalidate 会安全地把帧请求投递到主线程。其他由界面自身产生的变化仍需 Window.Update。', 'Subscriptions and selectors run on writer goroutines; Window.Invalidate safely posts the frame request to the main thread. Other UI-owned changes still require Window.Update.'],
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
