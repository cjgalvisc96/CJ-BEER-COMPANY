#!/usr/bin/env bash
# Coverage gate: 100% unit coverage, enforced.
#
# Coverage is collected across all internal packages with -coverpkg, so
# specification tests, e2e tests, and fitness tests all count toward the
# packages they exercise. Every internal package must sit at 100.0%,
# except:
#   - internal/app: composition-root wiring; its remaining branches are
#     fault-injection paths of the runtime (bus/server shutdown errors).
#     It is smoke-tested and reported, not gated.
#   - cmd/: the main() shell.
set -euo pipefail
cd "$(dirname "$0")/.."

EXEMPT_PREFIX="github.com/cjgalvisc96/cj-beer-company/internal/app"

go test -coverpkg=./internal/... -coverprofile=coverage.out ./... >/dev/null

echo "== per-function coverage below 100% (exempt: internal/app) =="
failures=0
while read -r file _ pct; do
  [[ "$file" == "total:" ]] && continue
  value="${pct%\%}"
  if [[ "$value" != "100.0" ]]; then
    if [[ "$file" == "$EXEMPT_PREFIX"* ]]; then
      echo "  (exempt) $file $pct"
    else
      echo "  FAIL     $file $pct"
      failures=1
    fi
  fi
done < <(go tool cover -func=coverage.out | awk '{print $1, $2, $3}')

total=$(go tool cover -func=coverage.out | awk '/^total:/ {print $3}')
echo "== total coverage: $total =="

if [[ "$failures" -ne 0 ]]; then
  echo "coverage gate FAILED: unit coverage must be 100%"
  exit 1
fi
echo "coverage gate OK"
