#!/usr/bin/env bash
# update-and-build-all.sh — dev helper: pull all repos, build official plugins, build desktop
# This is NOT part of CI. For local deterministic build, use scripts/build.sh in each repo.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSTAK_ROOT="$(cd "$ROOT/.." && pwd)"

echo "=== update-and-build-all ==="
echo ""

# ── 1. Pull all repos ──
echo "=== pull all repos ==="
repos=("verstak-desktop" "verstak-sdk" "verstak-official-plugins" "verstak-sync-server" "verstak-browser-extension" "verstak-docs")
for repo in "${repos[@]}"; do
  repo_path="$VERSTAK_ROOT/$repo"
  if [ ! -d "$repo_path" ]; then
    echo "  ⚠️  $repo: directory not found at $repo_path — skipping"
    continue
  fi
  echo "[$repo]"
  (cd "$repo_path" && git pull --ff-only 2>&1) && echo "  ✅ pulled" || echo "  ❌ git pull failed"
done

# ── 2. Build official plugins ──
echo ""
echo "=== build official plugins ==="
OFFICIAL="$VERSTAK_ROOT/verstak-official-plugins"
if [ ! -d "$OFFICIAL" ]; then
  echo "  ⚠️  verstak-official-plugins not found — skipping"
else
  # npm deps
  if [ ! -d "$OFFICIAL/node_modules" ] && [ -f "$OFFICIAL/package.json" ]; then
    echo "  📦 installing npm deps..."
    (cd "$OFFICIAL" && npm install --no-audit --no-fund)
  fi
  # Build each plugin that has a frontend or backend
  for plugin_dir in "$OFFICIAL"/plugins/*/; do
    [ -d "$plugin_dir" ] || continue
    plugin_name="$(basename "$plugin_dir")"

    # Frontend build
    fe_dir="$plugin_dir/frontend"
    if [ -d "$fe_dir" ] && [ -f "$fe_dir/package.json" ]; then
      echo "[$plugin_name] building frontend..."
      if [ ! -d "$fe_dir/node_modules" ]; then
        (cd "$fe_dir" && npm install --no-audit --no-fund)
      fi
      (cd "$fe_dir" && npm run build)
      echo "  ✅ $plugin_name frontend"
    fi

    # Backend build
    backend_dir="$plugin_dir/backend"
    if [ -f "$backend_dir/main.go" ]; then
      echo "[$plugin_name] building backend..."
      (cd "$backend_dir" && go build -o "$(basename "$backend_dir")" .)
      echo "  ✅ $plugin_name backend"
    fi
  done
  echo "  ✅ official plugins built"
fi

# ── 3. Copy plugins to desktop ──
echo ""
echo "=== install plugins to desktop ==="
DEST="$ROOT/plugins"
rm -rf "$DEST"
mkdir -p "$DEST"
if [ -d "$OFFICIAL/plugins" ]; then
  cp -r "$OFFICIAL/plugins/"* "$DEST/" 2>/dev/null
  echo "  ✅ plugins copied to $DEST"
else
  echo "  ℹ️  no plugins to copy"
fi

# ── 4. Build desktop ──
echo ""
echo "=== build desktop ==="
exec "$ROOT/scripts/build.sh"
