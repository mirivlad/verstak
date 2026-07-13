#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

for script in build-linux-bundle.sh package-deb.sh package-appimage.sh package-windows-portable.sh; do
  path="$ROOT/scripts/$script"
  if [[ ! -x "$path" ]]; then
    echo "packaging script is missing or not executable: $path" >&2
    exit 1
  fi
  bash -n "$path"
done

grep -Fq 'dpkg-deb' "$ROOT/scripts/package-deb.sh"
grep -Fq -- '--build' "$ROOT/scripts/package-deb.sh"
grep -Fq 'packaging/deb/control' "$ROOT/scripts/package-deb.sh"
grep -Fq 'libwebkit2gtk-4.1-0' "$ROOT/packaging/deb/control"
grep -Fq 'appimagetool' "$ROOT/scripts/package-appimage.sh"
grep -Fq 'WebKitWebProcess' "$ROOT/scripts/package-appimage.sh"
grep -Fq 'FixedVersionRuntime' "$ROOT/scripts/package-windows-portable.sh"
grep -Fq 'msedgewebview2.exe' "$ROOT/scripts/package-windows-portable.sh"
grep -Fq 'zip -qr' "$ROOT/scripts/package-windows-portable.sh"
grep -Fq 'WebviewBrowserPath' "$ROOT/platform_options_windows.go"
grep -Fq 'webview2' "$ROOT/platform_options_windows.go"
grep -Fq 'chmod -R a+rX' "$ROOT/scripts/build-linux-bundle.sh"

echo "desktop package script contracts passed"
