# Compatibility policy

This policy describes the compatibility target for Nodra as the project moves from v0.x to v1. It complements the API comparison check in CI; the check is evidence for a release, not a replacement for documenting behavior.

## v0.x

Minor releases may contain source or behavior changes when they are needed to correct an unsafe contract, finish an API migration, or make the MyGo integration track its supported version. Breaking changes use a release note and a `!` conventional commit. Migration guidance is added when a released consumer base or a practical migration path makes it useful. When a practical migration path exists, the old spelling or behavior is deprecated for at least one minor release before removal.

Patch releases are limited to bug fixes, security fixes, documentation, dependency updates, and compatibility fixes that do not intentionally change the documented API or semantics.

The v0.x contract includes only behavior stated in package documentation and tests. Ordering of concurrent callbacks, internal lock layout, DevTools HTML structure, and allocation counts are not compatibility promises unless an API explicitly exposes them.

## v1

The v1.0 release will stabilize the exported root-package API, the `Reader` and `Observable` contracts, persistence interfaces and codecs, and the documented ownership, versioning, rollback, shutdown, and transaction semantics. A v1 minor release will not remove or change the meaning of an exported identifier. Additions remain source-compatible and may introduce new optional fields or interfaces.

An exported API can still evolve in a v1 minor release when the change only adds a method to an interface that Nodra does not ask applications to implement. Interfaces intended for application implementations are kept minimal; adding a method to those interfaces is a breaking change and requires a major release or a new interface.

Documented error identities and `errors.Is` relationships are part of the v1 contract. Error text, callback timing outside the stated ordering guarantees, JSON whitespace, DevTools presentation, and performance are not stable interfaces. A bug fix may change an outcome that contradicted the documentation, even when an application had come to depend on it.

Callback failures follow the boundary that owns the callback. Store mutation, selector, projection, history and ordinary subscription callbacks run on the publishing goroutine; a panic propagates to that caller after internal locks and registrations have been released, and the already-committed source value is not rolled back. Async action work, apply callbacks, and scope cleanup turn panics into `PanicError` values. Persistence status/error observers and DevTools redaction/metrics observers contain callback panics so their worker or inspector remains usable. Callbacks must still be short, reentrant only where the relevant Go doc permits it, and safe for concurrent invocation.

## Deprecation and release checks

Deprecations name the replacement and remain in Go documentation. Removal requires a major version or the next planned breaking release, an entry in the changelog, and a migration example when the change affects normal use. Every release must pass the public API comparison, independent consumer build, race tests, vet, lint, module tidy check, and the supported MyGo version checks in CI.

The module follows Go module major-version rules. A future v2 module will use a `/v2` module path; v1 keeps the current `github.com/yldm-tech/nodra` path. Applications should pin a release and avoid depending on unexported fields or generated DevTools assets.

## Current status

Until v1.0, all APIs are v0.x APIs even when they are already documented and tested. `AtomicGroup` is additive in the current release line and follows the transaction guarantees in its Go doc; its exact surface may still receive source-compatible refinement before v1.0.

## 1.0 release gate

The v1 release review must record the final public API export and the supported environment matrix. A standalone upgrade guide is not required while Nodra has no released consumer base; release notes are sufficient until a real migration needs a dedicated guide. The review must also include external-package tests that import `github.com/yldm-tech/nodra` and cover the constructor, observation, mutation, derived-value, persistence, async-action, and DevTools attachment surfaces. A passing internal test suite alone is not evidence that the root façade is usable by a consumer.

The root API reference is [public-api.md](public-api.md); the exact toolchain and platform promise is [support.md](support.md); the v1.0 release evidence is [release-1.0.md](release-1.0.md). These documents are release inputs, not informal examples: changes to any of them require a compatibility review and a changelog entry.
