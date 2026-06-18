# GUI Testing

## Overview

Verstak Desktop uses **Playwright** for frontend E2E tests that run in a real
Chromium browser with mocked Wails bindings. This tests the Svelte component
logic, user interactions, and UI state transitions — without needing the actual
Wails desktop shell.

## What is tested

### Frontend E2E (Playwright)

Located in `frontend/e2e/`, run via `npm run test:e2e`.

These tests:

- Launch a Vite dev server with mock Wails bindings
- Open the app in a real Chromium browser via Playwright
- Simulate user clicks, wait for UI transitions, assert DOM state
- Collect console errors and page errors on failure
- Capture screenshots on failure

### Test suites

| File | Suite | Tests | Status |
|------|-------|-------|--------|
| `plugin-manager-disable-enable.spec.js` | A: Disable/Enable refresh | 4 | 3 pass, 1 fail* |
| `sidebar-opens-view.spec.js` | B: Sidebar → view routing | 3 | 3 pass |
| `reload-updates-state.spec.js` | C: Reload updates UI | 3 | 2 pass, 1 fail* |

\* Failing tests document **known bugs** (see below).

## Known bugs detected by tests

### Bug M5-1: Sidebar does not update when plugin state changes

**Symptom:** After disabling a plugin in Plugin Manager, the sidebar item for
that plugin remains visible. After re-enabling, it stays visible (doesn't
disappear then reappear — it was never gone).

**Root cause:** `Sidebar.svelte` loads plugin/contribution data once in
`onMount` and stores it in local `sidebarItems`. When `PluginManager`
disables/enables a plugin and calls `ReloadPlugins`, the `PluginManager`
component re-fetches data, but `Sidebar` does not react to the change — it
still holds the stale list.

**Affected tests:**
- `A: Disable plugin: button changes to Enable, sidebar item disappears`
- `A: Disable → Enable full flow in sequence`
- `C: Reload after mock state change reflects new plugin status`

**Fix needed:** Sidebar must either:
1. Re-fetch contributions when it receives a custom event (e.g.
   `verstak:plugins-reloaded`), or
2. Read plugin state reactively from a shared store that both
   PluginManager and Sidebar subscribe to.

## What is NOT tested

### Real desktop GUI (WebKitGTK + Wails native shell)

The Playwright tests run the frontend in a **standard Chromium browser** with
mocked Wails bindings. They do **not** test:

- Actual WebKitGTK rendering (Wails uses WebKitGTK, not Chromium)
- Native window management (minimize, maximize, resize)
- Native file dialogs (SelectDirectory, SelectVaultForOpen)
- Clipboard integration
- System tray / menu bar
- Plugin frontend bundle loading from real filesystem
- Wails event system (window.runtime.EventsOn/Emit)

For real Wails smoke tests, a separate layer is needed using:
- **AT-SPI2** (Linux accessibility tree inspection)
- **xdotool** / **ydotool** (input simulation)
- **scrot** / **import** (screenshot capture)

## Running tests

```bash
cd frontend

# Run all E2E tests (headless)
npm run test:e2e

# Run with Playwright UI (interactive)
npm run test:e2e:ui

# Run in headed browser (visible)
npm run test:e2e:headed
```

## Test infrastructure

### Mock bridge (`src/lib/test/wails-mock.js`)

Replaces `window['go']['api']['App']` with in-memory mock implementations of
all Wails backend methods. Provides:

- Mutable plugin state (enable/disable/status)
- Mutable vault state
- Mutable contributions (views, commands, sidebar items, settings panels)
- Test helpers via `window.__wailsMock`:
  - `reset()` — reset all state to defaults
  - `setPluginStatus(id, status, enabled)` — change plugin state
  - `getPluginState(id)` — read current state
  - `setVaultStatus(status)` — change vault state

### Test harness (`index.html`)

The same `index.html` is used for both production and test. It detects whether
the Wails runtime (`window['go']`) is present. If not (i.e. running in a plain
browser), it loads the mock bridge before the Svelte app.

### Playwright config (`playwright.config.js`)

- Dev server: `vite --mode test --port 5174`
- Browser: Chromium headless
- Timeouts: 30s test, 10s expect
- Workers: 1 (sequential)
- Screenshots: on failure
- Traces: on first retry
- Results: `e2e-results/test-results.json`

## Adding new tests

1. Create `e2e/your-test.spec.js`
2. Import helpers from `./helpers.js`
3. Use `test.beforeEach` to reset mock state and navigate to `/`
4. Use `test.afterEach` to assert no console errors
5. Write scenarios as user actions + assertions
6. Run with `npm run test:e2e`

### Selector conventions

- Plugin cards: `.plugin-card` filtered by text
- Buttons: `.btn-disable`, `.btn-enable`, `.btn-settings`, `.reload-btn`
- Sidebar items: `.sidebar .plugin-item`
- View container: `.view-container`
- View header: `.view-header h2`
- Status badges: `.status-badge`
- Toast: `.toast`
