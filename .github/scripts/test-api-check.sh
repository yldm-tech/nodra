#!/usr/bin/env bash
set -euo pipefail

# Use a minimal real module to prove that alias methods, fields, and interfaces are checked while unrelated internal packages remain outside the compatibility boundary.
api_test_source=$(cd "$(dirname "$0")/../.." && pwd)
api_test_tmp=$(mktemp -d)
trap 'rm -rf "$api_test_tmp"' EXIT
api_test_base="$api_test_tmp/base"
api_test_candidate="$api_test_tmp/candidate"
mkdir -p "$api_test_base/internal/core" "$api_test_base/internal/private" "$api_test_candidate/scripts"
printf 'module github.com/yldm-tech/nodra\n\ngo %s\n' "$(awk '/^go / {print $2}' "$api_test_source/go.mod")" > "$api_test_base/go.mod"
cat > "$api_test_base/api.go" <<'EOF'
package nodra
import "github.com/yldm-tech/nodra/internal/core"
type Store[T any] = core.Store[T]
type Options = core.Options
type Reader[T any] = core.Reader[T]
func New[T any]() *Store[T] { return new(Store[T]) }
EOF
cat > "$api_test_base/internal/core/core.go" <<'EOF'
package core
type Store[T any] struct{}
func (s *Store[T]) Get() T { var value T; return value }
type Options struct{ Limit int }
type Reader[T any] interface{ Get() T }
EOF
printf 'package private\ntype Detail struct{ Value int }\n' > "$api_test_base/internal/private/private.go"
cp -R "$api_test_base/." "$api_test_candidate/"
cp "$api_test_source/scripts/check-api.sh" "$api_test_candidate/scripts/check-api.sh"
git init -q "$api_test_base"
git -C "$api_test_base" add .
git -C "$api_test_base" -c user.name='API verification test' -c user.email='api-test@example.com' commit -qm 'test: baseline API'
printf 'package private\ntype Detail struct{}\n' > "$api_test_candidate/internal/private/private.go"
bash "$api_test_candidate/scripts/check-api.sh" "$api_test_base"

for change in method field interface; do
  case "$change" in
    method)
      sed '/^func (s \*Store\[T\]) Get()/d' "$api_test_base/internal/core/core.go" > "$api_test_candidate/internal/core/core.go"
      needle='Get: removed'
      ;;
    field)
      sed 's/Limit int/Limit string/' "$api_test_base/internal/core/core.go" > "$api_test_candidate/internal/core/core.go"
      needle='Limit: changed'
      ;;
    interface)
      sed 's/interface{ Get() T }/interface{ Get() T; Version() uint64 }/' "$api_test_base/internal/core/core.go" > "$api_test_candidate/internal/core/core.go"
      needle='Version: added'
      ;;
  esac
  if bash "$api_test_candidate/scripts/check-api.sh" "$api_test_base" > "$api_test_tmp/report" 2>&1; then
    echo "API check accepted an incompatible alias $change change" >&2
    exit 1
  fi
  if ! grep -F "$needle" "$api_test_tmp/report" || ! grep -F 'Unexpected API breakage.' "$api_test_tmp/report"; then
    cat "$api_test_tmp/report" >&2
    exit 1
  fi
done
echo 'Alias API regression checks passed'
