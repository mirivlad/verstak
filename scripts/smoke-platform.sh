#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "=== verstak-desktop: smoke-platform ==="

# ── verify platform-test is installed ──
PLUGIN_DIR="$ROOT/plugins/platform-test"
if [ ! -d "$PLUGIN_DIR" ]; then
  echo "❌ platform-test not installed at $PLUGIN_DIR"
  echo "   Run: ./scripts/install-dev-plugins.sh"
  exit 1
fi
echo "  ✅ plugin directory: $PLUGIN_DIR"

# ── validate plugin.json ──
if [ ! -f "$PLUGIN_DIR/plugin.json" ]; then
  echo "  ❌ plugin.json not found"
  exit 1
fi

if command -v python3 &>/dev/null; then
  python3 -c "
import json
with open('$PLUGIN_DIR/plugin.json') as f:
    m = json.load(f)
checks = {
    'id': m.get('id') == 'verstak.platform-test',
    'name': m.get('name') == 'Platform Test',
    'version': m.get('version') == '0.1.0',
    'apiVersion': m.get('apiVersion') == '0.1.0',
    'schemaVersion': m.get('schemaVersion') == 1,
    'provides': 'verstak/platform-test/v1' in m.get('provides', []),
    'permissions': 'vault.read' in m.get('permissions', []),
    'frontend.entry': m.get('frontend', {}).get('entry') == 'frontend/dist/index.js',
    'contributes.views': len(m.get('contributes', {}).get('views', [])) > 0,
    'contributes.commands': len(m.get('contributes', {}).get('commands', [])) > 0,
    'contributes.settingsPanels': len(m.get('contributes', {}).get('settingsPanels', [])) > 0,
    'permissions.storage': 'storage.namespace' in m.get('permissions', []),
    'permissions.ui': 'ui.register' in m.get('permissions', []),
}
all_ok = True
for name, ok in checks.items():
    print(f\"  {'✅' if ok else '❌'} manifest.{name}\")
    if not ok:
        all_ok = False
if not all_ok:
    exit(1)
" 2>&1 || { echo "  ❌ manifest validation failed"; exit 1; }
  echo "  ✅ manifest validation passed"
else
  echo "  ℹ️  python3 not available — skipping manifest validation"
fi

# ── run Go smoke command ──
echo ""
echo "[go smoke]"
(cd "$ROOT" && go run -mod=mod ./cmd/smoke-platform/ 2>&1)
SMOKE_EXIT=$?
if [ "$SMOKE_EXIT" -ne 0 ]; then
  echo "  ❌ smoke-platform: Go verification failed"
  exit 1
fi

# ── test enable/disable via Go smoke ──
echo ""
echo "[go smoke: enable/disable]"
(cd "$ROOT" && go run -mod=mod ./cmd/smoke-platform/ -test-enable-disable 2>&1)
SMOKE_ED_EXIT=$?
if [ "$SMOKE_ED_EXIT" -ne 0 ]; then
  echo "  ❌ smoke-platform: enable/disable test failed"
  exit 1
fi

# ── test workspace via Go smoke ──
echo ""
echo "[go smoke: workspace]"
(cd "$ROOT" && go run -mod=mod ./cmd/smoke-platform/ -test-workspace 2>&1)
SMOKE_WS_EXIT=$?
if [ "$SMOKE_WS_EXIT" -ne 0 ]; then
  echo "  ❌ smoke-platform: workspace test failed"
  exit 1
fi

# ── test contributions via Go smoke ──
echo ""
echo "[go smoke: contributions]"
(cd "$ROOT" && go run -mod=mod ./cmd/smoke-platform/ -test-contributions 2>&1)
SMOKE_CONT_EXIT=$?
if [ "$SMOKE_CONT_EXIT" -ne 0 ]; then
  echo "  ❌ smoke-platform: contributions test failed"
  exit 1
fi

# ── bundle host test (JS smoke for error boundary + real mount proof) ──
echo ""
echo "[bundle host test]"
if ! command -v node &>/dev/null; then
  echo "  ⚠️  node not found — skipping bundle-host-test"
else
  (node "$ROOT/frontend/tests/bundle-host-test.cjs" 2>&1)
  BUNDLE_HOST_EXIT=$?
  if [ "$BUNDLE_HOST_EXIT" -ne 0 ]; then
    echo "  ❌ bundle-host-test failed"
    exit 1
  fi
fi

echo ""
echo "✅ smoke-platform all tests done"
