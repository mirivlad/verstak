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

ensure_npm_deps() {
  local dir="$1"
  if [ ! -f "$dir/package.json" ]; then
    return 1
  fi
  if [ ! -d "$dir/node_modules" ]; then
    echo "  📦 node_modules missing — installing..."
    if [ -f "$dir/package-lock.json" ]; then
      (cd "$dir" && npm ci --no-audit --no-fund)
    else
      (cd "$dir" && npm install --no-audit --no-fund)
    fi
    report "npm install in $(basename "$dir")" $?
  fi
  return 0
}

echo "=== verstak-desktop test ==="

# ── Go tests ──
(cd "$ROOT" && go mod download)
GO_TEST_STATUS=0
OUTPUT=$(cd "$ROOT" && go test -count=1 -v ./... 2>&1) || GO_TEST_STATUS=$?
echo "$OUTPUT" | grep -E '(FAIL|PASS|---)' || true
report "go test" "$GO_TEST_STATUS"

WAILS_BINDINGS_STATUS=0
(cd "$ROOT" && node frontend/tests/wails-bindings-test.mjs) || WAILS_BINDINGS_STATUS=$?
report "Wails notification bindings" "$WAILS_BINDINGS_STATUS"

DEBUG_MODE_STATUS=0
(cd "$ROOT" && node --experimental-vm-modules frontend/tests/debug-mode-test.mjs) || DEBUG_MODE_STATUS=$?
report "session-only debug mode" "$DEBUG_MODE_STATUS"

# ── Frontend tests ──
echo "[frontend]"
if ensure_npm_deps "$ROOT/frontend"; then
  if grep -q '"test"' "$ROOT/frontend/package.json" 2>/dev/null; then
    FRONTEND_TEST_STATUS=0
    (cd "$ROOT/frontend" && npm test 2>&1) || FRONTEND_TEST_STATUS=$?
    report "frontend test" "$FRONTEND_TEST_STATUS"
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
