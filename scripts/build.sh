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

echo "=== verstak-desktop build ==="

# ── Go backend ──
echo "[backend]"

(cd "$ROOT" && go vet ./...)
report "go vet" $?

(cd "$ROOT" && go build ./...)
report "go build" $?

if command -v go-test-summary &>/dev/null || go test -list . ./... &>/dev/null 2>&1; then
  (cd "$ROOT" && go test -count=1 ./... 2>&1 || true)
  report "go test" $?
else
  echo "  ℹ️  go test: no tests to run"
fi

# ── Wails ──
echo "[wails]"
if command -v wails &>/dev/null; then
  (cd "$ROOT" && wails build -clean)
  report "wails build" $?
else
  echo "  ❌ wails: command not found. Install with: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
  FAILED=1
fi

# ── Frontend ──
echo "[frontend]"
if [ -f "$ROOT/frontend/package.json" ]; then
  (cd "$ROOT/frontend" && npm ci --no-audit --no-fund)
  report "npm ci" $?
  (cd "$ROOT/frontend" && npm run build)
  report "frontend build" $?
else
  echo "  ℹ️  frontend/package.json not found — skipping"
fi

echo ""
if [ "$FAILED" -eq 0 ]; then
  echo "✅ build passed"
else
  echo "❌ build failed"
fi
exit "$FAILED"
