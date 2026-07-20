#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OFFICIAL_PLUGINS="${VERSTAK_OFFICIAL_PLUGINS_DIR:-$ROOT/../verstak-official-plugins}"
WINDOWS_PLUGIN_DIST="${VERSTAK_WINDOWS_PLUGIN_DIST:-$OFFICIAL_PLUGINS/dist-windows}"
WINDOWS_OUTPUT="${VERSTAK_WINDOWS_OUTPUT_DIR:-$ROOT/build/windows-amd64}"
WINDOWS_CC="${VERSTAK_WINDOWS_CC:-x86_64-w64-mingw32-gcc}"

if ! command -v "$WINDOWS_CC" >/dev/null; then
  echo "Windows cross-compiler not found: $WINDOWS_CC" >&2
  echo "Install x86_64-w64-mingw32-gcc (for example: sudo apt install gcc-mingw-w64-x86-64)." >&2
  exit 1
fi
if [[ ! -d "$OFFICIAL_PLUGINS" ]]; then
  echo "official plugins repository not found: $OFFICIAL_PLUGINS" >&2
  exit 1
fi

WAILS="${WAILS_BIN:-}"
if [[ -z "$WAILS" ]]; then
  if command -v wails >/dev/null; then
    WAILS="wails"
  elif [[ -n "$(go env GOBIN 2>/dev/null)" && -x "$(go env GOBIN)/wails" ]]; then
    WAILS="$(go env GOBIN)/wails"
  elif [[ -x "$(go env GOPATH)/bin/wails" ]]; then
    WAILS="$(go env GOPATH)/bin/wails"
  fi
fi
if [[ -z "$WAILS" ]]; then
  echo "Wails CLI is required. Install it with: go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0" >&2
  exit 1
fi

cd "$ROOT"
echo "=== verstak desktop Windows amd64 build ==="

"$ROOT/scripts/generate-brand-icons.sh"

if [[ ! -d "$ROOT/frontend/node_modules" ]]; then
  if [[ -f "$ROOT/frontend/package-lock.json" ]]; then
    (cd "$ROOT/frontend" && npm ci --no-audit --no-fund)
  else
    (cd "$ROOT/frontend" && npm install --no-audit --no-fund)
  fi
fi
(cd "$ROOT/frontend" && npm run build)

go mod download
go test -count=1 ./...

"$OFFICIAL_PLUGINS/scripts/build-windows.sh"
if [[ ! -d "$WINDOWS_PLUGIN_DIST" ]]; then
  echo "Windows plugin packages were not produced: $WINDOWS_PLUGIN_DIST" >&2
  exit 1
fi
VERSTAK_RELEASE_PLUGIN_DIR="$WINDOWS_PLUGIN_DIST" go test ./internal/core/plugin -run TestBundledOfficialPluginRequirementsResolve -count=1

# Wails' -compiler option selects a Go binary, not a C compiler. Cross-CGO
# therefore has to be supplied through the standard Go environment instead.
# The portable archive uses the installed Evergreen WebView2 Runtime, so never
# compile an Evergreen downloader into the executable.
CC="$WINDOWS_CC" CGO_ENABLED=1 GOFLAGS="${GOFLAGS:+$GOFLAGS }-buildvcs=false" "$WAILS" build -clean -platform windows/amd64 \
  -webview2 error -o verstak-desktop.exe

WINDOWS_BINARY="$ROOT/build/bin/verstak-desktop.exe"
if [[ ! -f "$WINDOWS_BINARY" ]]; then
  echo "Windows executable was not produced: $WINDOWS_BINARY" >&2
  exit 1
fi

rm -rf "$WINDOWS_OUTPUT"
mkdir -p "$WINDOWS_OUTPUT/plugins"
cp "$WINDOWS_BINARY" "$WINDOWS_OUTPUT/verstak-desktop.exe"
cp -R "$WINDOWS_PLUGIN_DIST/." "$WINDOWS_OUTPUT/plugins/"

echo "Windows test bundle: $WINDOWS_OUTPUT"
echo "Copy this directory to Windows and run verstak-desktop.exe."
