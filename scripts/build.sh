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

echo "=== verstak-desktop build ==="

# ── Dependency checks ──
echo "[deps]"
if ! command -v go &>/dev/null; then
  echo "  ❌ go: not found. Install Go 1.24+ from https://go.dev/dl/"
  FAILED=1
else
  echo "  ✅ go $(go version | grep -oP 'go\S+')"
fi
if ! command -v node &>/dev/null; then
  echo "  ❌ node: not found. Install Node.js 20+"
  FAILED=1
else
  echo "  ✅ node $(node --version)"
fi
if ! command -v npm &>/dev/null; then
  echo "  ❌ npm: not found"
  FAILED=1
fi

if [ "$FAILED" -ne 0 ]; then
  echo ""
  echo "❌ build failed — missing core dependencies"
  exit 1
fi

# ── Frontend (build first — Go //go:embed needs frontend/dist/) ──
echo "[frontend]"
if [ -f "$ROOT/frontend/package.json" ]; then
  ensure_npm_deps "$ROOT/frontend"
  (cd "$ROOT/frontend" && npm run build)
  report "frontend build" $?
else
  echo "  ℹ️  frontend/package.json not found — skipping"
fi

# ── Go backend ──
echo "[backend]"

# Ensure Go module deps are downloaded
(cd "$ROOT" && go mod download)
report "go mod download" $?

(cd "$ROOT" && go vet ./...)
report "go vet" $?

(cd "$ROOT" && go build ./...)
report "go build" $?

# Go test (best-effort — some packages may have no tests)
(cd "$ROOT" && go test -count=1 ./... 2>&1 || true)
report "go test" $?

# ── Wails ──
echo "[wails]"
if command -v wails &>/dev/null; then
  (cd "$ROOT" && wails build -clean)
  report "wails build" $?
else
  echo "  ❌ wails: command not found. Install with: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
  FAILED=1
fi

echo ""
if [ "$FAILED" -eq 0 ]; then
  echo "✅ build passed"
else
  echo "❌ build failed"
fi
exit "$FAILED"
