# Changelog

## Unreleased

### Features

* **history:** add `History.Clear`, which drops every undo and redo step and makes the store's current value the new baseline, so after File > Open, New or Revert replaces the document Undo can no longer restore the previous one; Status publishes the empty stacks, so `nativeui.InvalidateHistory` and other Status subscribers follow without closing and re-tracking. It returns `ErrHistoryGroup` while a group is open, `ErrHistoryBatch` inside an open `Batch` and `ErrClosed` after `Close`
* **bridge:** add `Options.FrameInterval`, the least time between two frames of one watch: changes within it coalesce into the latest, which is sent when the interval ends, and a change after a quiet spell is sent at once. Zero keeps the previous behaviour, in which a page receives every change until about 1 MiB of frames is queued; docs/bridge.md suggests a display frame for fast-changing state
* **core:** add `Derived.Done`, closed by `Close`, so a bridge watch of a derived value ends with `ErrClosed` when the derived value closes
* **persistence:** add `AsyncOptions.MaxWait` to bound how long continuous updates can postpone an `AutoSaveAsync` debounced write; every update restarts the trailing debounce, so a store updated more often than `Debounce` was not written until Flush or Close. Zero keeps today's unbounded behaviour, and the postponement is now documented on `AsyncOptions` and in docs/architecture.md
* add the `nodra_nomygo` build tag for programs that use only the core and DevTools: it leaves out `Invalidate`, `InvalidateSelector`, `InvalidateSelectorWith`, `InvalidateProjection` and `InvalidateProjectionWith` so MyGo is not linked, which avoids MyGo's main-thread lock and makes a `CGO_ENABLED=0` Linux build statically linked; the default build and API are unchanged, and CI checks the tagged `examples/basic` build has no `PT_INTERP` header. docs/support.md now says the default Linux pure-Go build is dynamically linked, docs/architecture.md describes the tag, and docs/compatibility.md records it as part of the v1 contract with the v2 plan to drop MyGo from the root package
* **bridge:** let the web pages of a MyGo app follow a Store and change it through registered actions: `bridge.Expose` and `bridge.ExposeProjection` serve a source, or the part of it pages may see, through a bound service whose watch sends a frame only when the encoded value changes and coalesces to the latest while a page is slow; `bridge.Handle` registers the actions pages dispatch; `Options.AllowRead` and `Options.AllowDispatch` decide per page. A dependency-free JavaScript client (`bridge/client`, also served from Go as `bridge.Client`) fits React's `useSyncExternalStore`. See docs/bridge.md and examples/web
* **nativeui:** make a native UI window's Undo and Redo act on a History: `nativeui.UndoShortcuts` registers Cmd+Z and Shift+Cmd+Z (and Ctrl+Y on Windows) while history can act, so with MyGo 0.7.4 on macOS the Edit menu enables its items only then, and a focused text input keeps its own undo; `nativeui.InvalidateHistory` redraws the window when that changes. A separate subpackage, so apps without native UI do not link MyGo's ui package
* **async:** `ActionMetrics` gains `Superseded`, and superseded LatestWins runs are counted there instead of in `Failed`, matching the `TaskSuperseded` phase; docs/architecture.md describes the classification

### Bug Fixes

* **persistence:** an `AutoSaveAsync` writer with a `Debounce` no longer writes updates made during a save as soon as that save returns: they wait for the debounce and `MaxWait`, counted from when the save returns, so storage slower than the update rate no longer turns the writer into back-to-back writes. A zero `Debounce`, or a Flush or Close waiting on the save, still writes them at once; docs/architecture.md and the `AsyncOptions` doc say so
* **persistence:** `LoadWithRecovery` on a closed Store returns `ErrClosed` before reading anything, and a Store that closes while the primary is read returns `ErrClosed` without reading the backup, as documented; with a damaged primary it used to read and decode the backup as well and return a `RecoveryError`. An invalid configuration and cancellation still take precedence over `ErrClosed`
* **async:** retry backoff saturates instead of overflowing, so `MaxBackoff` still bounds every delay when `MaxAttempts` is large; canceling, superseding or closing during a backoff delay no longer starts one more work attempt with a canceled context
* **scope:** a late `Scope.Observe` contains a panicking stop function and returns `ErrClosed` joined with a `PanicError`, like a late `Defer`

