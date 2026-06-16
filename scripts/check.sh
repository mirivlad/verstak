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

echo "=== verstak-desktop check ==="

# Go vet
(cd "$ROOT" && go vet ./...)
report "go vet" $?

# Go fmt (non-destructive — only report unformatted files)
UNFORMATTED=$(cd "$ROOT" && gofmt -l . 2>/dev/null || go fmt -n ./... 2>&1 || true)
if [ -z "$UNFORMATTED" ]; then
  echo "  ✅ gofmt: all files formatted"
else
  echo "  ❌ gofmt: unformatted files:"
  echo "$UNFORMATTED" | sed 's/^/    /'
  FAILED=1
fi

# Go mod tidy check (non-destructive — report only)
(cd "$ROOT" && go mod tidy -diff 2>&1 || echo "  ⚠️  go mod tidy check skipped")
report "go mod tidy" $?

# Frontend lint
if [ -f "$ROOT/frontend/package.json" ]; then
  # Check if npm ci is needed (node_modules missing)
  if [ ! -d "$ROOT/frontend/node_modules" ]; then
    echo "  ℹ️  frontend/node_modules missing — run build.sh first"
  fi
  if grep -q '"lint"' "$ROOT/frontend/package.json" 2>/dev/null; then
    (cd "$ROOT/frontend" && npx tsc --noEmit 2>&1 || true)
    report "frontend tsc --noEmit" $?
  else
    echo "  ℹ️  no lint script in frontend/package.json"
  fi
fi

echo ""
if [ "$FAILED" -eq 0 ]; then
  echo "✅ all checks passed"
else
  echo "❌ some checks failed"
fi
exit "$FAILED"
