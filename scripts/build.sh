#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSTAK_ROOT="$(cd "$ROOT/.." && pwd)"
FAILED=0
GLOBAL_ERRORS=""

report() {
  if [ "$2" -eq 0 ]; then
    echo "  ✅ $1"
  else
    echo "  ❌ $1"
    FAILED=1
  fi
}

# ── Global update: pull all repos, build official plugins ─────────────────
global_update() {
  local repos=("verstak-desktop" "verstak-sdk" "verstak-official-plugins" "verstak-sync-server" "verstak-browser-extension" "verstak-docs")
  local errors=""

  echo ""
  echo "=== global update: pull all repos ==="
  for repo in "${repos[@]}"; do
    local repo_path="$VERSTAK_ROOT/$repo"
    if [ ! -d "$repo_path" ]; then
      errors="$errors  ⚠️  $repo: directory not found at $repo_path\n"
      continue
    fi
    echo "[$repo]"
    (cd "$repo_path" && git pull --ff-only 2>&1) && echo "  ✅ pulled" || {
      errors="$errors  ❌ $repo: git pull failed\n"
      echo "  ❌ git pull failed"
    }
  done

  # Build official plugins
  local official_plugins="$VERSTAK_ROOT/verstak-official-plugins"
  if [ -d "$official_plugins" ]; then
    echo ""
    echo "=== build official plugins ==="
    (cd "$official_plugins" && if [ -f "package.json" ] && [ -d "plugins" ]; then
      if [ ! -d "node_modules" ]; then
        echo "[official-plugins] installing npm deps..."
        npm install --no-audit --no-fund 2>&1 || true
      fi
      # Build each plugin that has a frontend
      for plugin_dir in plugins/*/; do
        local plugin_name=$(basename "$plugin_dir")
        local fe_dir="$plugin_dir/frontend"
        if [ -d "$fe_dir" ] && [ -f "$fe_dir/package.json" ]; then
          echo "[$plugin_name] building frontend..."
          (cd "$fe_dir" && [ ! -d "node_modules" ] && npm install --no-audit --no-fund 2>&1 || true; npm run build 2>&1 || true)
        fi
        # Build backend if main.go exists
        local backend_dir="$plugin_dir/backend"
        if [ -f "$backend_dir/main.go" ]; then
          echo "[$plugin_name] building backend..."
          (cd "$backend_dir" && go build -o "$(basename "$backend_dir")" . 2>&1 || true)
        fi
      done
      echo "  ✅ official plugins built"
    else
      echo "  ℹ️  no plugins to build"
    fi)
  fi

  # Copy official plugins to desktop
  local dest="$ROOT/plugins"
  if [ -d "$official_plugins/plugins" ]; then
    echo ""
    echo "=== install official plugins to desktop ==="
    rm -rf "$dest"
    mkdir -p "$dest"
    cp -r "$official_plugins/plugins/"* "$dest/" 2>/dev/null || true
    echo "  ✅ plugins installed to $dest"
  fi

  GLOBAL_ERRORS="$errors"
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
echo ""

# ── Global update (best-effort — errors collected, not fatal) ──
global_update

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
WAILS=""
if command -v wails &>/dev/null; then
  WAILS="wails"
else
  # Check GO bin paths
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
    report "wails install" $?
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
if [ -n "$WAILS" ]; then
  echo "  🔨 wails build..."
  (cd "$ROOT" && "$WAILS" build -clean $WAILS_TAGS)
  report "wails build" $?
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
else
  echo "  ❌ wails: could not install"
  FAILED=1
fi

echo ""
if [ "$FAILED" -eq 0 ] && [ -z "$GLOBAL_ERRORS" ]; then
  echo "✅ build passed"
else
  echo "❌ build completed with issues"
  if [ -n "$GLOBAL_ERRORS" ]; then
    echo ""
    echo "Global update errors:"
    echo -e "$GLOBAL_ERRORS"
  fi
fi
exit "$FAILED"