### Bug Fixes

* **core:** an update callback that reads another store no longer deadlocks every store when an `Atomic2` or `AtomicGroup` starts meanwhile; store reads no longer take the global transaction gate, and the docs state that update callbacks may read other stores but must not write to any store
* **core:** transactions settle derived values by height, so a `Combine` whose inputs are Stores, read-only views or derived values at different depths, and anything derived from it, no longer sees a half-applied transaction, and a derived value settles on the newest snapshot of a transaction. An application `Observable` counts as height zero, so one that forwards a derived value is still not settled after it
* **core:** a `Derive` or `Combine` constructed while a transaction is being dispatched no longer loses its catch-up and stays stale
* **core:** an `Atomic2` or `AtomicGroup` commit into a store with an open batch becomes the batch's publication boundary: the batch no longer republishes the transaction's value, and each mutation's `Previous` is the value subscribers last received
* **bridge:** `Expose` and `ExposeProjection` return `ErrInvalid` for a zero `ReadOnly` (or a `Store` or `ReadOnly` without a store) instead of serving its zero value through a watch that never ends
* **core:** closing a store inside an open batch gives its last committed value a new version instead of one already published with an older value

### Bug Fixes

* **persistence:** a journaled FileStorage no longer fails every Load, WithConflictDetection and CheckedFileStorage call after a crash or failed removal during journal cleanup: cleanup now removes the commit marker before the journal, and recovery discards a commit marker that has no matching journal instead of returning "read committed journal" or "committed journal checksum mismatch"
* **persistence:** `FileStorage{Durable: true}` no longer fails every Save on Windows with an access-denied directory sync after the primary was already replaced, which also skipped the backup; Windows has no directory fsync, so the sync is skipped there, as the journal path already did
* **persistence:** `LoadWithRecovery` now falls back to the backup when the primary fails to decode with an error that wraps `ErrInvalidPersistence`, such as a damaged `VersionedJSONCodec` envelope; it used to return that load error as if the configuration were invalid and never tried the backup. Configuration errors still stop before the backup and are not reported as `RecoveryError`: the bare sentinel from a storage or codec (an empty `FileStorage` path, an `AESGCMCodec` without `Inner` or `Keys`, a `VersionedJSONCodec` with `Version` below 1) and any read-stage rejection, such as a `CheckedFileStorage` primary that became a symlink
* **persistence:** the `AutoSaveAsync` worker now turns a panic from `PersistenceOptions.Validate`, `Codec.Encode` or `Storage.Save` into a `PanicError` write failure, reported through Errors, Flush and Close with the state left pending, instead of crashing the process; synchronous Save and AutoSave still propagate panics to their caller

### Bug Fixes

* **core:** a `Combine` no longer projects or publishes a pair of input values that no committed state contained when concurrent `Atomic2` or `AtomicGroup` transactions write the same stores: settling one transaction published whatever pair the Combine held at that moment, which could hold half of another transaction, so a projector that relied on an invariant every transaction keeps, such as an index into a slice written in the same transaction, could panic. Combine now checks that both inputs still hold a pair, down to the Stores below derived inputs, before projecting it, and otherwise reads a pair both hold, recomputing derived inputs that have not caught up, so concurrent writers may see fewer intermediate values. The same check covers a write outside a transaction that reaches a Combine through one input before the other
* **core:** a `Combine` constructed while a transaction was being dispatched could start from, and keep until the dispatch ended, a pair that mixed values from before and after the transaction, because it read its two inputs one after the other and its catch-up could publish one input's new value with the other's old one; it now reads its first pair the same consistent way, including through derived inputs that have not settled the transaction yet

### Dependencies

* support MyGo 0.7.4; go.mod now requires it instead of 0.6.3 (by way of 0.7.2 and 0.7.3), and the invalidation helpers keep the same signatures and behaviour

### Documentation

