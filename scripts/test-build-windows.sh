#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BUILDER="$ROOT/scripts/build-windows.sh"

if [[ ! -x "$BUILDER" ]]; then
  echo "Windows desktop builder is missing or not executable: $BUILDER" >&2
  exit 1
fi

bash -n "$BUILDER"
grep -Fq -- '-platform windows/amd64' "$BUILDER"
grep -Fq -- '-compiler "$WINDOWS_CC"' "$BUILDER"
grep -Fq 'dist-windows' "$BUILDER"
grep -Fq 'verstak-desktop.exe' "$BUILDER"
grep -Fq 'x86_64-w64-mingw32-gcc' "$BUILDER"

echo "Windows desktop build script contract passed"
