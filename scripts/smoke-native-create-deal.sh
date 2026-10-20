#!/usr/bin/env bash
# Run after build.sh and staging shipping plugins. Uses only a disposable vault.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${VERSTAK_PROBE_OUT:-$ROOT/build/native-create-deal}"
DISPLAY_NUM="${VERSTAK_PROBE_DISPLAY:-:96}"
PORT="${VERSTAK_PROBE_INSPECTOR_PORT:-9234}"
for tool in Xvfb xdotool import node python3 setsid dbus-run-session; do
  command -v "$tool" >/dev/null || { echo "missing tool: $tool" >&2; exit 1; }
done
mkdir -p "$OUT"
PROBE_HOME="$(mktemp -d)"
VAULT="$PROBE_HOME/vault"
APP_GROUP=""
XVFB_PID=""
cleanup() {
  if [[ -n "$APP_GROUP" ]]; then kill -TERM -- "-$APP_GROUP" 2>/dev/null || true; kill -KILL -- "-$APP_GROUP" 2>/dev/null || true; fi
  [[ -z "$XVFB_PID" ]] || kill "$XVFB_PID" 2>/dev/null || true
  # Private D-Bus services may mount gvfs under this disposable HOME.
  if command -v fusermount3 >/dev/null; then
    fusermount3 -u "$PROBE_HOME/.gvfs" 2>/dev/null || true
    fusermount3 -u "$PROBE_HOME/run/gvfs" 2>/dev/null || true
  fi
  rm -rf "$PROBE_HOME"
}
trap cleanup EXIT
python3 - "$VAULT" "$PROBE_HOME" <<'PY'
import json, pathlib, sys
vault, home = map(pathlib.Path, sys.argv[1:])
for relative in ['.verstak/plugin-data/verstak.templates', '.verstak/plugin-settings', '.verstak/trash', 'Clients/.verstak']:
    (vault / relative).mkdir(parents=True, exist_ok=True)
(home / '.config/verstak').mkdir(parents=True)
(home / 'run').mkdir(mode=0o700)
(vault / '.verstak/vault.json').write_text(json.dumps({'schemaVersion':1,'vaultId':'00000000-0000-4000-8000-0000000000aa','createdAt':'2026-10-21T00:00:00Z','updatedAt':'2026-10-21T00:00:00Z','app':'verstak'}))
(vault / 'Clients/.verstak/folder.json').write_text(json.dumps({'schemaVersion':1,'folderId':'11111111-1111-4111-8111-111111111111'}))
(home / '.config/verstak/config.json').write_text(json.dumps({'currentVaultPath':str(vault),'language':'ru','sidebarWidth':260}))
rows = [dict(id='seed-general', seed=True, name='General', description='A flexible Deal with notes and files.', version=1, workspaceTools=['verstak.notes','verstak.files','verstak.activity'], initialFolders=['Notes','Files'], initialFiles=[], toolConfig={}), dict(id='custom-unavailable', name='My older recipe', description='Preserve custom data', version=7, workspaceTools=['verstak.activity','unknown.plugin'], initialFolders=['Brief'], initialFiles=[], toolConfig={'custom':{'preserve':True}})]
(vault / '.verstak/plugin-data/verstak.templates/templates.ndjson').write_text(''.join(json.dumps(row)+'\n' for row in rows))
PY
Xvfb "$DISPLAY_NUM" -screen 0 1280x960x24 -nolisten tcp > "$OUT/xvfb.log" 2>&1 &
XVFB_PID=$!
sleep 1
kill -0 "$XVFB_PID"
(
  cd "$ROOT/build/bin"
  # JSC's GC signal must not replace Go's SIGUSR1 handler in this test process.
  exec setsid env -i HOME="$PROBE_HOME" PATH="$PATH" DISPLAY="$DISPLAY_NUM" XDG_RUNTIME_DIR="$PROBE_HOME/run" \
    WEBKIT_DISABLE_COMPOSITING_MODE=1 JSC_SIGNAL_FOR_GC=12 WEBKIT_INSPECTOR_HTTP_SERVER="127.0.0.1:$PORT" \
    dbus-run-session -- ./verstak-desktop --debug
) > "$OUT/app.log" 2>&1 &
APP_GROUP=$!
WINDOW=""
for _ in $(seq 1 80); do
  WINDOW="$(DISPLAY="$DISPLAY_NUM" xdotool search --name '^Verstak$' 2>/dev/null | tail -1 || true)"
  [[ -z "$WINDOW" ]] || break
  sleep .25
done
[[ -n "$WINDOW" ]] || { echo 'Native window did not appear; see app.log' >&2; exit 1; }
DISPLAY="$DISPLAY_NUM" node --experimental-websocket "$ROOT/scripts/native-create-deal-test.mjs" "$VAULT" "$OUT" "$WINDOW" "$PORT"
