#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "=== verstak-desktop: install dev plugins ==="

# ── locate sibling repo ──
OFFICIAL_PLUGINS="$ROOT/../verstak-official-plugins"
if [ ! -d "$OFFICIAL_PLUGINS" ]; then
  echo "❌ sibling repo not found at $OFFICIAL_PLUGINS"
  echo "   Expected structure:"
  echo "     ../verstak-official-plugins/"
  echo "     ../verstak-desktop/   (this repo)"
  exit 1
fi

# ── ensure dist package exists ──
DIST_PACKAGE="$OFFICIAL_PLUGINS/dist/platform-test"
if [ ! -d "$DIST_PACKAGE" ]; then
  echo "  ℹ️  dist package not found at $DIST_PACKAGE"
  echo "  → Running build.sh in verstak-official-plugins..."
  (cd "$OFFICIAL_PLUGINS" && ./scripts/build.sh)
  echo ""
  if [ ! -d "$DIST_PACKAGE" ]; then
    echo "❌ dist package still missing after build"
    exit 1
  fi
fi

# ── create ./plugins/platform-test ──
PLUGIN_DIR="$ROOT/plugins/platform-test"
echo "  → installing platform-test to $PLUGIN_DIR"

mkdir -p "$ROOT/plugins"

# Clean up any leftover temp directories
for tmp in "$ROOT/plugins"/.platform-test-tmp.*; do
  [ -d "$tmp" ] && rm -rf "$tmp"
done

# Atomic replace: install to temp then rename
TMP_DIR=$(mktemp -d "$ROOT/plugins/.platform-test-tmp.XXXXXX")
cp -r "$DIST_PACKAGE/." "$TMP_DIR/"
# Remove old directory (fix permissions first if needed)
if [ -d "$PLUGIN_DIR" ]; then
  chmod -R u+rwx "$PLUGIN_DIR" 2>/dev/null || true
  rm -rf "$PLUGIN_DIR"
fi
mv "$TMP_DIR" "$PLUGIN_DIR"

# ── verify ──
if [ -f "$PLUGIN_DIR/plugin.json" ]; then
  PLUGIN_ID=$(python3 -c "import json; print(json.load(open('$PLUGIN_DIR/plugin.json')).get('id','unknown'))" 2>/dev/null || echo "unknown")
  FILE_COUNT=$(find "$PLUGIN_DIR" -type f | wc -l)
  echo "  ✅ installed: $PLUGIN_DIR"
  echo "     plugin id: $PLUGIN_ID"
  echo "     files:     $FILE_COUNT"
else
  echo "❌ install failed: plugin.json missing in $PLUGIN_DIR"
  exit 1
fi

echo "✅ install-dev-plugins done"
