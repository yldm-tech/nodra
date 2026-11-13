# Supported environments

This matrix is the release contract for Nodra 1.0. CI is expected to exercise the rows marked as tested; other combinations may work but are not release targets until added here.

| Component | Supported baseline | Verification |
| --- | --- | --- |
| Go toolchain | Go 1.27.3 or newer within the same language/module compatibility line | `go.mod`, test/vet/lint/module jobs |
| MyGo integration | MyGo 0.7.4 | `go.mod`, native examples and invalidation tests |
| Bridge client | Pages in MyGo's web views, which run ES modules; no Node or browser outside MyGo | `node --test bridge/client/nodra.test.mjs` on Node 24; the bridge's Go side in the race and vet matrix |
| Edit menu integration (`nativeui`) | MyGo 0.7.4: the keys on every platform, and on macOS the Edit menu following history | `nativeui` and `examples/mygo` UI-tester tests in the race and vet matrix |
| Operating systems | Linux, macOS, Windows | race and vet matrix |
| Architectures | `amd64` on Linux/macOS/Windows; native macOS arm64; pure-Go Linux and Windows arm64 builds | explicit native architecture checks and race/vet matrix on `ubuntu-latest`, `macos-26-intel`, `macos-26`, and `windows-latest`; Linux/Windows arm64 cross-builds |
| CGO | `CGO_ENABLED=0` builds compile on every listed target; CGO is not required by Nodra. On Linux the default build still links MyGo, whose purego fallback makes the executable dynamically linked against glibc; build with `-tags nodra_nomygo` for a static executable | Linux CI pure-Go builds; `examples/basic` built with `-tags nodra_nomygo` must have no `PT_INTERP` header |
| Filesystems | Local regular files for `FileStorage`; advisory locking is cooperative | persistence and conflict-detection tests |

Nodra does not promise network-filesystem locking, hard-link alias detection, or arbitrary uncoordinated writers for `CheckedFileStorage`. The public API is usable without a running MyGo application, but the module intentionally keeps MyGo as a dependency for the invalidation helpers. Importing the root package therefore links MyGo by default, including its `init`, which locks the main goroutine to its OS thread. Programs that use only the core and DevTools can build with `-tags nodra_nomygo` to leave out `Invalidate`, `InvalidateSelector`, `InvalidateSelectorWith`, `InvalidateProjection` and `InvalidateProjectionWith` and drop MyGo from the binary; `InvalidateTarget` remains available. The `bridge` and `nativeui` packages need MyGo and do not build without it. macOS arm64 runs native tests; the Linux/Windows arm64 build rows are compile verification only and do not promise native runtime testing.

When a supported toolchain or MyGo version changes, update this file, the compatibility policy, and CI in the same pull request. A release must not rely on an undocumented local toolchain.
