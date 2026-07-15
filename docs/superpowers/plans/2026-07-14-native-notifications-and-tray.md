# Native Notifications, Tray, and Public README Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver native Todo reminders and Windows/Linux tray operation, then publish updated alpha artifacts and public README documentation with real application screenshots.

**Architecture:** Desktop core stores and delivers plugin-owned notification schedules and controls the tray lifecycle. Todo uses the public plugin API to replace its desired reminders. A small core tray adapter keeps the Wails process alive after window close. Documentation uses the supplied English and Russian README sources plus screenshots from an actual test vault.

**Tech Stack:** Go 1.24, Wails v2.12 runtime notifications, `fyne.io/systray`, plain JavaScript plugin API, Node smoke tests, Playwright, bash packaging.

## Global constraints

- Support Windows and Linux only; no background daemon after an explicit Quit.
- Closing a window hides it only after the tray reports ready; otherwise normal
  window close ends the process.
- `verstak.todo` requires `verstak/core/notifications/v1` and `notifications.schedule`.
- Plugins cannot receive Wails runtime access.
- Keep Wails generated bindings out of commits.
- Commit each independent change and immediately push it to GitHub and `mirror`.

---

### Task 1: Persisted notification scheduler

**Files:**
- Create: `internal/core/notifications/manager.go`
- Create: `internal/core/notifications/store.go`
- Create: `internal/core/notifications/manager_test.go`

**Interfaces:**
- `type Request struct { ID, DueAt, Title, Body string }`
- `type Item struct { PluginID, ID, DueAt, Title, Body, SentForDueAt string }`
- `type Sender interface { Send(context.Context, Item) error }`
- `Manager.Replace(pluginID string, requests []Request) error`, `Clear(pluginID string) error`, `Start(context.Context)`, and `Stop()`.
- Persistent state: `<vault>/.verstak/notifications/schedules.json`.

- [ ] **Step 1: Add failing scheduler tests**

```go
func TestTickRetriesFailedSendAndAcknowledgesOneDelivery(t *testing.T) {
    sender := &fakeSender{err: errors.New("unavailable")}
    m := newTestManager(t, sender, time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC))
    requireNoError(t, m.Replace("verstak.todo", []Request{{ID: "todo-1", DueAt: "2026-07-14T11:00:00Z", Title: "Reminder"}}))
    m.Tick(context.Background())
    if sender.calls != 1 || m.Items()[0].SentForDueAt != "" { t.Fatal("failed send was acknowledged") }
    sender.err = nil
    m.Tick(context.Background())
    m.Tick(context.Background())
    if sender.calls != 2 || m.Items()[0].SentForDueAt != "2026-07-14T11:00:00Z" { t.Fatal("delivery was not exactly once") }
}
```

Also test stale records are removed by `Replace`, changing `DueAt` rearms a record, state reloads from disk, and a past-due unsent item is sent once.

- [ ] **Step 2: Confirm red**

Run: `GOCACHE=/tmp/verstak-go-cache go test ./internal/core/notifications -count=1`

Expected: fails because the package does not exist.

- [ ] **Step 3: Implement the minimal manager**

Use one mutex, injected clock/sender, an atomic same-directory temp-file rename, and a 30-second ticker. Preserve `SentForDueAt` only if the requested due time is unchanged. Validate non-empty unique IDs, RFC3339 UTC dates, at most 500 requests, and bounded text. Mark sent only after `Sender.Send` succeeds.

- [ ] **Step 4: Verify and commit**

Run:
```bash
gofmt -w internal/core/notifications/*.go
GOCACHE=/tmp/verstak-go-cache go test ./internal/core/notifications ./internal/core/... -count=1
git diff --check
```

Commit/push:
```bash
git add internal/core/notifications
git commit -m "feat: add persisted notification scheduler"
git push https://github.com/mirivlad/verstak.git main
git push mirror main
```

### Task 2: Permission-gated core/plugin notification API

