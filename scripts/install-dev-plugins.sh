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

# ── ensure dist packages exist ──
DIST_ROOT="$OFFICIAL_PLUGINS/dist"
if [ ! -d "$DIST_ROOT" ] || [ -z "$(find "$DIST_ROOT" -mindepth 1 -maxdepth 1 -type d 2>/dev/null)" ] || [ ! -f "$DIST_ROOT/import/plugin.json" ]; then
  echo "  ℹ️  dist packages not found at $DIST_ROOT"
  echo "  → Running build.sh in verstak-official-plugins..."
  (cd "$OFFICIAL_PLUGINS" && ./scripts/build.sh)
  echo ""
  if [ ! -d "$DIST_ROOT" ] || [ -z "$(find "$DIST_ROOT" -mindepth 1 -maxdepth 1 -type d 2>/dev/null)" ] || [ ! -f "$DIST_ROOT/import/plugin.json" ]; then
    echo "❌ dist packages still missing after build"
    exit 1
  fi
fi

# ── create ./plugins/* from official dist packages ──
PLUGIN_DIR="$ROOT/plugins"
echo "  → installing official plugins to $PLUGIN_DIR"

mkdir -p "$ROOT/plugins"

# Clean up any leftover temp directories
for tmp in "$ROOT"/.official-plugins-tmp.*; do
  [ -d "$tmp" ] && rm -rf "$tmp"
done

# Atomic replace: install to temp then rename
TMP_DIR=$(mktemp -d "$ROOT/.official-plugins-tmp.XXXXXX")
cp -r "$DIST_ROOT/"* "$TMP_DIR/"
# Remove old plugin directories (fix permissions first if needed)
if [ -d "$PLUGIN_DIR" ]; then
  chmod -R u+rwx "$PLUGIN_DIR" 2>/dev/null || true
  rm -rf "$PLUGIN_DIR"
fi
mv "$TMP_DIR" "$PLUGIN_DIR"

# ── verify ──
if [ -d "$PLUGIN_DIR" ]; then
  if [ ! -f "$PLUGIN_DIR/import/plugin.json" ] || [ ! -f "$PLUGIN_DIR/import/frontend/dist/index.js" ]; then
    echo "❌ install failed: official import plugin is incomplete"
    exit 1
  fi
  FILE_COUNT=$(find "$PLUGIN_DIR" -type f | wc -l)
  echo "  ✅ installed: $PLUGIN_DIR"
  echo "     files:     $FILE_COUNT"
else
  echo "❌ install failed: $PLUGIN_DIR missing"
  exit 1
fi

echo "✅ install-dev-plugins done"
