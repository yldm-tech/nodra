# Nodra 1.0 release checklist

This checklist is the release gate for `github.com/yldm-tech/nodra` v1.0.0. It records the evidence needed for the release; it is not an upgrade guide.

## Before the release PR

- [ ] The exported root API is frozen and reviewed against [`docs/public-api.md`](public-api.md).
- [ ] The compatibility policy and supported environment matrix have been reviewed without unresolved changes.
- [ ] The independent consumer test covers the root package and DevTools from a separate module: `bash scripts/check-consumer.sh`.
- [ ] The full local checks pass: `go test ./...`, `go test -race ./...`, `go vet ./...`, `golangci-lint run ./...`, and `go mod tidy -diff`.
- [ ] API comparison against the previous release reports no unreviewed incompatible change.
- [ ] The DevTools browser smoke test passes with the pinned Playwright version.
- [ ] The performance comparison and govulncheck jobs have completed successfully.

## Release PR and tag

- [ ] Set the release manifest to `1.0.0` only in the release PR after the API review is signed off.
- [ ] Confirm the release PR contains the final changelog entry and no unrelated feature changes.
- [ ] Merge the release PR and verify that the `v1.0.0` tag points at its release commit.
- [ ] Confirm the tag resolves through `proxy.golang.org` and `sum.golang.org`.
- [ ] Run `bash scripts/check-consumer.sh v1.0.0` from an independent module with `GOWORK=off`.

## After publication

- [ ] Verify `go list -m github.com/yldm-tech/nodra@v1.0.0` and `go mod download github.com/yldm-tech/nodra@v1.0.0` through the public proxy.
- [ ] Verify the release page, checksum, changelog links, and package documentation.
- [ ] Record the final API comparison, consumer result, support matrix, test runs, security result, and proxy result in the release review.

## Support boundary

The release supports Go 1.27.3 or newer within the same language/module compatibility line, MyGo 0.6.3, Linux/macOS/Windows on amd64, and pure-Go Linux/Windows arm64 builds. The arm64 rows are cross-build verified in CI; they do not claim native arm64 test execution. File persistence support is limited to local regular files with the documented cooperative locking contract.
