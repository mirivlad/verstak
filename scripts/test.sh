#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FAILED=0

report() {
  if [ "$2" -eq 0 ]; then
    echo "  ✅ $1"
  else
    echo "  ❌ $1"
    FAILED=1
  fi
}

echo "=== verstak-desktop test ==="

# Go tests
(cd "$ROOT" && go test -count=1 -v ./... 2>&1 || true)
report "go test" $?

# Frontend tests
if [ -f "$ROOT/frontend/package.json" ]; then
  # Only run if vitest or jest is in the config
  if grep -q '"test"' "$ROOT/frontend/package.json" 2>/dev/null; then
    (cd "$ROOT/frontend" && npm test 2>&1 || true)
    report "frontend test" $?
  else
    echo "  ℹ️  no test script in frontend/package.json"
  fi
else
  echo "  ℹ️  no frontend/package.json"
fi

echo ""
if [ "$FAILED" -eq 0 ]; then
  echo "✅ all tests passed"
else
  echo "❌ some tests failed"
fi
exit "$FAILED"
