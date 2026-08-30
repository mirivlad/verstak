#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ICONS=(
  "$ROOT/internal/shell/tray/verstak.png"
  "$ROOT/internal/shell/tray/verstak.ico"
  "$ROOT/build/appicon.png"
  "$ROOT/build/windows/icon.ico"
)
for size in 16 24 32 48 64 128 256; do
  ICONS+=("$ROOT/build/linux-icons/hicolor/${size}x${size}/apps/verstak.png")
done

"$ROOT/scripts/generate-brand-icons.sh"
first="$(sha256sum "${ICONS[@]}")"
"$ROOT/scripts/generate-brand-icons.sh"
second="$(sha256sum "${ICONS[@]}")"

if [[ "$first" != "$second" ]]; then
  echo "brand icon generation is not deterministic" >&2
  exit 1
fi

echo "desktop brand icon generation is deterministic"
