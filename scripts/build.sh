#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "=== verstak-desktop build ==="
echo ""

"$ROOT/scripts/generate-brand-icons.sh"

# ── Dependency checks ──
echo "[deps]"
if ! command -v go &>/dev/null; then
  echo "  ❌ go: not found. Install Go 1.24+ from https://go.dev/dl/"
  exit 1
fi
echo "  ✅ go $(go version | grep -oP 'go\S+')"

if ! command -v node &>/dev/null; then
  echo "  ❌ node: not found. Install Node.js 20+"
  exit 1
fi
echo "  ✅ node $(node --version)"

if ! command -v npm &>/dev/null; then
  echo "  ❌ npm: not found"
  exit 1
fi
echo "  ✅ npm $(npm --version)"

# ── Frontend (build first — Go //go:embed needs frontend/dist/) ──
echo ""
echo "[frontend]"
if [ -f "$ROOT/frontend/package.json" ]; then
  if [ ! -d "$ROOT/frontend/node_modules" ]; then
    echo "  📦 node_modules missing — installing..."
    if [ -f "$ROOT/frontend/package-lock.json" ]; then
      (cd "$ROOT/frontend" && npm ci --no-audit --no-fund)
    else
      (cd "$ROOT/frontend" && npm install --no-audit --no-fund)
    fi
  fi
  (cd "$ROOT/frontend" && npm run build)
  echo "  ✅ frontend build"
else
  echo "  ℹ️  frontend/package.json not found — skipping"
fi

# ── Go backend ──
echo ""
echo "[backend]"

echo "  📦 go mod download..."
(cd "$ROOT" && go mod download)
echo "  ✅ go mod download"

echo "  🔍 go vet..."
(cd "$ROOT" && go vet ./...)
echo "  ✅ go vet"

echo "  🔨 go build..."
(cd "$ROOT" && go build -buildvcs=false ./...)
echo "  ✅ go build"

echo "  🧪 go test..."
(cd "$ROOT" && go test -count=1 ./...)
echo "  ✅ go test"

# ── Mouse button patch ──
# WebKitGTK does not propagate buttons 8/9 (XButton1/XButton2) into DOM events.
# We patch Wails' window.c on Linux to intercept these GTK signals and dispatch
# CustomEvent('verstak:navigate-back'/'verstak:navigate-forward') into the page.
echo ""
echo "[mouse-buttons]"
PATCH_FILE="$ROOT/patches/window.c.button-press.patch"
if [ -f "$PATCH_FILE" ]; then
  echo "  📦 go mod vendor..."
  (cd "$ROOT" && go mod vendor)
  echo "  ✅ go mod vendor"
  echo "  📝 applying window.c.button-press.patch..."
  (cd "$ROOT" && patch -p0 --forward < "$PATCH_FILE" 2>/dev/null || true)
  rm -f "$ROOT/vendor/github.com/wailsapp/wails/v2/internal/frontend/desktop/linux/window.c.rej"
  echo "  ✅ window.c patched"
else
  echo "  ℹ️  patch file not found at $PATCH_FILE — skipping"
fi

# ── Wails ──
echo ""
echo "[wails]"
WAILS=""
if command -v wails &>/dev/null; then
  WAILS="wails"
else
  GOBIN="$(go env GOBIN 2>/dev/null)"
  GOPATH="$(go env GOPATH 2>/dev/null)"
  if [ -n "$GOBIN" ] && [ -f "$GOBIN/wails" ]; then
    WAILS="$GOBIN/wails"
  elif [ -f "$GOPATH/bin/wails" ]; then
    WAILS="$GOPATH/bin/wails"
  fi
  if [ -z "$WAILS" ]; then
    echo "  📦 wails not found — installing..."
    go install github.com/wailsapp/wails/v2/cmd/wails@latest
    if [ -n "$GOBIN" ] && [ -f "$GOBIN/wails" ]; then
      WAILS="$GOBIN/wails"
    elif [ -f "$GOPATH/bin/wails" ]; then
      WAILS="$GOPATH/bin/wails"
    fi
  fi
fi

WAILS_BINARY="verstak-desktop"
WAILS_TAGS=""
if command -v pkg-config &>/dev/null; then
  if pkg-config --exists webkit2gtk-4.1 2>/dev/null; then
    WAILS_TAGS="-tags webkit2_41"
    echo "  ℹ️  using webkit2gtk-4.1"
  elif pkg-config --exists webkit2gtk-4.0 2>/dev/null; then
    echo "  ℹ️  using webkit2gtk-4.0"
  else
    echo "  ⚠️  no webkit2gtk found — wails requires libwebkit2gtk-4.0-dev or libwebkit2gtk-4.1-dev"
  fi
else
  echo "  ⚠️  pkg-config not found"
fi

if [ -z "$WAILS" ]; then
  echo "  ❌ wails: could not find or install"
  exit 1
fi

echo "  🔨 wails build..."
(cd "$ROOT" && GOFLAGS="${GOFLAGS:+$GOFLAGS }-buildvcs=false" "$WAILS" build -clean $WAILS_TAGS)
echo "  ✅ wails build"

# Copy plugins/ to build/bin/ so the binary can find them at runtime
if [ -d "$ROOT/plugins" ]; then
  mkdir -p "$ROOT/build/bin/plugins"
  cp -r "$ROOT/plugins/"* "$ROOT/build/bin/plugins/" 2>/dev/null || true
  echo "  📦 plugins copied to build/bin/plugins/"
fi

# Show where the binary ended up
if [ -f "$ROOT/build/bin/$WAILS_BINARY" ]; then
  echo "  📦 binary: $ROOT/build/bin/$WAILS_BINARY"
fi
if [ -f "$ROOT/$WAILS_BINARY" ]; then
  echo "  📦 binary: $ROOT/$WAILS_BINARY"
fi

echo ""
echo "✅ build passed"
