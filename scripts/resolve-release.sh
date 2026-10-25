#!/usr/bin/env bash
set -euo pipefail

# Resolve an immutable release commit and verify its manifest before any published-module checks. Output is suitable for GITHUB_OUTPUT; git diagnostics stay on stderr.
if [[ $# -gt 1 ]]; then
  echo 'Usage: scripts/resolve-release.sh [vX.Y.Z]' >&2
  exit 2
fi
release_source=$(cd "$(dirname "$0")/.." && pwd)
release_version=${1:-v$(jq -er '.["."]' "$release_source/.release-please-manifest.json")}
if [[ ! "$release_version" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  echo "Invalid release version: $release_version; expected vX.Y.Z" >&2
  exit 1
fi
git -C "$release_source" fetch --no-tags origin "refs/tags/$release_version"
release_commit=$(git -C "$release_source" rev-parse --verify 'FETCH_HEAD^{commit}')
release_manifest=$(git -C "$release_source" show "$release_commit:.release-please-manifest.json" | jq -er '.["."]')
if [[ "v$release_manifest" != "$release_version" ]]; then
  echo "Tag $release_version points to manifest version $release_manifest at $release_commit" >&2
  exit 1
fi
printf 'version=%s\ncommit=%s\n' "$release_version" "$release_commit"
