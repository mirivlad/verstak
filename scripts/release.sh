#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:-}"

if [[ -z "$VERSION" || ! "$VERSION" =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]]; then
  echo "usage: $0 <version>" >&2
  echo "example: $0 v0.1.0-alpha.1" >&2
  exit 2
fi

OFFICIAL_PLUGINS="${VERSTAK_OFFICIAL_PLUGINS_DIR:-$ROOT/../verstak-official-plugins}"
if [[ ! -d "$OFFICIAL_PLUGINS" ]]; then
  echo "official plugins repository not found: $OFFICIAL_PLUGINS" >&2
  exit 1
fi

echo "=== verstak desktop release $VERSION ==="
(cd "$OFFICIAL_PLUGINS" && ./scripts/build.sh)
"$ROOT/scripts/install-dev-plugins.sh"
"$ROOT/scripts/build.sh"

BINARY="$(find "$ROOT/build/bin" -maxdepth 1 -type f -name 'verstak-desktop*' -print -quit)"
if [[ -z "$BINARY" ]]; then
  echo "desktop binary was not produced in build/bin" >&2
  exit 1
fi

RELEASE_ROOT="$ROOT/release"
STAGING="$RELEASE_ROOT/verstak-desktop-$VERSION-linux-amd64"
ARCHIVE="$RELEASE_ROOT/verstak-desktop-linux-amd64-$VERSION.tar.gz"
rm -rf "$STAGING" "$ARCHIVE"
mkdir -p "$STAGING"

cp "$BINARY" "$STAGING/verstak-desktop"
cp "$ROOT/README.md" "$ROOT/LICENSE" "$STAGING/"
cp -R "$ROOT/plugins" "$STAGING/plugins"
tar -C "$RELEASE_ROOT" -czf "$ARCHIVE" "$(basename "$STAGING")"
(cd "$RELEASE_ROOT" && sha256sum "$(basename "$ARCHIVE")" > SHA256SUMS)

echo "release archive: $ARCHIVE"
echo "checksums:       $RELEASE_ROOT/SHA256SUMS"
