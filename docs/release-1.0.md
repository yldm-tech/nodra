# Nodra 1.0 release checklist

This checklist is the release gate for `github.com/yldm-tech/nodra` v1.0.0. It records the evidence needed for the release; it is not an upgrade guide.

## Before the release PR

- [x] The exported root API is frozen and reviewed against [`docs/public-api.md`](public-api.md).
- [x] The compatibility policy and supported environment matrix have been reviewed without unresolved changes.
- [x] The independent consumer test covers the root package and DevTools from a separate module: `bash scripts/check-consumer.sh`.
- [x] The full local checks pass: `go test ./...`, `go test -race ./...`, `go vet ./...`, `golangci-lint run ./...`, and `go mod tidy -diff`.
- [x] API comparison against the previous release reports no unreviewed incompatible change.
- [x] The DevTools browser smoke test passes with the pinned Playwright version.
- [x] The performance comparison and govulncheck jobs have completed successfully.

## Release PR and tag

- [x] Set the release manifest to `1.0.0` only in the release PR after the API review is signed off.
- [x] Confirm the release PR contains the final changelog entry and no unrelated feature changes.
- [x] Merge the release PR and verify that the `v1.0.0` tag points at its release commit.
- [x] Confirm the tag resolves through `proxy.golang.org` and `sum.golang.org`.
- [x] Run `bash scripts/check-consumer.sh v1.0.0` from an independent module with `GOWORK=off`.

## After publication

- [x] Verify `go list -m github.com/yldm-tech/nodra@v1.0.0` and `go mod download github.com/yldm-tech/nodra@v1.0.0` through the public proxy.
- [x] Verify the release page, checksum, and changelog comparison link.
- [x] Verify versioned root-package and DevTools documentation on pkg.go.dev.
- [x] Record the final API comparison, consumer result, support matrix, test runs, security result, and proxy result in the release review.

## Support boundary

The release supports Go 1.27.3 or newer within the same language/module compatibility line, MyGo 0.6.3, Linux/macOS/Windows on amd64, native macOS arm64, and pure-Go Linux/Windows arm64 builds. macOS arm64 runs native race and vet tests; Linux/Windows arm64 are cross-build verified without a native runtime testing claim. File persistence support is limited to local regular files with the documented cooperative locking contract.

## Candidate review

[Release PR #72](https://github.com/yldm-tech/nodra/pull/72) contains the final candidate and its CI evidence. The API job retains a `public-api-<commit>` artifact with root, defining-type, and DevTools export data plus the commit, toolchain, and comparison-tool versions. The API review includes source ownership, Batch versus transaction rollback, atomic preparation versus notification panics, repeated Close results, sealed Reader versus application-defined Observable implementations, asynchronous task completion, persistence drain barriers, and native runner architectures. The candidate adds independent consumer regressions for preparation panics, commit-clone panics, and post-commit snapshot/mutation panics for both Atomic2 and AtomicGroup.

Candidate `2230de1` passed the [test matrix, API comparison, independent consumer, and DevTools browser checks](https://github.com/yldm-tech/nodra/actions/runs/40178082883), the [performance comparison](https://github.com/yldm-tech/nodra/actions/runs/40178082905), and [govulncheck](https://github.com/yldm-tech/nodra/actions/runs/40178082891). The reviewed API artifact is `public-api-e311e665f098af3a5ab582f3c4b31618905dbd40`, where the suffix identifies the tested merge commit.

## Publication review

[Release v1.0.0](https://github.com/yldm-tech/nodra/releases/tag/v1.0.0) was published on 2026-10-25 from merged PR #72. The tag, release target, and public proxy origin all identify commit `d351cbfa356ee53a2587874d6630e94ceeea80c1`; its release manifest records `1.0.0`. The [changelog comparison](https://github.com/yldm-tech/nodra/compare/v0.13.0...v1.0.0) resolves successfully.

The [post-release workflow](https://github.com/yldm-tech/nodra/actions/runs/40182458864) passed tag/manifest validation and tested the exact release commit. Both CI and a local Go 1.27.3 macOS arm64 run downloaded `v1.0.0` through `GOPROXY=https://proxy.golang.org` with `GOSUMDB=sum.golang.org`, an initially empty module cache, and workspace wiring disabled. The independent consumer passed with race detection and no local replacement. The public checksum database records:

```text
github.com/yldm-tech/nodra v1.0.0 h1:SPXH/3XwofFQSrbHF85ARKOxvU4X/J1bTSKjX7sbqms=
github.com/yldm-tech/nodra v1.0.0/go.mod h1:Xv8gzEzE8B2noBatHYTLGloXenXxabR6i9HFeHcD4Yk=
```

The release commit's [test workflow](https://github.com/yldm-tech/nodra/actions/runs/40182458848) passed API comparisons against its base and the latest published release, an independent consumer, module tidy, lint, the native race/vet matrix in [support.md](support.md), Linux/Windows arm64 cross-builds, persistence decoding fuzzing, and the DevTools browser smoke test. Its retained API artifact is [`public-api-d351cbfa356ee53a2587874d6630e94ceeea80c1`](https://github.com/yldm-tech/nodra/actions/runs/40182458848/artifacts/12594871664); the downloaded artifact's commit and toolchain records identify the exact release commit and Go 1.27.3. The release commit also passed [govulncheck](https://github.com/yldm-tech/nodra/actions/runs/40182458885).

The versioned [root-package documentation](https://pkg.go.dev/github.com/yldm-tech/nodra@v1.0.0) and [DevTools documentation](https://pkg.go.dev/github.com/yldm-tech/nodra@v1.0.0/devtools) both return HTTP 200 and identify v1.0.0. The documentation pages were initially unavailable; both pages were verified after requesting pkg.go.dev indexing.
