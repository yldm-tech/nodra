# Supported environments

This matrix is the release contract for Nodra 1.0. CI is expected to exercise the rows marked as tested; other combinations may work but are not release targets until added here.

| Component | Supported baseline | Verification |
| --- | --- | --- |
| Go toolchain | Go 1.27.3 or newer within the same language/module compatibility line | `go.mod`, test/vet/lint/module jobs |
| MyGo integration | MyGo 0.6.3 | `go.mod`, native examples and invalidation tests |
| Operating systems | Linux, macOS, Windows | race and vet matrix |
| Architectures | `amd64` on Linux/macOS/Windows; native macOS arm64; pure-Go Linux and Windows arm64 builds | explicit native architecture checks and race/vet matrix on `ubuntu-latest`, `macos-26-intel`, `macos-26`, and `windows-latest`; Linux/Windows arm64 cross-builds |
| CGO | Pure-Go builds are supported with `CGO_ENABLED=0`; CGO is not required by Nodra | Linux CI pure-Go build |
| Filesystems | Local regular files for `FileStorage`; advisory locking is cooperative | persistence and conflict-detection tests |

Nodra does not promise network-filesystem locking, hard-link alias detection, or arbitrary uncoordinated writers for `CheckedFileStorage`. The public API is usable without a running MyGo application, but the module intentionally keeps MyGo as a dependency for the invalidation helpers. macOS arm64 runs native tests; the Linux/Windows arm64 build rows are compile verification only and do not promise native runtime testing.

When a supported toolchain or MyGo version changes, update this file, the compatibility policy, and CI in the same pull request. A release must not rely on an undocumented local toolchain.
