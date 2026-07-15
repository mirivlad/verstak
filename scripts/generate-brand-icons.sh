#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MAGICK="${MAGICK_BIN:-magick}"
SOURCE="$ROOT/packaging/linux/verstak.svg"

if ! command -v "$MAGICK" >/dev/null; then
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

# Wails consumes this PNG while it prepares the Windows executable resources.
# Both files are generated build inputs and therefore deliberately ignored by Git.
render_png 1024 "$ROOT/build/appicon.png"
render_ico "$ROOT/build/windows/icon.ico" 16 32 48 64 128 256

echo "generated Verstak tray and application icons from $SOURCE"