**Files:**
- Modify: `main.go`
- Modify: `internal/api/app.go`
- Modify: `internal/api/app_test.go`
- Modify: `internal/core/permissions/registry.go`
- Modify: `frontend/src/lib/plugin-host/VerstakPluginAPI.js`
- Modify: `frontend/tests/plugin-api-contributions-test.mjs`

**Interfaces:**
- Core capability: `verstak/core/notifications/v1`.
- Permission: `notifications.schedule`, non-dangerous.
- Bound methods: `ReplacePluginNotifications(pluginID string, items []notifications.Request) string` and `ClearPluginNotifications(pluginID string) string`.
- Plugin methods: `api.notifications.replace(items)` and `api.notifications.clear()`.

- [ ] **Step 1: Add failing backend and host-API tests**

```go
func TestReplacePluginNotificationsRequiresCapabilityAndPermission(t *testing.T) {
    app, scheduler := newNotificationTestApp(t, manifestWithoutNotificationPermission)
    if got := app.ReplacePluginNotifications("example", []notifications.Request{{ID: "r", DueAt: "2026-07-14T10:00:00Z", Title: "R"}}); got == "" {
        t.Fatal("missing permission was accepted")
    }
    if scheduler.replaceCalls != 0 { t.Fatalf("replace calls = %d", scheduler.replaceCalls) }
}
```

The JS test mocks `App.ReplacePluginNotifications`, calls `api.notifications.replace`, and asserts both plugin ID forwarding and a namespaced rejection for a non-empty backend error.

- [ ] **Step 2: Confirm red**

```bash
GOCACHE=/tmp/verstak-go-cache go test ./internal/api -run TestReplacePluginNotificationsRequiresCapabilityAndPermission -count=1
node frontend/tests/plugin-api-contributions-test.mjs
```

Expected: missing API methods.

- [ ] **Step 3: Implement guarded bridge**

Register the capability in `main.go`, construct the scheduler, and guard bound calls with both existing access helpers before forwarding. Add `notifications.schedule` to the registry. Add the public host methods using `callBackendErrorString`; do not expose `window.runtime` to plugins.

- [ ] **Step 4: Verify and commit**

```bash
gofmt -w main.go internal/api/app.go internal/api/app_test.go internal/core/permissions/registry.go
GOCACHE=/tmp/verstak-go-cache go test ./internal/api -count=1
node frontend/tests/plugin-api-contributions-test.mjs
git diff --check
```

```bash
git add main.go internal/api/app.go internal/api/app_test.go internal/core/permissions/registry.go frontend/src/lib/plugin-host/VerstakPluginAPI.js frontend/tests/plugin-api-contributions-test.mjs
git commit -m "feat: expose scheduled notifications to plugins"
git push https://github.com/mirivlad/verstak.git main
git push mirror main
```

### Task 3: Todo reminder synchronization and Wails lifecycle

**Files:**
- Modify: `main.go`
- Modify: `internal/api/app.go`
- Modify: `internal/api/app_test.go`
- Modify: `../verstak-official-plugins/plugins/todo/plugin.json`
- Modify: `../verstak-official-plugins/plugins/todo/frontend/src/index.js`
- Modify: `../verstak-official-plugins/plugins/todo/locales/en.json`
- Modify: `../verstak-official-plugins/plugins/todo/locales/ru.json`
- Modify: `../verstak-official-plugins/scripts/smoke-todo-plugin.js`

**Interfaces:**
- `App.DomReady(ctx)` initializes Wails notifications before starting the scheduler.
- `App.Shutdown(ctx)` stops it before Wails cleanup.
- Todo replaces the complete reminder list only after Todo persistence succeeds and after Todo data loads.

- [ ] **Step 1: Add failing lifecycle and Todo smoke tests**

The App test uses function variables for Wails calls and proves initialize → start and stop → cleanup order. The Todo smoke mock records `notifications.replace`; it asserts one open reminder request with the Todo ID and a later empty replacement after completion or deletion.

- [ ] **Step 2: Confirm red**

