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

BRAND_ICONS_STATUS=0
(cd "$ROOT" && ./scripts/test-brand-icons.sh) || BRAND_ICONS_STATUS=$?
report "desktop brand icon generation" "$BRAND_ICONS_STATUS"

# ── Frontend contract tests ──
# Every file in frontend/tests/ runs. Adding a test file is enough to enrol it;
# nothing here may be skipped selectively.
echo "[contract tests]"
CONTRACT_FOUND=0
for test_file in "$ROOT"/frontend/tests/*.mjs "$ROOT"/frontend/tests/*.cjs; do
  [ -f "$test_file" ] || continue
  CONTRACT_FOUND=$((CONTRACT_FOUND + 1))
  CONTRACT_STATUS=0
  (cd "$ROOT" && node --experimental-vm-modules "$test_file") || CONTRACT_STATUS=$?
  report "$(basename "$test_file")" "$CONTRACT_STATUS"
done
if [ "$CONTRACT_FOUND" -eq 0 ]; then
  echo "  ❌ no contract tests found in frontend/tests/"
  FAILED=1
fi

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
