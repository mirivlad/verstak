#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:-}"

if [[ -z "$VERSION" || ! "$VERSION" =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]]; then
  echo "usage: $0 <version>" >&2
  echo "example: $0 v0.1.0-alpha.1" >&2
  exit 2
fi

echo "=== verstak desktop release $VERSION ==="
RELEASE_ROOT="$ROOT/release"
rm -rf "$RELEASE_ROOT"
mkdir -p "$RELEASE_ROOT"

"$ROOT/scripts/package-deb.sh" "$VERSION"
"$ROOT/scripts/package-appimage.sh" "$VERSION"
"$ROOT/scripts/package-windows-portable.sh" "$VERSION"

(cd "$RELEASE_ROOT" && find . -maxdepth 1 -type f \( -name '*.deb' -o -name '*.AppImage' -o -name '*.zip' \) \
  -printf '%f\n' | LC_ALL=C sort | xargs -r sha256sum > SHA256SUMS)

echo "release assets: $RELEASE_ROOT"
echo "checksums:      $RELEASE_ROOT/SHA256SUMS"
