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
grep -Fq 'libayatana-appindicator3-1' "$ROOT/packaging/deb/control"
grep -Fq 'appimagetool' "$ROOT/scripts/package-appimage.sh"
grep -Fq 'WebKitWebProcess' "$ROOT/scripts/package-appimage.sh"
grep -Fq 'libayatana-appindicator3.so.' "$ROOT/scripts/package-appimage.sh"
grep -Fq 'ayatana-appindicator3-0.1' "$ROOT/scripts/build.sh"
if grep -Fq 'FixedVersionRuntime' "$ROOT/scripts/package-windows-portable.sh"; then
  echo "Windows portable archive must not bundle Fixed Version WebView2" >&2
  exit 1
fi
if grep -Fq 'msedgewebview2.exe' "$ROOT/scripts/package-windows-portable.sh"; then
  echo "Windows portable archive must not bundle WebView2 binaries" >&2
  exit 1
fi
if [[ -e "$ROOT/platform_options_windows.go" ]]; then
  echo "Windows runtime override must not be present" >&2
  exit 1
fi
grep -Fq 'zip -qr' "$ROOT/scripts/package-windows-portable.sh"
grep -Fq 'LinkId=2124701' "$ROOT/README.md"
grep -Fq 'WebView2 Runtime' "$ROOT/README.md"
grep -Fq 'package-deb.sh' "$ROOT/scripts/release.sh"
grep -Fq 'package-appimage.sh' "$ROOT/scripts/release.sh"
grep -Fq 'package-windows-portable.sh' "$ROOT/scripts/release.sh"
git -C "$ROOT" check-ignore -q verstak-desktop-res.syso
grep -Fq 'chmod -R a+rX' "$ROOT/scripts/build-linux-bundle.sh"

echo "desktop package script contracts passed"
