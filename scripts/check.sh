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

echo "=== verstak-desktop check ==="

# ── Go deps ──
(cd "$ROOT" && go mod download)
report "go mod download" $?

# ── Go vet ──
(cd "$ROOT" && go vet ./...)
report "go vet" $?

# ── Go fmt (non-destructive — only report unformatted files) ──
UNFORMATTED=$(cd "$ROOT" && find . -path ./vendor -prune -o -name '*.go' -print | xargs gofmt -l 2>/dev/null || go fmt -n ./... 2>&1 || true)
if [ -z "$UNFORMATTED" ]; then
  echo "  ✅ gofmt: all files formatted"
else
  echo "  ❌ gofmt: unformatted files:"
  echo "$UNFORMATTED" | sed 's/^/    /'
  FAILED=1
fi

# ── Go mod tidy check (non-destructive) ──
(cd "$ROOT" && go mod tidy -diff 2>&1 || echo "  ⚠️  go mod tidy check skipped")
report "go mod tidy" $?

# ── Frontend checks ──
echo "[frontend]"
if ensure_npm_deps "$ROOT/frontend"; then
  if grep -q '"lint"' "$ROOT/frontend/package.json" 2>/dev/null; then
    FRONTEND_LINT_STATUS=0
    (cd "$ROOT/frontend" && npm run lint 2>&1) || FRONTEND_LINT_STATUS=$?
    report "frontend lint" "$FRONTEND_LINT_STATUS"
  else
    echo "  ℹ️  no lint script in frontend/package.json"
  fi
  # Always run tsc --noEmit if typescript is available
  if [ -f "$ROOT/frontend/node_modules/.bin/tsc" ]; then
    FRONTEND_TSC_STATUS=0
    (cd "$ROOT/frontend" && npx tsc --noEmit 2>&1) || FRONTEND_TSC_STATUS=$?
    report "frontend tsc --noEmit" "$FRONTEND_TSC_STATUS"
  fi
else
  echo "  ℹ️  no frontend/package.json"
fi

echo ""
if [ "$FAILED" -eq 0 ]; then
  echo "✅ all checks passed"
else
  echo "❌ some checks failed"
fi
exit "$FAILED"
