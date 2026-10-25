# Changelog

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
