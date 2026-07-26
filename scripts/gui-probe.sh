#!/usr/bin/env bash
# Runs the real WebKitGTK application on a virtual display so its rendering can
# be looked at, not guessed at.
#
# The Playwright suite drives the Svelte shell in Chromium, which is the wrong
# engine for anything that depends on how the toolkit paints: native scrollbar
# placement, compositing order, GTK overlay widgets. Two scrollbar fixes were
# shipped blind before this existed.
#
# Everything happens inside a throwaway HOME, so the real vault and settings
# are never touched.
#
#   scripts/gui-probe.sh                       # launch, screenshot, exit
#   scripts/gui-probe.sh --keep                # leave it running for xdotool
#   scripts/gui-probe.sh --shot NAME           # name the screenshot
#
# Output goes to build/gui-probe/.
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${VERSTAK_PROBE_OUT:-$ROOT/build/gui-probe}"
BINARY="$ROOT/build/bin/verstak-desktop"
DISPLAY_NUM="${VERSTAK_PROBE_DISPLAY:-:97}"
GEOMETRY="${VERSTAK_PROBE_GEOMETRY:-1200x820x24}"
SHOT_NAME="startup"
KEEP=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --keep) KEEP=1 ;;
    --shot) SHOT_NAME="${2:?--shot needs a name}"; shift ;;
    *) echo "unknown option: $1" >&2; exit 2 ;;
  esac
  shift
done

if [ ! -x "$BINARY" ]; then
  echo "desktop binary not built: $BINARY" >&2
  echo "run scripts/build.sh first" >&2
  exit 1
fi

for tool in Xvfb xdotool import; do
  command -v "$tool" >/dev/null || { echo "missing tool: $tool" >&2; exit 1; }
done

mkdir -p "$OUT"
PROBE_HOME="$(mktemp -d)"
VAULT="$PROBE_HOME/vault"

cleanup() {
  [ -n "${APP_PID:-}" ] && kill "$APP_PID" 2>/dev/null
  [ -n "${XVFB_PID:-}" ] && kill "$XVFB_PID" 2>/dev/null
  [ "$KEEP" -eq 0 ] && rm -rf "$PROBE_HOME"
  return 0
}
[ "$KEEP" -eq 0 ] && trap cleanup EXIT

# ── A vault with enough shape to reproduce tree problems ────────────────────
# Folders deep enough that the sidebar must scroll, which is the only way the
# scrollbar is on screen at all.
# The application only opens a directory it recognises as a vault, which means
# the .verstak layout has to exist before the config points at it. Without this
# the probe just photographs the vault-selection screen.
mkdir -p "$VAULT"/.verstak/{plugin-data,plugin-settings,plugin-cache,trash,logs}
NOW="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
cat > "$VAULT/.verstak/vault.json" <<VAULTJSON
{
  "schemaVersion": 1,
  "vaultId": "00000000-0000-4000-8000-0000000000aa",
  "createdAt": "$NOW",
  "updatedAt": "$NOW",
  "app": "verstak"
}
VAULTJSON
make_deal() {
  local path="$1"
  mkdir -p "$VAULT/$path/Notes" "$VAULT/$path/Files"
  printf '# %s\n' "$(basename "$path")" > "$VAULT/$path/Notes/Overview.md"
}
for folder in Alpha Beta Gamma; do
  for deal in one two three four five; do
    make_deal "$folder/$folder-$deal"
  done
done
for deal in loose-one loose-two loose-three; do
  make_deal "$deal"
done

mkdir -p "$PROBE_HOME/.config/verstak"
cat > "$PROBE_HOME/.config/verstak/config.json" <<EOF
{"currentVaultPath":"$VAULT","language":"ru","sidebarWidth":260}
EOF

# ── Display ─────────────────────────────────────────────────────────────────
Xvfb "$DISPLAY_NUM" -screen 0 "$GEOMETRY" -nolisten tcp >"$OUT/xvfb.log" 2>&1 &
XVFB_PID=$!
sleep 1
if ! kill -0 "$XVFB_PID" 2>/dev/null; then
  echo "Xvfb failed to start; see $OUT/xvfb.log" >&2
  exit 1
fi

# ── Application ─────────────────────────────────────────────────────────────
env -i \
  HOME="$PROBE_HOME" \
  DISPLAY="$DISPLAY_NUM" \
  PATH="/usr/bin:/bin" \
  XDG_RUNTIME_DIR="$PROBE_HOME/run" \
  WEBKIT_DISABLE_COMPOSITING_MODE="${WEBKIT_DISABLE_COMPOSITING_MODE:-0}" \
  "$BINARY" --debug >"$OUT/app.log" 2>&1 &
APP_PID=$!

# Wait for the window rather than sleeping a fixed amount.
WINDOW=""
for _ in $(seq 1 60); do
  WINDOW="$(DISPLAY="$DISPLAY_NUM" xdotool search --name 'Verstak' 2>/dev/null | tail -1)"
  [ -n "$WINDOW" ] && break
  sleep 0.5
done
if [ -z "$WINDOW" ]; then
  echo "the application window never appeared; see $OUT/app.log" >&2
  tail -20 "$OUT/app.log" >&2
  exit 1
fi

DISPLAY="$DISPLAY_NUM" xdotool windowactivate "$WINDOW" 2>/dev/null
sleep 3
DISPLAY="$DISPLAY_NUM" import -window root "$OUT/$SHOT_NAME.png"

echo "window:     $WINDOW"
echo "display:    $DISPLAY_NUM"
echo "home:       $PROBE_HOME"
echo "vault:      $VAULT"
echo "screenshot: $OUT/$SHOT_NAME.png"
echo "app log:    $OUT/app.log"

if [ "$KEEP" -eq 1 ]; then
  echo ""
  echo "Left running. Drive it with:"
  echo "  DISPLAY=$DISPLAY_NUM xdotool mousemove X Y click 3"
  echo "  DISPLAY=$DISPLAY_NUM import -window root $OUT/next.png"
  echo "Stop it with: kill $APP_PID $XVFB_PID; rm -rf $PROBE_HOME"
fi
