#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OFFICIAL_PLUGINS="${VERSTAK_OFFICIAL_PLUGINS_DIR:-$ROOT/../verstak-official-plugins}"
OUTPUT="${VERSTAK_LINUX_BUNDLE_DIR:-$ROOT/build/linux-amd64}"

if [[ ! -d "$OFFICIAL_PLUGINS" ]]; then
  echo "official plugins repository not found: $OFFICIAL_PLUGINS" >&2
  exit 1
fi

echo "=== verstak desktop Linux amd64 bundle ==="
(cd "$OFFICIAL_PLUGINS" && ./scripts/build.sh)
"$ROOT/scripts/install-dev-plugins.sh"
"$ROOT/scripts/build.sh"

BINARY="$ROOT/build/bin/verstak-desktop"
if [[ ! -x "$BINARY" ]]; then
  echo "desktop binary was not produced: $BINARY" >&2
  exit 1
fi

rm -rf "$OUTPUT"
mkdir -p "$OUTPUT"
install -m 755 "$BINARY" "$OUTPUT/verstak-desktop"
install -m 644 "$ROOT/README.md" "$ROOT/LICENSE" "$OUTPUT/"
cp -R "$ROOT/plugins" "$OUTPUT/plugins"
chmod -R a+rX "$OUTPUT/plugins"

echo "Linux bundle: $OUTPUT"