```bash
GOCACHE=/tmp/verstak-go-cache go test ./internal/api -run 'TestDomReadyInitializesNotificationsBeforeScheduler|TestShutdownStopsNotifications' -count=1
cd ../verstak-official-plugins && node scripts/smoke-todo-plugin.js
```

Expected: lifecycle methods and schedule calls are absent.

- [ ] **Step 3: Implement lifecycle and Todo desired state**

Use `runtime.InitializeNotifications`, `runtime.SendNotification`, and `runtime.CleanupNotifications` only in core. Register `OnDomReady`/ `OnShutdown` in Wails options. Todo maps open valid `reminderAt` values to ISO UTC using `new Date(todo.reminderAt).toISOString()`, localizes title/body, and calls `api.notifications.replace`. A scheduling failure reports status but never rolls back already-saved Todo data.

- [ ] **Step 4: Verify and commit both repositories**

```bash
GOCACHE=/tmp/verstak-go-cache go test ./internal/api -count=1
cd ../verstak-official-plugins && node scripts/smoke-todo-plugin.js && ./scripts/check.sh
```

```bash
cd ../verstak-desktop
git add main.go internal/api/app.go internal/api/app_test.go
git commit -m "feat: start native notifications with desktop app"
git push https://github.com/mirivlad/verstak.git main
git push mirror main
cd ../verstak-official-plugins
git add plugins/todo scripts/smoke-todo-plugin.js
git commit -m "feat: schedule native Todo reminders"
git push https://github.com/mirivlad/verstak-official-plugins.git main
git push mirror main
```

### Task 4: Windows/Linux tray, close policy, and single instance

**Files:**
- Create: `internal/shell/tray/controller.go`
- Create: `internal/shell/tray/controller_test.go`
- Modify: `main.go`
- Modify: `internal/api/app.go`
- Modify: `internal/api/app_test.go`
- Modify: `go.mod`
- Modify: `go.sum`

**Interfaces:**
- `tray.Start(icon []byte, actions Actions) error`, with `Actions.Show` and `Actions.Quit`.
- `App.BeforeClose(ctx) bool` hides and returns true unless `App.Quit()` made shutdown explicit.
- `App.ShowWindow()` reveals the existing window.

- [ ] **Step 1: Add failing close-policy and tray action tests**

```go
func TestBeforeCloseHidesWindowUntilExplicitQuit(t *testing.T) {
    app, window := newWindowTestApp(t)
    if prevent := app.BeforeClose(context.Background()); !prevent || window.hideCalls != 1 { t.Fatal("ordinary close must hide") }
    app.Quit()
    if prevent := app.BeforeClose(context.Background()); prevent { t.Fatal("explicit quit was prevented") }
}
```

Use a fake tray adapter to assert exactly two labels, **Show Verstak** and **Quit**, and one callback invocation per click.

- [ ] **Step 2: Confirm red**

```bash
GOCACHE=/tmp/verstak-go-cache go test ./internal/api -run TestBeforeCloseHidesWindowUntilExplicitQuit -count=1
GOCACHE=/tmp/verstak-go-cache go test ./internal/shell/tray -count=1
```

Expected: absent packages/methods.

- [ ] **Step 3: Implement adapter and Wails wiring**

Use `fyne.io/systray`. The production adapter calls `RunWithExternalLoop` so the
Windows native message loop runs alongside Wails, embeds a multi-resolution
ICO on Windows and PNG on Linux, routes one left click to `app.ShowWindow`, and
leaves the platform-native right-click menu active. `main.go` registers
`OnBeforeClose` and uses `options.SingleInstanceLock` whose second launch calls
`app.ShowWindow`. Do not embed ignored Wails-generated files from `build/`.

- [ ] **Step 4: Verify and commit**

```bash
gofmt -w main.go internal/api/app.go internal/api/app_test.go internal/shell/tray/*.go
GOCACHE=/tmp/verstak-go-cache go test ./internal/shell/tray ./internal/api -count=1
./scripts/build.sh
! rg -n 'getlantern|appindicator' go.mod go.sum packaging scripts/build.sh
git diff --check
```

