// Package nodra is a concurrency-safe reactive state store for Go desktop applications.
//
// The root package is the stable public API; implementation details are kept in
// internal/core. Start with New or NewWithClone, then use Store.Get, Set,
// Update, Subscribe and Close. The public API reference and the v1 support
// contract are documented in docs/public-api.md and docs/support.md in the
// repository. Those documents are kept alongside the package because Go type
// aliases intentionally keep implementation methods out of some go doc views.
package nodra
