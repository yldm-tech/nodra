#!/usr/bin/env bash
set -euo pipefail

# Exercise real local git remotes so tag peeling, manifest validation, and old-version reruns do not depend on GitHub or the Go proxy.
release_test_source=$(cd "$(dirname "$0")/../.." && pwd)
release_test_tmp=$(mktemp -d)
trap 'rm -rf "$release_test_tmp"' EXIT
release_test_origin="$release_test_tmp/origin"
release_test_candidate="$release_test_tmp/candidate checkout"
git init -q --initial-branch=main "$release_test_origin"
git -C "$release_test_origin" config user.name 'Release verification test'
git -C "$release_test_origin" config user.email 'release-test@example.com'
git init -q "$release_test_candidate"
git -C "$release_test_candidate" remote add origin "$release_test_origin"
mkdir -p "$release_test_candidate/scripts"
cp "$release_test_source/scripts/resolve-release.sh" "$release_test_candidate/scripts/resolve-release.sh"
printf '{".":"1.0.1"}\n' > "$release_test_candidate/.release-please-manifest.json"

commit_manifest() {
  printf '%s\n' "$1" > "$release_test_origin/.release-please-manifest.json"
  git -C "$release_test_origin" add .release-please-manifest.json
  git -C "$release_test_origin" commit -qm 'test: update release manifest'
  git -C "$release_test_origin" rev-parse HEAD
}

first_commit=$(commit_manifest '{".":"1.0.0"}')
git -C "$release_test_origin" tag v1.0.0
second_commit=$(commit_manifest '{".":"1.0.1"}')
git -C "$release_test_origin" tag -am 'Release 1.0.1' v1.0.1
commit_manifest '{".":"1.0.2"}' > /dev/null
git -C "$release_test_origin" tag v1.0.3
commit_manifest 'invalid json' > /dev/null
git -C "$release_test_origin" tag v1.0.4
commit_manifest '{}' > /dev/null
git -C "$release_test_origin" tag v1.0.5

assert_release() {
  local actual expected
  actual=$(bash "$release_test_candidate/scripts/resolve-release.sh" "$1")
  expected=$(printf 'version=%s\ncommit=%s' "$2" "$3")
  if [[ "$actual" != "$expected" ]]; then
    printf 'Unexpected release resolution:\n%s\nExpected:\n%s\n' "$actual" "$expected" >&2
    exit 1
  fi
}

assert_release v1.0.0 v1.0.0 "$first_commit"
assert_release v1.0.1 v1.0.1 "$second_commit"
assert_release '' v1.0.1 "$second_commit"
for version in v1.0.3 v1.0.4 v1.0.5 v9.9.9 v1.0.0suffix v01.0.0 main 'v1.0.0/../main'; do
  if bash "$release_test_candidate/scripts/resolve-release.sh" "$version" > "$release_test_tmp/output" 2> "$release_test_tmp/error"; then
    echo "Unexpectedly accepted release $version" >&2
    exit 1
  fi
  if [[ -s "$release_test_tmp/output" ]]; then
    echo "Failed resolution emitted release outputs for $version" >&2
    exit 1
  fi
done
echo 'Release resolution regression checks passed'