```bash
git add main.go internal/api/app.go internal/api/app_test.go internal/shell/tray go.mod go.sum
git commit -m "feat: keep desktop app in system tray"
git push https://github.com/mirivlad/verstak.git main
git push mirror main
```

### Task 5: Packaging and public README screenshots

**Files:**
- Modify: `packaging/deb/control`
- Modify: `scripts/build.sh`
- Modify: `scripts/package-appimage.sh`
- Modify: `scripts/test-package-formats.sh`
- Modify: `README.md`
- Create: `README.ru.md`
- Create: `docs/screenshots/overview.png`
- Create: `docs/screenshots/workspace-files-notes.png`
- Create: `docs/screenshots/activity-journal.png`

- [ ] **Step 1: Add failing package-contract checks**

Add assertions that the Linux build guidance, Debian metadata, and AppImage
packager no longer require the removed AppIndicator backend.

- [ ] **Step 2: Confirm red**

Run: `./scripts/test-package-formats.sh`

Expected: the first tray dependency assertion fails.

- [ ] **Step 3: Implement portable package support**

Keep Debian and AppImage focused on the Wails/WebKitGTK runtime. Do not add a
tray-specific AppIndicator dependency. Keep the Windows system-WebView2 policy
unchanged.

- [ ] **Step 4: Install the supplied public README sources**

Replace this repository's `README.md` with the supplied English source at the workspace root and add the supplied Russian source as `README.ru.md`. Add an image strip after “What is Verstak?” / “Что такое Верстак?” that links to three tracked PNG screenshots and uses matching localized alt text.

- [ ] **Step 5: Produce real screenshots**

Build and run the desktop app against `/home/mirivlad/Nextcloud/Verstak/VerstakVault` on the active X display. Capture:
1. Overview showing recent work and entry points;
2. a populated workspace Files/Notes view;
3. Activity with a suggested session and its Journal review.

Use `scrot` or ImageMagick `import`, crop only surrounding desktop chrome, inspect every PNG visually, and do not add generated mockups. Remove all non-user-safe text from the test vault before capture.

- [ ] **Step 6: Verify documentation and package contracts, commit, push**

```bash
./scripts/test-package-formats.sh
git diff --check
test -s docs/screenshots/overview.png
test -s docs/screenshots/workspace-files-notes.png
test -s docs/screenshots/activity-journal.png
```

```bash
git add packaging/deb/control scripts/build.sh scripts/package-appimage.sh scripts/test-package-formats.sh README.md README.ru.md docs/screenshots
git commit -m "docs: add public README and product screenshots"
git push https://github.com/mirivlad/verstak.git main
git push mirror main
```

### Task 6: Complete verification and GitHub alpha releases

**Files:**
- No source changes expected unless verification exposes a defect.

- [ ] **Step 1: Run all checks**

```bash
GOCACHE=/tmp/verstak-go-cache ./scripts/test.sh
./scripts/check.sh
./scripts/test-package-formats.sh
./scripts/test-build-windows.sh
cd ../verstak-official-plugins && ./scripts/check.sh && ./scripts/test-package-portable.sh && ./scripts/test-publish-github-release.sh
```

- [ ] **Step 2: Build and checksum both alpha releases**

```bash
cd ../verstak-official-plugins && ./scripts/release.sh v0.1.0-alpha.2 && (cd release && sha256sum --check SHA256SUMS)
cd ../verstak-desktop && ./scripts/release.sh v0.1.0-alpha.2 && (cd release && sha256sum --check SHA256SUMS)
```

- [ ] **Step 3: Publish and inspect releases**

```bash
cd ../verstak-official-plugins && ./scripts/publish-github-release.sh v0.1.0-alpha.2
cd ../verstak-desktop && ./scripts/publish-github-release.sh v0.1.0-alpha.2
gh release view v0.1.0-alpha.2 -R mirivlad/verstak-official-plugins
gh release view v0.1.0-alpha.2 -R mirivlad/verstak
```

The final report distinguishes automated verification from the two required real desktop manual checks: tray close/show/quit and a near-future Todo notification while the window is hidden.
