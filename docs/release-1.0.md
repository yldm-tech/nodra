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
- [ ] Merge the release PR and verify that the `v1.0.0` tag points at its release commit.
- [ ] Confirm the tag resolves through `proxy.golang.org` and `sum.golang.org`.
- [ ] Run `bash scripts/check-consumer.sh v1.0.0` from an independent module with `GOWORK=off`.

## After publication

- [ ] Verify `go list -m github.com/yldm-tech/nodra@v1.0.0` and `go mod download github.com/yldm-tech/nodra@v1.0.0` through the public proxy.
- [ ] Verify the release page, checksum, changelog links, and package documentation.
- [ ] Record the final API comparison, consumer result, support matrix, test runs, security result, and proxy result in the release review.

## Support boundary

The release supports Go 1.27.3 or newer within the same language/module compatibility line, MyGo 0.6.3, Linux/macOS/Windows on amd64, native macOS arm64, and pure-Go Linux/Windows arm64 builds. macOS arm64 runs native race and vet tests; Linux/Windows arm64 are cross-build verified without a native runtime testing claim. File persistence support is limited to local regular files with the documented cooperative locking contract.

## Candidate review

[Release PR #72](https://github.com/yldm-tech/nodra/pull/72) contains the final candidate and its CI evidence. The API job retains a `public-api-<commit>` artifact with root, defining-type, and DevTools export data plus the commit, toolchain, and comparison-tool versions. The API review includes source ownership, Batch versus transaction rollback, atomic preparation versus notification panics, repeated Close results, sealed Reader versus application-defined Observable implementations, asynchronous task completion, persistence drain barriers, and native runner architectures. The candidate adds independent consumer regressions for preparation panics, commit-clone panics, and post-commit snapshot/mutation panics for both Atomic2 and AtomicGroup. Publication, proxy, checksum, and final documentation checks remain unchecked until `v1.0.0` exists.

Candidate `2230de1` passed the [test matrix, API comparison, independent consumer, and DevTools browser checks](https://github.com/yldm-tech/nodra/actions/runs/40178082883), the [performance comparison](https://github.com/yldm-tech/nodra/actions/runs/40178082905), and [govulncheck](https://github.com/yldm-tech/nodra/actions/runs/40178082891). The reviewed API artifact is `public-api-e311e665f098af3a5ab582f3c4b31618905dbd40`, where the suffix identifies the tested merge commit.
