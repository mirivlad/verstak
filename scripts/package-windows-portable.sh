#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:-}"
WEBVIEW2_CAB="${VERSTAK_WEBVIEW2_CAB:-}"
WEBVIEW2_URL="${VERSTAK_WEBVIEW2_URL:-}"

if [[ -z "$VERSION" || ! "$VERSION" =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]]; then
  echo "usage: $0 <version>" >&2
  echo "example: $0 v0.1.0-alpha.1" >&2
  exit 2
fi
for command in cabextract curl python3 zip; do
  if ! command -v "$command" >/dev/null; then
    echo "$command is required to create the Windows portable archive" >&2
    exit 1
  fi
done

if [[ -z "$WEBVIEW2_CAB" ]]; then
  if [[ -z "$WEBVIEW2_URL" ]]; then
    WEBVIEW2_URL="$(python3 - <<'PY'
import re
import sys
import urllib.request

with urllib.request.urlopen('https://developer.microsoft.com/en-us/microsoft-edge/webview2/', timeout=30) as response:
    page = response.read().decode('utf-8')
page = page.replace(r'\u002F', '/')
match = re.search(r'"x64","(https://[^\"]+Microsoft\.WebView2\.FixedVersionRuntime[^\"]+\.x64\.cab)"', page)
if not match:
    sys.exit('could not locate the x64 FixedVersionRuntime CAB on the Microsoft WebView2 download page')
print(match.group(1))
PY
)"
  fi
  WEBVIEW2_CAB="$ROOT/build/downloads/$(basename "${WEBVIEW2_URL%%\?*}")"
  mkdir -p "$(dirname "$WEBVIEW2_CAB")"
  if [[ ! -f "$WEBVIEW2_CAB" ]]; then
    curl --fail --location --show-error "$WEBVIEW2_URL" -o "$WEBVIEW2_CAB"
  fi
fi

if [[ ! -f "$WEBVIEW2_CAB" ]]; then
  echo "FixedVersionRuntime CAB was not found: $WEBVIEW2_CAB" >&2
  exit 1
fi

"$ROOT/scripts/build-windows.sh"

RUNTIME_EXTRACT="$ROOT/build/webview2-runtime"
rm -rf "$RUNTIME_EXTRACT"
mkdir -p "$RUNTIME_EXTRACT"
cabextract -q -d "$RUNTIME_EXTRACT" "$WEBVIEW2_CAB"
WEBVIEW2_RUNTIME_DIR="$(dirname "$(find "$RUNTIME_EXTRACT" -type f -iname msedgewebview2.exe -print -quit)")"
if [[ -z "$WEBVIEW2_RUNTIME_DIR" || ! -f "$WEBVIEW2_RUNTIME_DIR/msedgewebview2.exe" ]]; then
  echo "FixedVersionRuntime CAB does not contain msedgewebview2.exe" >&2
  exit 1
fi

RELEASE_ROOT="$ROOT/release"
STAGING="$RELEASE_ROOT/verstak-windows-amd64-$VERSION"
ARCHIVE="$RELEASE_ROOT/verstak-windows-amd64-$VERSION.zip"
rm -rf "$STAGING" "$ARCHIVE"
mkdir -p "$STAGING"
cp -R "$ROOT/build/windows-amd64/." "$STAGING/"
install -m 644 "$ROOT/README.md" "$ROOT/LICENSE" "$STAGING/"
install -m 644 "$ROOT/packaging/windows/Verstak.cmd" "$STAGING/Verstak.cmd"
cp -R "$WEBVIEW2_RUNTIME_DIR" "$STAGING/webview2"

(cd "$RELEASE_ROOT" && zip -qr "$(basename "$ARCHIVE")" "$(basename "$STAGING")")
(cd "$RELEASE_ROOT" && find . -maxdepth 1 -type f \( -name '*.deb' -o -name '*.AppImage' -o -name '*.zip' \) \
  -printf '%f\n' | LC_ALL=C sort | xargs -r sha256sum > SHA256SUMS)

echo "Windows portable archive: $ARCHIVE"
echo "checksums:                $RELEASE_ROOT/SHA256SUMS"
