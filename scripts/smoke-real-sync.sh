#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DESKTOP_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
SYNC_SERVER_DIR="$(cd "$DESKTOP_DIR/../verstak-sync-server" && pwd)"

PORT="${VERSTAK_SYNC_SMOKE_PORT:-47733}"
DATA_DIR="$(mktemp -d)"
LOG_FILE="$DATA_DIR/server.log"
SERVER_PID=""

cleanup() {
  if [[ -n "$SERVER_PID" ]] && kill -0 "$SERVER_PID" 2>/dev/null; then
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
  rm -rf "$DATA_DIR"
}
trap cleanup EXIT

if ss -ltn | grep -q ":$PORT "; then
  echo "port $PORT is already in use" >&2
  exit 1
fi

(
  cd "$SYNC_SERVER_DIR"
  go run ./cmd/server --port "$PORT" --data "$DATA_DIR"
) >"$LOG_FILE" 2>&1 &
SERVER_PID="$!"

# A cold Go cache can take longer than 20 seconds to compile `go run`; wait
# for the real listener rather than treating a still-running compiler as a
# healthy server.
for _ in $(seq 1 300); do
  if curl -fsS "http://127.0.0.1:$PORT/api/v1/health" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "$SERVER_PID" 2>/dev/null; then
    cat "$LOG_FILE" >&2
    exit 1
  fi
  sleep 0.25
done

curl -fsS "http://127.0.0.1:$PORT/api/v1/health" >/dev/null

NOW="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
sqlite3 "$DATA_DIR/server.db" "
INSERT INTO server_users (id, username, email, password_hash, confirmed, created_at)
VALUES ('smoke-user', 'smoke-user', 'smoke@example.test', 'unused', 1, '$NOW');
INSERT INTO server_devices (id, name, api_key, legacy_api_key, user_id, vault_id, last_seen, created_at)
VALUES ('smoke-device-a', 'Smoke Device A', 'smoke-key-a', 1, 'smoke-user', 'smoke-vault', '$NOW', '$NOW');
INSERT INTO server_devices (id, name, api_key, legacy_api_key, user_id, vault_id, last_seen, created_at)
VALUES ('smoke-device-b', 'Smoke Device B', 'smoke-key-b', 1, 'smoke-user', 'smoke-vault', '$NOW', '$NOW');
"

(
  cd "$DESKTOP_DIR"
  VERSTAK_SYNC_SMOKE_SERVER_URL="http://127.0.0.1:$PORT" \
  VERSTAK_SYNC_SMOKE_DEVICE_A="smoke-device-a" \
  VERSTAK_SYNC_SMOKE_DEVICE_B="smoke-device-b" \
  VERSTAK_SYNC_SMOKE_KEY_A="smoke-key-a" \
  VERSTAK_SYNC_SMOKE_KEY_B="smoke-key-b" \
  go test ./internal/api -run TestSyncNowAgainstRealServerTwoVaults -count=1 -v
)
