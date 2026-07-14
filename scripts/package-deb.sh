#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:-}"

if [[ -z "$VERSION" || ! "$VERSION" =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]]; then
  echo "usage: $0 <version>" >&2
  echo "example: $0 v0.1.0-alpha.1" >&2
  exit 2
fi
if ! command -v dpkg-deb >/dev/null; then
  echo "dpkg-deb is required to create a Debian package" >&2
  exit 1
fi

"$ROOT/scripts/build-linux-bundle.sh"

PACKAGE_VERSION="${VERSION#v}"
BUNDLE="${VERSTAK_LINUX_BUNDLE_DIR:-$ROOT/build/linux-amd64}"
STAGING="$ROOT/build/deb/verstak_$PACKAGE_VERSION"
RELEASE_ROOT="$ROOT/release"
ARCHIVE="$RELEASE_ROOT/verstak_${PACKAGE_VERSION}_amd64.deb"

rm -rf "$STAGING" "$ARCHIVE"
mkdir -p "$STAGING/DEBIAN" "$STAGING/opt/verstak" "$STAGING/usr/bin" \
  "$STAGING/usr/share/applications" "$STAGING/usr/share/icons/hicolor/scalable/apps" \
  "$STAGING/usr/share/doc/verstak"

sed "s/@VERSION@/$PACKAGE_VERSION/g" "$ROOT/packaging/deb/control" > "$STAGING/DEBIAN/control"
install -m 755 "$ROOT/packaging/deb/postinst" "$STAGING/DEBIAN/postinst"
install -m 755 "$ROOT/packaging/deb/verstak" "$STAGING/usr/bin/verstak"
install -m 755 "$BUNDLE/verstak-desktop" "$STAGING/opt/verstak/verstak-desktop"
install -m 644 "$BUNDLE/README.md" "$BUNDLE/LICENSE" "$STAGING/usr/share/doc/verstak/"
install -m 644 "$ROOT/packaging/linux/verstak.desktop" "$STAGING/usr/share/applications/verstak.desktop"
install -m 644 "$ROOT/packaging/linux/verstak.svg" "$STAGING/usr/share/icons/hicolor/scalable/apps/verstak.svg"
cp -R "$BUNDLE/plugins" "$STAGING/opt/verstak/plugins"

mkdir -p "$RELEASE_ROOT"
dpkg-deb --root-owner-group --build "$STAGING" "$ARCHIVE"
(cd "$RELEASE_ROOT" && find . -maxdepth 1 -type f \( -name '*.deb' -o -name '*.AppImage' -o -name '*.zip' \) \
  -printf '%f\n' | LC_ALL=C sort | xargs -r sha256sum > SHA256SUMS)

echo "Debian package: $ARCHIVE"
echo "checksums:      $RELEASE_ROOT/SHA256SUMS"