* **bridge:** docs/bridge.md declares the `PageState` its Go example exposes, with the `json:"count"` tag its React example reads as `state?.count`; an untagged field is sent as `Count`, so a page written from the document showed its placeholder forever. A test checks that the page reads keys the documented type is sent with
* the copy-on-write `UpdateErr` example in docs/architecture.md copies the map with `make` and `maps.Copy` instead of `maps.Clone`, which returns nil for a nil map, so the example no longer panics on its first update of a zero-value state; a test now runs the example and keeps the document identical to it
* **core:** state that a derived value with a nil equals function notifies on every accepted source version, and that an equality function is what filters unchanged values
* document every root-package function in Go doc; index `Define`, `DefineWithClone`, `Definition`, `BindField`, `Field`, `Derived`, `NewMemoryKeyProvider`, `NewScope` and the MyGo invalidation helpers in docs/public-api.md; list the methods of every re-exported type under that type in docs/public-api.md and in the package comment, including ReadOnly, the MemoryKeyProvider key-rotation methods, FileStorage, CheckedFileStorage and the codecs, because Go doc shows no methods for the root aliases; remove a duplicated paragraph from docs/architecture.md. No API or behaviour changes
* record v1.0.0 publication checks, checksums, API evidence, and package documentation; activate the published v1 compatibility policy
* state in docs/architecture.md and the Go docs that `SubscribeMetrics` callbacks are not covered by `Task.Wait` or `AsyncAction.Close` and can run after unsubscribing, and that `Metrics` read after `Close` is the final aggregate

## [1.4.0](https://github.com/yldm-tech/nodra/compare/v1.3.0...v1.4.0) (2026-11-13)


### Features

* **bridge:** space a watch's frames out with Options.FrameInterval ([#112](https://github.com/yldm-tech/nodra/issues/112)) ([8a1ce63](https://github.com/yldm-tech/nodra/commit/8a1ce632e79f1dd08db0339f51c23889e64d53f8))

## [1.3.0](https://github.com/yldm-tech/nodra/compare/v1.2.0...v1.3.0) (2026-11-13)


### Features

