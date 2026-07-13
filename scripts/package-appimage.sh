#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:-}"
APPIMAGETOOL_URL="${APPIMAGETOOL_URL:-https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage}"

if [[ -z "$VERSION" || ! "$VERSION" =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]]; then
  echo "usage: $0 <version>" >&2
  echo "example: $0 v0.1.0-alpha.1" >&2
  exit 2
fi
if ! command -v ldd >/dev/null || ! command -v file >/dev/null; then
  echo "ldd and file are required to bundle AppImage runtime libraries" >&2
  exit 1
fi

"$ROOT/scripts/build-linux-bundle.sh"

APPIMAGETOOL="${APPIMAGETOOL_BIN:-$ROOT/build/tools/appimagetool-x86_64.AppImage}"
if [[ ! -x "$APPIMAGETOOL" ]]; then
  if ! command -v curl >/dev/null; then
    echo "appimagetool is missing; install curl or set APPIMAGETOOL_BIN" >&2
    exit 1
  fi
  mkdir -p "$(dirname "$APPIMAGETOOL")"
  curl --fail --location --show-error "$APPIMAGETOOL_URL" -o "$APPIMAGETOOL"
  chmod +x "$APPIMAGETOOL"
fi

BUNDLE="${VERSTAK_LINUX_BUNDLE_DIR:-$ROOT/build/linux-amd64}"
APPDIR="$ROOT/build/appimage/AppDir"
RELEASE_ROOT="$ROOT/release"
ARCHIVE="$RELEASE_ROOT/verstak-linux-x86_64-$VERSION.AppImage"
WEBKIT_RUNTIME_DIR="${VERSTAK_WEBKIT_RUNTIME_DIR:-/usr/lib/x86_64-linux-gnu/webkit2gtk-4.1}"

if [[ ! -d "$WEBKIT_RUNTIME_DIR" ]] || [[ ! -x "$WEBKIT_RUNTIME_DIR/WebKitWebProcess" ]]; then
  echo "WebKitWebProcess was not found at $WEBKIT_RUNTIME_DIR" >&2
  echo "Set VERSTAK_WEBKIT_RUNTIME_DIR to the WebKitGTK 4.1 runtime directory." >&2
  exit 1
fi

rm -rf "$APPDIR" "$ARCHIVE"
mkdir -p "$APPDIR/usr/bin" "$APPDIR/usr/lib" "$APPDIR/usr/share/applications" \
  "$APPDIR/usr/share/icons/hicolor/scalable/apps"
install -m 755 "$ROOT/packaging/linux/AppRun" "$APPDIR/AppRun"
install -m 755 "$BUNDLE/verstak-desktop" "$APPDIR/usr/bin/verstak-desktop"
install -m 644 "$ROOT/packaging/linux/verstak.desktop" "$APPDIR/verstak.desktop"
install -m 644 "$ROOT/packaging/linux/verstak.desktop" "$APPDIR/usr/share/applications/verstak.desktop"
install -m 644 "$ROOT/packaging/linux/verstak.svg" "$APPDIR/verstak.svg"
install -m 644 "$ROOT/packaging/linux/verstak.svg" "$APPDIR/usr/share/icons/hicolor/scalable/apps/verstak.svg"
cp -R "$BUNDLE/plugins" "$APPDIR/usr/bin/plugins"
cp -a "$WEBKIT_RUNTIME_DIR" "$APPDIR/usr/lib/webkit2gtk-4.1"

copy_runtime_dir() {
  local source="$1"
  local target="$2"
  if [[ -d "$source" ]]; then
    mkdir -p "$target"
    cp -a "$source/." "$target/"
  fi
}

copy_runtime_dir /usr/lib/x86_64-linux-gnu/gio/modules "$APPDIR/usr/lib/gio/modules"
copy_runtime_dir /usr/lib/x86_64-linux-gnu/gstreamer-1.0 "$APPDIR/usr/lib/gstreamer-1.0"
copy_runtime_dir /usr/lib/x86_64-linux-gnu/gdk-pixbuf-2.0 "$APPDIR/usr/lib/gdk-pixbuf-2.0"
copy_runtime_dir /usr/share/glib-2.0/schemas "$APPDIR/usr/share/glib-2.0/schemas"

declare -A queued=()
queue_elf() {
  local candidate="$1"
  [[ -f "$candidate" ]] || return
  if [[ "$(file --brief "$candidate")" != *ELF* ]]; then
    return
  fi
  if [[ -z "${queued[$candidate]:-}" ]]; then
    queued[$candidate]=1
    printf '%s\n' "$candidate" >> "$APPDIR/.elf-queue"
  fi
}

queue_elf "$APPDIR/usr/bin/verstak-desktop"
while IFS= read -r candidate; do
  queue_elf "$candidate"
done < <(find "$APPDIR/usr/lib/webkit2gtk-4.1" "$APPDIR/usr/lib/gio/modules" \
  "$APPDIR/usr/lib/gstreamer-1.0" "$APPDIR/usr/lib/gdk-pixbuf-2.0" -type f)

while IFS= read -r candidate; do
  if ldd "$candidate" | grep -q 'not found'; then
    echo "unresolved shared library in $candidate" >&2
    exit 1
  fi
  while IFS= read -r library; do
    case "$(basename "$library")" in
      libc.so.6|libm.so.6|libpthread.so.0|librt.so.1|libdl.so.2|ld-linux-x86-64.so.2)
        continue
        ;;
    esac
    bundled="$APPDIR/usr/lib/$(basename "$library")"
    if [[ ! -e "$bundled" ]]; then
      cp -aL "$library" "$bundled"
    fi
    queue_elf "$bundled"
  done < <(ldd "$candidate" | awk '/=> \/[^ ]+/ { print $3 } /^\// { print $1 }')
done < "$APPDIR/.elf-queue"
rm -f "$APPDIR/.elf-queue"

if command -v glib-compile-schemas >/dev/null; then
  glib-compile-schemas "$APPDIR/usr/share/glib-2.0/schemas"
fi

mkdir -p "$RELEASE_ROOT"
ARCH=x86_64 APPIMAGE_EXTRACT_AND_RUN=1 "$APPIMAGETOOL" "$APPDIR" "$ARCHIVE"
(cd "$RELEASE_ROOT" && find . -maxdepth 1 -type f \( -name '*.deb' -o -name '*.AppImage' -o -name '*.zip' \) \
  -printf '%f\n' | LC_ALL=C sort | xargs -r sha256sum > SHA256SUMS)

echo "AppImage:  $ARCHIVE"
echo "checksums: $RELEASE_ROOT/SHA256SUMS"
