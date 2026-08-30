#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SOURCE="$ROOT/packaging/linux/verstak.svg"

if [[ -n "${MAGICK_BIN:-}" ]]; then
  MAGICK="$MAGICK_BIN"
elif command -v magick >/dev/null 2>&1; then
  MAGICK="magick"
elif command -v convert >/dev/null 2>&1; then
  # Ubuntu 24.04 ships ImageMagick 6, where the ImageMagick 7 `magick`
  # launcher does not exist. The operations used below have the same CLI
  # syntax through the IM6 `convert` entry point.
  MAGICK="convert"
else
  echo "ImageMagick is required to generate Verstak application icons: neither magick nor convert was found" >&2
  exit 1
fi

if ! command -v "$MAGICK" >/dev/null 2>&1; then
  echo "ImageMagick is required to generate Verstak application icons: $MAGICK not found" >&2
  exit 1
fi
if [[ ! -f "$SOURCE" ]]; then
  echo "Verstak SVG icon source is missing: $SOURCE" >&2
  exit 1
fi

render_png() {
  local size="$1"
  local target="$2"
  mkdir -p "$(dirname "$target")"
  "$MAGICK" -background none "$SOURCE" -resize "${size}x${size}" -strip "PNG32:$target"
}

render_ico() {
  local target="$1"
  shift
  local temporary
  temporary="$(mktemp -d)"
  local images=()
  for size in "$@"; do
    local image="$temporary/${size}.png"
    render_png "$size" "$image"
    images+=("$image")
  done
  mkdir -p "$(dirname "$target")"
  "$MAGICK" "${images[@]}" "$target"
  rm -rf "$temporary"
}

render_png 256 "$ROOT/internal/shell/tray/verstak.png"
render_ico "$ROOT/internal/shell/tray/verstak.ico" 16 20 24 32 48 256

# XFCE and other freedesktop panels often resolve the window icon through the
# raster hicolor hierarchy. Keep these generated from the same canonical SVG
# as the tray and Windows resources.
for size in 16 24 32 48 64 128 256; do
  render_png "$size" "$ROOT/build/linux-icons/hicolor/${size}x${size}/apps/verstak.png"
done

# Wails consumes this PNG while it prepares the Windows executable resources.
# Both files are generated build inputs and therefore deliberately ignored by Git.
render_png 1024 "$ROOT/build/appicon.png"
render_ico "$ROOT/build/windows/icon.ico" 16 32 48 64 128 256

echo "generated Verstak tray, Linux and application icons from $SOURCE using $MAGICK"