* add the nodra_nomygo build tag to build without MyGo ([#107](https://github.com/yldm-tech/nodra/issues/107)) ([6d0609e](https://github.com/yldm-tech/nodra/commit/6d0609eb18d9e6b911fc5a0ce9a0ee5ec966d77b))
* **persistence:** bound AutoSaveAsync debounce with MaxWait and fix journal, Windows durability, recovery and worker panic bugs ([#103](https://github.com/yldm-tech/nodra/issues/103)) ([88900fa](https://github.com/yldm-tech/nodra/commit/88900fa786dc8a9acf2a40823884d37313ccb0cf))


### Bug Fixes

* **async:** bound retry backoff, count superseded runs, and contain late Observe panics ([#104](https://github.com/yldm-tech/nodra/issues/104)) ([eb32678](https://github.com/yldm-tech/nodra/commit/eb32678dcb4de87f535bce8ddea5a9e5da79af6d))
* **bridge:** close audit findings in the bridge and its JavaScript client ([#105](https://github.com/yldm-tech/nodra/issues/105)) ([f55a935](https://github.com/yldm-tech/nodra/commit/f55a93593b6ce9e56f3a78b52acd5f1fc3b88cf3))
* **core:** make transactions, batches and derived values settle consistently ([#101](https://github.com/yldm-tech/nodra/issues/101)) ([1052dd9](https://github.com/yldm-tech/nodra/commit/1052dd9dceed76c806c0ec44b2e6facbdbfc0736))
* **devtools:** fix search, export copies, redaction and restore size limits ([#106](https://github.com/yldm-tech/nodra/issues/106)) ([97b8f00](https://github.com/yldm-tech/nodra/commit/97b8f0010f50ae1d3a17e3a888ed1bf7afd28850))
* **history:** align CanUndo/CanRedo with HistoryState and document store-close ownership ([#102](https://github.com/yldm-tech/nodra/issues/102)) ([d74070a](https://github.com/yldm-tech/nodra/commit/d74070aa32c9fdbac40df7ef7d4178455772efe9))

## [1.2.0](https://github.com/yldm-tech/nodra/compare/v1.1.0...v1.2.0) (2026-11-06)


### Features

* **bridge:** let web pages follow a Store and dispatch actions ([#94](https://github.com/yldm-tech/nodra/issues/94)) ([1f106b7](https://github.com/yldm-tech/nodra/commit/1f106b7055f8357250278a7088e2742616618f7a))
* **nativeui:** make a window's Undo and Redo act on a History ([#95](https://github.com/yldm-tech/nodra/issues/95)) ([134d213](https://github.com/yldm-tech/nodra/commit/134d2133c3f09e854bd98e7c976a8de49a527e5b))

## [1.1.0](https://github.com/yldm-tech/nodra/compare/v1.0.0...v1.1.0) (2026-10-25)


### Features

* **site:** add the official website ([#75](https://github.com/yldm-tech/nodra/issues/75)) ([c3449ea](https://github.com/yldm-tech/nodra/commit/c3449ea90afb0f3ec6a4e851ad90865e44fad094))

## [1.0.0](https://github.com/yldm-tech/nodra/compare/v0.13.0...v1.0.0) (2026-10-25)

### Stable API

* stabilize the root-package and DevTools Go APIs with explicit ownership, transaction, callback, persistence, and shutdown contracts
* support Go 1.27.3 and MyGo 0.6.3, with native Linux/macOS/Windows amd64 tests, native macOS arm64 tests, and pure-Go Linux/Windows arm64 cross-builds

### Bug Fixes

* propagate original Atomic2 and AtomicGroup notification panics after commit, while preserving rollback and PanicError results for preparation failures
* correct the API reference for Store.Close, shallow ownership, sealed readers, conflict detection, async persistence, and task completion
* **ci:** verify aliased APIs and exact release commits ([#71](https://github.com/yldm-tech/nodra/issues/71)) ([ca6732a](https://github.com/yldm-tech/nodra/commit/ca6732a98ba6e0eca909a3dc4709914b9248184d))

### Verification

* accept public state, persistence, actions, and DevTools from an independent consumer module with race detection
* detect incompatible methods, fields, and interfaces behind public type aliases and compare with the latest published API
* validate release tags and manifests before testing the exact released commit through the public Go proxy and checksum service
* record the release gate in docs/release-1.0.md and verify DevTools in a browser with pinned Playwright

## [0.13.0](https://github.com/yldm-tech/nodra/compare/v0.12.1...v0.13.0) (2026-10-25)


### Features

* **example:** add workbench localization ([#66](https://github.com/yldm-tech/nodra/issues/66)) ([fec2b9e](https://github.com/yldm-tech/nodra/commit/fec2b9e49b5638737830e9b7cd2861ee0f4e1892))

## [0.12.1](https://github.com/yldm-tech/nodra/compare/v0.12.0...v0.12.1) (2026-10-25)


### Bug Fixes

* guard nil memory key providers ([#55](https://github.com/yldm-tech/nodra/issues/55)) ([15d7e72](https://github.com/yldm-tech/nodra/commit/15d7e725302b4ceb68f8693711653d3a92b0f120))
* include custom data in inspector budget ([#60](https://github.com/yldm-tech/nodra/issues/60)) ([90d2a36](https://github.com/yldm-tech/nodra/commit/90d2a36f635523959c9bfefd8ea78203ce04fbd0))
* make zero read-only views safe ([#56](https://github.com/yldm-tech/nodra/issues/56)) ([dfdd0c3](https://github.com/yldm-tech/nodra/commit/dfdd0c38143c597828a7a3a391053879ab337d0b))
* reject all typed-nil invalidation targets ([#54](https://github.com/yldm-tech/nodra/issues/54)) ([bdfecc9](https://github.com/yldm-tech/nodra/commit/bdfecc9152b63f8e92e1d08446570d4f8bb3d5df))
* reject empty encryption key IDs ([#58](https://github.com/yldm-tech/nodra/issues/58)) ([a1b5e8b](https://github.com/yldm-tech/nodra/commit/a1b5e8b36951d6dc445b9e8dcd6ddbf23abe3389))

## [0.12.0](https://github.com/yldm-tech/nodra/compare/v0.11.1...v0.12.0) (2026-10-25)


### ⚠ BREAKING CHANGES

* move implementation behind root facade ([#50](https://github.com/yldm-tech/nodra/issues/50))

### Bug Fixes

* normalize nil file storage contexts ([#52](https://github.com/yldm-tech/nodra/issues/52)) ([3c87333](https://github.com/yldm-tech/nodra/commit/3c87333fd0515803c62656934313737a82ea138e))
* reject typed-nil persistence dependencies ([#51](https://github.com/yldm-tech/nodra/issues/51)) ([1615043](https://github.com/yldm-tech/nodra/commit/1615043afbc7d6f20c8be6a8f5fb2d64b57ecac8))


### Code Refactoring

* move implementation behind root facade ([#50](https://github.com/yldm-tech/nodra/issues/50)) ([cfd230e](https://github.com/yldm-tech/nodra/commit/cfd230e2c844376b682099b8cc42c165aafa42db))

## [0.11.1](https://github.com/yldm-tech/nodra/compare/v0.11.0...v0.11.1) (2026-10-25)


### Bug Fixes

* **file:** discard uncommitted journal records ([#46](https://github.com/yldm-tech/nodra/issues/46)) ([ef0645f](https://github.com/yldm-tech/nodra/commit/ef0645fc14d9697550a9b6275cd53a64f8f841f9))
* **invalidate:** ignore typed nil targets ([#49](https://github.com/yldm-tech/nodra/issues/49)) ([692a5ea](https://github.com/yldm-tech/nodra/commit/692a5eaa1f82e5fcfe1ea0271b43432ca4bd0bbd))
* **persistence:** reset debounce on latest request ([#48](https://github.com/yldm-tech/nodra/issues/48)) ([556672f](https://github.com/yldm-tech/nodra/commit/556672f799b100103dbfb1a6268e9454d6a11a65))

## [0.11.0](https://github.com/yldm-tech/nodra/compare/v0.10.0...v0.11.0) (2026-10-25)


### ⚠ BREAKING CHANGES

* compose derived state, debounce actions and bound diagnostic costs ([#25](https://github.com/yldm-tech/nodra/issues/25))
* integrate read-only state, task scheduling and recovery diagnostics ([#23](https://github.com/yldm-tech/nodra/issues/23))
* the import path github.com/yldm-tech/nodra/mygo is removed. Replace mygo.SubscribeInvalidation(store, window) from that package with nodra.Invalidate(store, window) from github.com/yldm-tech/nodra.
* publish as one module at the repository root, github.com/yldm-tech/nodra ([#8](https://github.com/yldm-tech/nodra/issues/8))

### Features

* add dependency-free metrics sink ([#35](https://github.com/yldm-tech/nodra/issues/35)) ([6c58848](https://github.com/yldm-tech/nodra/commit/6c588482877dcabfbef2140ef2bb786ff9acb2a9))
* add journaled file storage recovery ([#34](https://github.com/yldm-tech/nodra/issues/34)) ([806ecc6](https://github.com/yldm-tech/nodra/commit/806ecc618a14fcdd59cb0783062654e8b3404d1b))
* add save conflict detection and production state diagnostics ([#27](https://github.com/yldm-tech/nodra/issues/27)) ([27b7b99](https://github.com/yldm-tech/nodra/commit/27b7b99ad1c90cd36f818d79de3e184e370f2521))
* add selector-based MyGo invalidation with automatic cleanup ([#19](https://github.com/yldm-tech/nodra/issues/19)) ([dc7cb70](https://github.com/yldm-tech/nodra/commit/dc7cb70ffbf8806014137eb931fddab3719c49f1))
* add state isolation, migrations, async persistence, and DevTools ([#12](https://github.com/yldm-tech/nodra/issues/12)) ([3b652d2](https://github.com/yldm-tech/nodra/commit/3b652d28de2ee2c8a4778a29c132852149f9c289))
* complete async state, history and DevTools workflows ([#21](https://github.com/yldm-tech/nodra/issues/21)) ([e2d780d](https://github.com/yldm-tech/nodra/commit/e2d780dbfe88029c23b18782436f01b46c8614ec))
* compose derived state, debounce actions and bound diagnostic costs ([#25](https://github.com/yldm-tech/nodra/issues/25)) ([797a848](https://github.com/yldm-tech/nodra/commit/797a848b375772cb9aea85fccf6f084aad1797b2))
* expose generic invalidation targets ([#37](https://github.com/yldm-tech/nodra/issues/37)) ([566514c](https://github.com/yldm-tech/nodra/commit/566514c2db696d63a63981639b789544d275e07d))
* harden transactions retries persistence and DevTools ([#29](https://github.com/yldm-tech/nodra/issues/29)) ([ae923af](https://github.com/yldm-tech/nodra/commit/ae923affcfdc96827295aff19d93389ab2c13d3c))
* integrate read-only state, task scheduling and recovery diagnostics ([#23](https://github.com/yldm-tech/nodra/issues/23)) ([c317a3e](https://github.com/yldm-tech/nodra/commit/c317a3eb453927783b62cc252afb74a47fb2c48c))
* move MyGo invalidation into the root package as nodra.Invalidate ([#10](https://github.com/yldm-tech/nodra/issues/10)) ([3ed08e7](https://github.com/yldm-tech/nodra/commit/3ed08e7405971f75aa126695d819554a63cf2ebd))
* **persistence:** add explicit backup recovery ([#38](https://github.com/yldm-tech/nodra/issues/38)) ([d5f5506](https://github.com/yldm-tech/nodra/commit/d5f550604e30928338d42d414ba198365c8f496e))
* publish as one module at the repository root, github.com/yldm-tech/nodra ([#8](https://github.com/yldm-tech/nodra/issues/8)) ([e9ace62](https://github.com/yldm-tech/nodra/commit/e9ace62137e470e36a92d2e94580ef72058a883f))
* rotate DevTools tokens and limit requests ([#36](https://github.com/yldm-tech/nodra/issues/36)) ([7e438a8](https://github.com/yldm-tech/nodra/commit/7e438a826442782f3c0289cc662278a1e338decb))
* settle derived state across atomic transactions ([#31](https://github.com/yldm-tech/nodra/issues/31)) ([f2ac9e0](https://github.com/yldm-tech/nodra/commit/f2ac9e057e739cab70ec6dbb2d3db814d055a56c))
* **store:** add error-returning batches ([#40](https://github.com/yldm-tech/nodra/issues/40)) ([84bea3c](https://github.com/yldm-tech/nodra/commit/84bea3ce33251d6a865a498f0ad4674330a44fbd))
* support collection selectors and bounded large-state DevTools ([#16](https://github.com/yldm-tech/nodra/issues/16)) ([8938323](https://github.com/yldm-tech/nodra/commit/893832330aa4975bd7b6aa3f915bd1675ea5e086))


### Bug Fixes

* preserve reactive versions and history commit consistency ([#14](https://github.com/yldm-tech/nodra/issues/14)) ([eed35be](https://github.com/yldm-tech/nodra/commit/eed35be1ac6147edf34321013dde18a501e9da6c))
* **store:** preserve Patch errors ([#39](https://github.com/yldm-tech/nodra/issues/39)) ([60dd825](https://github.com/yldm-tech/nodra/commit/60dd825da297f64a260b28731f72b876539c40be))

## [0.10.0](https://github.com/yldm-tech/nodra/compare/v0.9.0...v0.10.0) (2026-10-25)


### Features

* **store:** add error-returning batches ([#40](https://github.com/yldm-tech/nodra/issues/40)) ([84bea3c](https://github.com/yldm-tech/nodra/commit/84bea3ce33251d6a865a498f0ad4674330a44fbd))


### Bug Fixes

* **store:** preserve Patch errors ([#39](https://github.com/yldm-tech/nodra/issues/39)) ([60dd825](https://github.com/yldm-tech/nodra/commit/60dd825da297f64a260b28731f72b876539c40be))

## [0.9.0](https://github.com/yldm-tech/nodra/compare/v0.8.0...v0.9.0) (2026-10-25)


### ⚠ BREAKING CHANGES

* compose derived state, debounce actions and bound diagnostic costs ([#25](https://github.com/yldm-tech/nodra/issues/25))
* integrate read-only state, task scheduling and recovery diagnostics ([#23](https://github.com/yldm-tech/nodra/issues/23))
* the import path github.com/yldm-tech/nodra/mygo is removed. Replace mygo.SubscribeInvalidation(store, window) from that package with nodra.Invalidate(store, window) from github.com/yldm-tech/nodra.
* publish as one module at the repository root, github.com/yldm-tech/nodra ([#8](https://github.com/yldm-tech/nodra/issues/8))

### Features

* add save conflict detection and production state diagnostics ([#27](https://github.com/yldm-tech/nodra/issues/27)) ([27b7b99](https://github.com/yldm-tech/nodra/commit/27b7b99ad1c90cd36f818d79de3e184e370f2521))
* add selector-based MyGo invalidation with automatic cleanup ([#19](https://github.com/yldm-tech/nodra/issues/19)) ([dc7cb70](https://github.com/yldm-tech/nodra/commit/dc7cb70ffbf8806014137eb931fddab3719c49f1))
* add state isolation, migrations, async persistence, and DevTools ([#12](https://github.com/yldm-tech/nodra/issues/12)) ([3b652d2](https://github.com/yldm-tech/nodra/commit/3b652d28de2ee2c8a4778a29c132852149f9c289))
* complete async state, history and DevTools workflows ([#21](https://github.com/yldm-tech/nodra/issues/21)) ([e2d780d](https://github.com/yldm-tech/nodra/commit/e2d780dbfe88029c23b18782436f01b46c8614ec))
* compose derived state, debounce actions and bound diagnostic costs ([#25](https://github.com/yldm-tech/nodra/issues/25)) ([797a848](https://github.com/yldm-tech/nodra/commit/797a848b375772cb9aea85fccf6f084aad1797b2))
* expose generic invalidation targets ([#37](https://github.com/yldm-tech/nodra/issues/37)) ([566514c](https://github.com/yldm-tech/nodra/commit/566514c2db696d63a63981639b789544d275e07d))
* harden transactions retries persistence and DevTools ([#29](https://github.com/yldm-tech/nodra/issues/29)) ([ae923af](https://github.com/yldm-tech/nodra/commit/ae923affcfdc96827295aff19d93389ab2c13d3c))
* integrate read-only state, task scheduling and recovery diagnostics ([#23](https://github.com/yldm-tech/nodra/issues/23)) ([c317a3e](https://github.com/yldm-tech/nodra/commit/c317a3eb453927783b62cc252afb74a47fb2c48c))
* move MyGo invalidation into the root package as nodra.Invalidate ([#10](https://github.com/yldm-tech/nodra/issues/10)) ([3ed08e7](https://github.com/yldm-tech/nodra/commit/3ed08e7405971f75aa126695d819554a63cf2ebd))
* **persistence:** add explicit backup recovery ([#38](https://github.com/yldm-tech/nodra/issues/38)) ([d5f5506](https://github.com/yldm-tech/nodra/commit/d5f550604e30928338d42d414ba198365c8f496e))
* publish as one module at the repository root, github.com/yldm-tech/nodra ([#8](https://github.com/yldm-tech/nodra/issues/8)) ([e9ace62](https://github.com/yldm-tech/nodra/commit/e9ace62137e470e36a92d2e94580ef72058a883f))
* settle derived state across atomic transactions ([#31](https://github.com/yldm-tech/nodra/issues/31)) ([f2ac9e0](https://github.com/yldm-tech/nodra/commit/f2ac9e057e739cab70ec6dbb2d3db814d055a56c))
* support collection selectors and bounded large-state DevTools ([#16](https://github.com/yldm-tech/nodra/issues/16)) ([8938323](https://github.com/yldm-tech/nodra/commit/893832330aa4975bd7b6aa3f915bd1675ea5e086))


### Bug Fixes

* preserve reactive versions and history commit consistency ([#14](https://github.com/yldm-tech/nodra/issues/14)) ([eed35be](https://github.com/yldm-tech/nodra/commit/eed35be1ac6147edf34321013dde18a501e9da6c))

## [0.8.0](https://github.com/yldm-tech/nodra/compare/v0.7.0...v0.8.0) (2026-10-25)


### Features

* harden transactions retries persistence and DevTools ([#29](https://github.com/yldm-tech/nodra/issues/29)) ([ae923af](https://github.com/yldm-tech/nodra/commit/ae923affcfdc96827295aff19d93389ab2c13d3c))

## [0.7.0](https://github.com/yldm-tech/nodra/compare/v0.6.0...v0.7.0) (2026-10-24)


### ⚠ BREAKING CHANGES

* compose derived state, debounce actions and bound diagnostic costs ([#25](https://github.com/yldm-tech/nodra/issues/25))

### Features

* compose derived state, debounce actions and bound diagnostic costs ([#25](https://github.com/yldm-tech/nodra/issues/25)) ([797a848](https://github.com/yldm-tech/nodra/commit/797a848b375772cb9aea85fccf6f084aad1797b2))

## [0.6.0](https://github.com/yldm-tech/nodra/compare/v0.5.0...v0.6.0) (2026-10-24)


### Features

* complete async state, history and DevTools workflows ([#21](https://github.com/yldm-tech/nodra/issues/21)) ([e2d780d](https://github.com/yldm-tech/nodra/commit/e2d780dbfe88029c23b18782436f01b46c8614ec))

## [0.5.0](https://github.com/yldm-tech/nodra/compare/v0.4.0...v0.5.0) (2026-10-24)


### Features

* add selector-based MyGo invalidation with automatic cleanup ([#19](https://github.com/yldm-tech/nodra/issues/19)) ([dc7cb70](https://github.com/yldm-tech/nodra/commit/dc7cb70ffbf8806014137eb931fddab3719c49f1))

## [0.4.0](https://github.com/yldm-tech/nodra/compare/v0.3.0...v0.4.0) (2026-10-24)


### Features

* support collection selectors and bounded large-state DevTools ([#16](https://github.com/yldm-tech/nodra/issues/16)) ([8938323](https://github.com/yldm-tech/nodra/commit/893832330aa4975bd7b6aa3f915bd1675ea5e086))

## [0.3.0](https://github.com/yldm-tech/nodra/compare/v0.2.0...v0.3.0) (2026-10-24)


### Features

* add state isolation, migrations, async persistence, and DevTools ([#12](https://github.com/yldm-tech/nodra/issues/12)) ([3b652d2](https://github.com/yldm-tech/nodra/commit/3b652d28de2ee2c8a4778a29c132852149f9c289))


### Bug Fixes

* preserve reactive versions and history commit consistency ([#14](https://github.com/yldm-tech/nodra/issues/14)) ([eed35be](https://github.com/yldm-tech/nodra/commit/eed35be1ac6147edf34321013dde18a501e9da6c))

## [0.2.0](https://github.com/yldm-tech/nodra/compare/v0.1.0...v0.2.0) (2026-10-21)


### ⚠ BREAKING CHANGES

* the import path github.com/yldm-tech/nodra/mygo is removed. Replace mygo.SubscribeInvalidation(store, window) from that package with nodra.Invalidate(store, window) from github.com/yldm-tech/nodra.

### Features

* move MyGo invalidation into the root package as nodra.Invalidate ([#10](https://github.com/yldm-tech/nodra/issues/10)) ([3ed08e7](https://github.com/yldm-tech/nodra/commit/3ed08e7405971f75aa126695d819554a63cf2ebd))

## 0.1.0 (2026-10-21)


### ⚠ BREAKING CHANGES

* publish as one module at the repository root, github.com/yldm-tech/nodra ([#8](https://github.com/yldm-tech/nodra/issues/8))

### Features

* publish as one module at the repository root, github.com/yldm-tech/nodra ([#8](https://github.com/yldm-tech/nodra/issues/8)) ([e9ace62](https://github.com/yldm-tech/nodra/commit/e9ace62137e470e36a92d2e94580ef72058a883f))

## [packages/nodra 0.1.0 and packages/mygo 0.1.0](https://github.com/yldm-tech/nodra/releases/tag/packages%2Fnodra%2Fv0.1.0) (2026-10-21)

Superseded. These were released as two separate modules, `github.com/yldm-tech/mygo-nodra/packages/nodra` (tag `packages/nodra/v0.1.0`) and `github.com/yldm-tech/mygo-nodra/packages/mygo` (tag `packages/mygo/v0.1.0`), from the repository before it was renamed to `yldm-tech/nodra`. The code now lives in the single module `github.com/yldm-tech/nodra`: import the core as `github.com/yldm-tech/nodra` and the MyGo adapter as `github.com/yldm-tech/nodra/mygo`. The old tags stay in place because the Go module proxy has cached them, but they receive no further releases.

### Highlights

- Added store lifecycle and `ErrClosed`.
- Added immediate and equality-aware subscriptions.
- Added rollback-safe `UpdateErr`.
- Added nested batches, bounded undo/redo history and pluggable persistence.
- Added observable autosave failures and store-closing watch channels.
- Added JSONCodec, atomic FileStorage, and a runnable native MyGo example.
- Added Pinia/Zustand-style Create, Define, State, Patch, Reset, Derive, selector previous values, named actions, mutation metadata, and action lifecycle hooks.
- Added the MyGo adapter, `SubscribeInvalidation`, which requests a new native frame when a store publishes.
