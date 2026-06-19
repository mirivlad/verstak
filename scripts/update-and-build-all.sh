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
  (cd "$OFFICIAL" && ./scripts/build.sh)
  echo "  ✅ official plugins built"
fi

# ── 3. Copy plugin packages to desktop ──
echo ""
echo "=== install plugins to desktop ==="
DEST="$ROOT/plugins"
rm -rf "$DEST"
mkdir -p "$DEST"
if [ -d "$OFFICIAL/dist" ] && [ -n "$(find "$OFFICIAL/dist" -mindepth 1 -maxdepth 1 -type d 2>/dev/null)" ]; then
  cp -r "$OFFICIAL/dist/"* "$DEST/" 2>/dev/null
  echo "  ✅ plugins copied to $DEST"
else
  echo "  ℹ️  no plugin packages to copy"
fi

# ── 4. Build desktop ──
echo ""
echo "=== build desktop ==="
exec "$ROOT/scripts/build.sh"
