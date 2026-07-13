#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:-}"

if [[ -z "$VERSION" || ! "$VERSION" =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]]; then
  echo "usage: $0 <version>" >&2
  echo "example: $0 v0.1.0-alpha.1" >&2
  exit 2
fi
for command in zip; do
  if ! command -v "$command" >/dev/null; then
    echo "$command is required to create the Windows portable archive" >&2
    exit 1
  fi
done

"$ROOT/scripts/build-windows.sh"

RELEASE_ROOT="$ROOT/release"
STAGING="$RELEASE_ROOT/verstak-windows-amd64-$VERSION"
ARCHIVE="$RELEASE_ROOT/verstak-windows-amd64-$VERSION.zip"
rm -rf "$STAGING" "$ARCHIVE"
mkdir -p "$STAGING"
cp -R "$ROOT/build/windows-amd64/." "$STAGING/"
install -m 644 "$ROOT/README.md" "$ROOT/LICENSE" "$STAGING/"
install -m 644 "$ROOT/packaging/windows/Verstak.cmd" "$STAGING/Verstak.cmd"

(cd "$RELEASE_ROOT" && zip -qr "$(basename "$ARCHIVE")" "$(basename "$STAGING")")
(cd "$RELEASE_ROOT" && find . -maxdepth 1 -type f \( -name '*.deb' -o -name '*.AppImage' -o -name '*.zip' \) \
  -printf '%f\n' | LC_ALL=C sort | xargs -r sha256sum > SHA256SUMS)

echo "Windows portable archive: $ARCHIVE"
echo "checksums:                $RELEASE_ROOT/SHA256SUMS"
