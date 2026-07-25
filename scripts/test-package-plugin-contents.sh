#!/usr/bin/env bash
# Verifies that packaged plugin trees exclude development-only plugins while
# the development tree keeps them.
#
# The exclusion lives at the packaging boundary, which is easy to bypass by
# adding a new package format that copies plugins/ directly. This checks the
# staging helper itself and, when built artifacts exist, the artifacts.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FAILED=0

report() {
  if [ "$2" -eq 0 ]; then
    echo "  ✅ $1"
  else
    echo "  ❌ $1"
    FAILED=1
  fi
}

echo "=== packaged plugin contents ==="

FIXTURE="$(mktemp -d)"
trap 'rm -rf "$FIXTURE"' EXIT
mkdir -p "$FIXTURE/src/keep" "$FIXTURE/src/drop"
printf '{"id":"keep"}\n' > "$FIXTURE/src/keep/plugin.json"
printf '{"id":"drop","development":true}\n' > "$FIXTURE/src/drop/plugin.json"

"$ROOT/scripts/stage-shipping-plugins.sh" "$FIXTURE/src" "$FIXTURE/out" >/dev/null

[ -d "$FIXTURE/out/keep" ]
report "shipping plugin is staged" $?
[ ! -d "$FIXTURE/out/drop" ]
report "development plugin is excluded" $?

# Every package format must route through the staging helper.
for script in build-linux-bundle.sh build-windows.sh; do
  if grep -qE '^\s*cp -R .*plugins.*\$\{?(OUTPUT|WINDOWS_OUTPUT)' "$ROOT/scripts/$script"; then
    echo "  ❌ $script copies plugins directly instead of staging them"
    FAILED=1
  else
    echo "  ✅ $script stages plugins"
  fi
done

# If a bundle has been built, check the real thing too.
for built in "$ROOT/build/linux-amd64/plugins" "$ROOT/build/windows-amd64/plugins"; do
  [ -d "$built" ] || continue
  offenders=""
  for manifest in "$built"/*/plugin.json; do
    [ -f "$manifest" ] || continue
    if python3 -c "import json,sys; sys.exit(0 if json.load(open(sys.argv[1])).get('development') else 1)" "$manifest"; then
      offenders="$offenders $(basename "$(dirname "$manifest")")"
    fi
  done
  if [ -n "$offenders" ]; then
    echo "  ❌ $built contains development plugins:$offenders"
    FAILED=1
  else
    echo "  ✅ $built ships only user-facing plugins"
  fi
done

echo ""
if [ "$FAILED" -eq 0 ]; then
  echo "✅ packaged plugin contents check passed"
else
  echo "❌ packaged plugin contents check failed"
fi
exit "$FAILED"
