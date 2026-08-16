# GUI Testing

## Overview

Verstak has two complementary GUI test layers:

1. **Playwright + Chromium** for deterministic frontend behavior with mocked
   Wails bindings and real official plugin bundles.
2. **Real Wails + WebKitGTK under Xvfb** for native desktop rendering and
   startup smoke evidence.

Neither layer replaces the other. Playwright is the fast, controllable product
flow test. The Wails probe verifies that the application we actually ship can be
built, started and rendered by the Linux desktop engine.

## Frontend E2E (Playwright)

Tests live in `frontend/e2e/` and run with:

```bash
cd frontend
npm run test:e2e
```

The test-mode Vite server loads `src/lib/test/wails-mock.js`, which provides
in-memory Wails API implementations. The desktop CI job checks out and uses the
current `verstak-official-plugins` bundles, so plugin screens are exercised from
real plugin frontend code rather than shell-owned stand-ins.

Coverage includes, among other flows:

- plugin enable/disable and contribution refresh;
- sidebar and workspace navigation;
- Overview/resume flow;
- Notes, Files, Todo, Activity, Journal, Browser Inbox and Trash;
- default editor and Workbench routing;
- global search and command palette;
- Settings and diagnostics;
- workspace templates, tree overlays, resize and drag-and-drop;
- localization and plugin API bridge contracts.

Playwright also collects console/page errors. On CI failures, screenshots,
traces, video-on-retry and JSON results are kept under `frontend/e2e-results/`
and uploaded as the `e2e-results` Actions artifact.

## Real desktop GUI (Wails + WebKitGTK)

`scripts/gui-probe.sh` starts the built `build/bin/verstak-desktop` on an Xvfb
virtual display with an isolated temporary `HOME` and a synthetic vault. It
waits for the real window with `xdotool`, captures the X11 root window with
ImageMagick, and writes evidence under:

```text
build/gui-probe/
  startup.png
  app.log
  xvfb.log
```

The probe launches from the binary directory. This is deliberate: plugin
discovery supports both a developer `./plugins` directory and plugins beside the
executable. Running a packaged binary from the source checkout would discover
the same plugins twice and would not represent an installed application.

Run it locally after a build with:

```bash
bash scripts/gui-probe.sh --shot startup
```

Use `--keep` when manual `xdotool` interaction or additional screenshots are
needed:

```bash
bash scripts/gui-probe.sh --keep --shot startup
```

### GitHub Actions GUI audit

`.github/workflows/gui-audit.yml` runs the same real-desktop probe on Ubuntu. It:

- checks out the desktop and official-plugin repositories;
- builds the current official plugins;
- builds the real Wails application with the Wails CLI version paired to
  `go.mod`;
- replaces the build output with the **shipping** plugin set, excluding manifests
  marked `development: true`;
- launches the application under Xvfb;
- uploads screenshot and logs as a `gui-audit-*` artifact even if the probe
  fails.

The workflow runs for pull requests and can also be started manually. A manual
run may supply another official-plugin git ref when validating coordinated
cross-repository work.

## What the two layers do not prove

The Xvfb probe is genuine Wails/WebKitGTK rendering, but a headless CI machine
still cannot fully validate desktop integration that depends on the user's
session. In particular, final release smoke on a real desktop remains useful
for:

- tray integration with the actual desktop environment;
- native file dialogs and window-manager behavior;
- clipboard integration across real applications;
- physical DPI/font/theme differences;
- long interactive workflows that are better inspected by a person.

The GitHub Actions screenshots are therefore valid visual evidence for layout
and WebKitGTK rendering, not a claim that every Linux desktop integration has
been exercised.

## Running Playwright interactively

```bash
cd frontend

# All E2E tests, headless
npm run test:e2e

# Playwright UI
npm run test:e2e:ui

# Headed Chromium
npm run test:e2e:headed
```

## Playwright infrastructure

### Mock bridge (`src/lib/test/wails-mock.js`)

Provides mutable plugin, vault, contribution and storage state plus test helpers
through `window.__wailsMock`. This lets a test construct deterministic product
states without editing a real vault.

### Test harness (`index.html`)

Production and test use the same page. When the Wails runtime is absent, test
mode loads the mock bridge before the Svelte application.

### Playwright config (`playwright.config.js`)

- Vite test server on port 5174;
- Chromium headless by default;
- 30 second per-test timeout, 10 second assertion timeout;
- one worker for deterministic shared mock state;
- screenshots on failure;
- trace and video on first retry;
- JSON results under `e2e-results/`.

## Adding coverage

For behavior, add or extend an E2E scenario in `frontend/e2e/` and assert user
visible outcomes rather than implementation details. For a rendering bug that
may be WebKitGTK-specific, reproduce it with `scripts/gui-probe.sh` as well and
keep the screenshot/log evidence with the change or CI run.
