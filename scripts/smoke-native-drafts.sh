#!/usr/bin/env bash
# Exercise editor drafts with real Wails/WebKitGTK, a private HOME and synthetic data.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${VERSTAK_PROBE_OUT:-$ROOT/build/native-drafts}"
BINARY="${VERSTAK_DRAFTS_BINARY:-$ROOT/build/bin/verstak-desktop}"
DISPLAY_NUM="${VERSTAK_PROBE_DISPLAY:-:94}"
PORT="${VERSTAK_PROBE_INSPECTOR_PORT:-9236}"
for tool in Xvfb xdotool import node python3 setsid dbus-run-session; do
  command -v "$tool" >/dev/null || { echo "missing tool: $tool" >&2; exit 1; }
done
[[ -x "$BINARY" ]] || { echo "binary missing: $BINARY" >&2; exit 1; }
mkdir -p "$OUT"
PROBE_HOME="$(mktemp -d)"
VAULT="$PROBE_HOME/vault"
APP_GROUP=""
WINDOW_PID=""
XVFB_PID=""
cleanup() {
  [[ -z "$WINDOW_PID" ]] || { kill -TERM "$WINDOW_PID" 2>/dev/null || true; kill -KILL "$WINDOW_PID" 2>/dev/null || true; }
  [[ -z "$APP_GROUP" ]] || { kill -TERM -- "-$APP_GROUP" 2>/dev/null || true; kill -KILL -- "-$APP_GROUP" 2>/dev/null || true; }
  [[ -z "$XVFB_PID" ]] || kill "$XVFB_PID" 2>/dev/null || true
  if command -v fusermount3 >/dev/null; then
    fusermount3 -u "$PROBE_HOME/run/gvfs" 2>/dev/null || true
  fi
  rm -rf "$PROBE_HOME"
}
trap cleanup EXIT
python3 - "$VAULT" "$PROBE_HOME" <<'PY'
import json,pathlib,sys
vault,home=map(pathlib.Path,sys.argv[1:])
(vault/'.verstak').mkdir(parents=True)
(home/'.config/verstak').mkdir(parents=True)
(home/'run').mkdir(mode=0o700)
(vault/'.verstak/vault.json').write_text(json.dumps({'schemaVersion':1,'vaultId':'00000000-0000-4000-8000-00000000aa99','createdAt':'2026-10-22T00:00:00Z','updatedAt':'2026-10-22T00:00:00Z','app':'verstak'}))
(home/'.config/verstak/config.json').write_text(json.dumps({'currentVaultPath':str(vault),'language':'ru','theme':'light'}))
PY
Xvfb "$DISPLAY_NUM" -screen 0 1280x960x24 -nolisten tcp > "$OUT/xvfb.log" 2>&1 &
XVFB_PID=$!
sleep 1
kill -0 "$XVFB_PID"
(
  cd "$(dirname "$BINARY")"
  exec setsid env -i HOME="$PROBE_HOME" PATH="$PATH" DISPLAY="$DISPLAY_NUM" XDG_RUNTIME_DIR="$PROBE_HOME/run" \
    WEBKIT_DISABLE_COMPOSITING_MODE=1 JSC_SIGNAL_FOR_GC=12 WEBKIT_INSPECTOR_HTTP_SERVER="127.0.0.1:$PORT" \
    dbus-run-session -- "$BINARY" --debug
) > "$OUT/app.log" 2>&1 &
APP_GROUP=$!
WINDOW=""
for _ in $(seq 1 80); do
  WINDOW="$(DISPLAY="$DISPLAY_NUM" xdotool search --name '^Verstak$' 2>/dev/null | tail -1 || true)"
  [[ -z "$WINDOW" ]] || break
  sleep .25
done
[[ -n "$WINDOW" ]] || { echo 'Native window did not appear; see app.log' >&2; exit 1; }
WINDOW_PID="$(DISPLAY="$DISPLAY_NUM" xdotool getwindowpid "$WINDOW")"
DISPLAY="$DISPLAY_NUM" node --experimental-websocket "$ROOT/scripts/native-drafts-test.mjs" "$VAULT" "$OUT" "$WINDOW" "$PORT"
