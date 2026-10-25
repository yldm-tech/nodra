# AGENTS.md

Nodra is a concurrency-safe reactive state store for Go desktop applications. This repository holds the Go module `github.com/yldm-tech/nodra` and, under `site/`, its official website at https://nodra.yldm.ai.

## Layout

- `api.go`, `doc.go`: the root package, which re-exports `internal/core`. It is the only supported import path.
- `internal/core/`: the implementation of every Store, history, persistence, async and scope type.
- `devtools/`: the optional inspector; the core never imports it.
- `examples/`, `example_test.go`, `public_api_test.go`: runnable examples and the external-package tests that define the v1 contract.
- `docs/`: `public-api.md`, `architecture.md`, `compatibility.md`, `support.md` and the release evidence. They are release inputs: changing them needs a compatibility review and a changelog entry.
- `site/`: the website (TanStack Start, TanStack Router, Tailwind CSS v4, React 19 on Cloudflare Workers).

## Go module

Follow [CONTRIBUTING.md](CONTRIBUTING.md). Before a pull request run `go test ./...`, `go test -race ./...`, `go vet ./...` and `go mod tidy -diff`. API changes run `scripts/check-api.sh` against the base; an intentional incompatibility is recorded under `.github/api-changes/`. The v1 compatibility rules in `docs/compatibility.md` apply to every exported identifier and documented behaviour.

## Releases

release-please publishes the module from Conventional Commits on `main`, and pull requests are squash-merged, so the pull request title becomes the commit that release-please reads.

- `site/` is excluded from releases, but only for commits whose files are all inside `site/`. A squashed commit that also touches anything outside `site/` counts as a library change, and a `feat` or `fix` title then publishes a new Nodra version. v1.1.0 was released this way by a website pull request that also edited CI and release configuration.
- Keep website changes and everything else in separate pull requests. Changes to workflows, release configuration, READMEs or these instruction files use a non-releasing type such as `ci`, `chore` or `docs`.

## Website

Work from `site/` with pnpm: `pnpm dev`, `pnpm typecheck`, `pnpm build`. The `site` workflow type-checks and builds every change and deploys `main` to Cloudflare Workers once the `CLOUDFLARE_API_TOKEN` secret is set. To deploy by hand, run `CLOUDFLARE_ACCOUNT_ID=e2b27aea5f599efdbe0a7016a2580d05 pnpm run deploy` (`pnpm deploy` is a different, built-in pnpm command).

- The API reference in `site/src/docs/content.ts` must match the source. Check every signature against `api.go`, `internal/core` and `devtools`, compare behaviour notes with the Go doc comments and `docs/architecture.md`, and compile the examples against the module before changing them. The original design text contained wrong signatures and an incorrect claim that `Store.Close` is idempotent.
- Chinese is served at the bare path and English at `?lang=en`; keep both languages in step for every string.
- Search and social metadata come from `site/src/lib/seo.ts`. The social cards in `site/public/og/` are rendered from `site/scripts/og-image.html` by `sh site/scripts/og-image.sh`; re-render them after changing that copy.
- The home page shows GitHub stars, the version `proxy.golang.org` resolves and the pkg.go.dev importer count (`site/src/lib/stats.ts`). Go modules have no public download counts; do not show estimated or derived numbers in their place.
