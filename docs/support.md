# Supported environments

This matrix is the release contract for Nodra 1.0. CI is expected to exercise the rows marked as tested; other combinations may work but are not release targets until added here.

| Component | Supported baseline | Verification |
| --- | --- | --- |
| Go toolchain | Go 1.27.3 or newer within the same language/module compatibility line | `go.mod`, test/vet/lint/module jobs |
| MyGo integration | MyGo 0.6.3 | `go.mod`, native examples and invalidation tests |
| Operating systems | Linux, macOS, Windows | race and vet matrix |
| Architectures | `amd64` on the OS test matrix; `arm64` for pure-Go Linux and Windows builds | native CI runners where available; explicit Linux/Windows arm64 cross-builds |
| CGO | Pure-Go builds are supported with `CGO_ENABLED=0`; CGO is not required by Nodra | Linux CI pure-Go build |
| Filesystems | Local regular files for `FileStorage`; advisory locking is cooperative | persistence and conflict-detection tests |

Nodra does not promise network-filesystem locking, hard-link alias detection, or arbitrary uncoordinated writers for `CheckedFileStorage`. The public API is usable without a running MyGo application, but the module intentionally keeps MyGo as a dependency for the invalidation helpers.

When a supported toolchain or MyGo version changes, update this file, the compatibility policy, and CI in the same pull request. A release must not rely on an undocumented local toolchain.
