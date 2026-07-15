# Native Notifications and System Tray Design

**Status:** implemented; tray reliability update recorded on 2026-07-15

## Goal

Deliver Todo reminders as native notifications on Windows and Linux, and keep
Verstak running in the system tray when its main window is closed. The feature
must work in the portable Windows archive, Debian package, and AppImage.

## Scope

- The desktop core owns notification delivery, scheduling, and tray lifetime.
- `verstak.todo` owns the Todo-specific reminder policy and text.
- No new official notifications plugin is introduced. A dynamic plugin cannot
  call Wails directly, so it cannot be the native-notification transport.
- macOS is out of scope for this alpha. The interfaces remain platform-neutral
  where that costs nothing.

## Tray behavior

On Windows and Linux, a tray icon is initialized after Wails reaches
`OnDomReady`. Its menu contains exactly two actions:

1. **Show Verstak** — shows and focuses the existing main window.
2. **Quit** — exits the process deliberately.

One left click restores and focuses the existing main window. A native right
click opens the menu. Closing the main window with its window-manager close
control hides the window only after the tray has successfully initialized; it
then keeps the process, plugins, local browser receiver, and reminder scheduler
alive. If tray initialization fails or the native message loop exits, the
ordinary close path exits normally rather than leaving an unreachable process.
The quit action allows the close lifecycle to finish and exits normally.

The app has a single-instance lock. If a user launches the executable while an
instance is hidden in the tray, the existing instance shows its window instead
of creating a second process.

The implementation uses `fyne.io/systray` through a small
`internal/shell/tray` adapter. `RunWithExternalLoop` starts the Windows native
message loop without making Wails relinquish ownership of its GUI lifecycle.
The icon is a source-controlled multi-resolution ICO on Windows (16, 20, 24,
32, 48, and 256 pixels with transparency) and a PNG on Linux. Both are embedded
in the binary, so a clean build does not depend on ignored Wails-generated
files. Tray readiness is published only after icon, tooltip, and menu creation
all succeed; lifecycle diagnostics are logged for startup, readiness, clicks,
failure fallback, and shutdown.

## Notification capability and permission

The core registers the capability:

```
verstak/core/notifications/v1
```

Plugins that use it must both require that capability and declare the
`notifications.schedule` permission. The plugin-host API exposes only two
operations within the calling plugin namespace:

```
api.notifications.replace(items)
api.notifications.clear()
```

`replace` is an atomic desired-state replacement, not an append operation. An
item contains a plugin-local stable `id`, an ISO-8601 UTC `dueAt`, a title, and
a body. The core supplies the plugin ID and rejects calls from disabled,
missing-permission, or undeclared-capability plugins. It also validates empty
IDs, duplicate IDs, invalid timestamps, and unsafe oversized text.

No plugin can send arbitrary immediate native notifications or address another
plugin's schedules in this alpha.

## Scheduler and persistence

`internal/core/notifications` persists one canonical schedule file at:

```
<vault>/.verstak/notifications/schedules.json
```

Each record contains `{pluginId, id, dueAt, title, body, sentForDueAt}`.
The composite `(pluginId, id)` is unique. Replacing an item with the same due
time preserves `sentForDueAt`; changing `dueAt` clears it. Replacing a plugin's
list removes its stale records. This provides deterministic cancellation for
completed, deleted, and rescheduled Todos.

The manager starts after Wails reaches `OnDomReady`, initializes Wails native
notifications, and evaluates the persisted schedule immediately and then at
least every 30 seconds. A sender is injected behind an interface for unit
tests. After a successful delivery, the manager atomically records
`sentForDueAt`. A delivery error leaves the schedule pending and is logged for
a later retry.

An expired record that has not been sent is delivered once after the next app
start. A record already sent for its current due time is never sent again.
Completely quitting Verstak stops the scheduler: no separate daemon or OS
background service is added. Hiding the window in the tray does **not** stop it.

The core calls `CleanupNotifications` during shutdown, including on Linux where
it releases the D-Bus connection.

## Todo behavior

`verstak.todo` adds the core notifications capability to `requires` and adds
the `notifications.schedule` permission. After every successful Todo storage
write, it derives the complete desired reminder list:

- include only open Todos with a valid `reminderAt`;
- use the Todo ID as the stable notification ID;
- convert local `datetime-local` input to an ISO-8601 UTC instant;
- use the Todo title in the notification body and locale-aware reminder text;
- call `api.notifications.replace` with the full list.

The same replacement runs after loading persisted Todos, so a transient
schedule-write failure repairs itself next time the Todo view is opened. A
schedule API failure does not roll back Todo data; the UI reports the failure
instead. The existing in-view overdue/reminder badge remains useful context and
is not removed.

## Packaging

`fyne.io/systray` supplies the Windows message loop and uses the session D-Bus
on Linux. It does not require the removed AppIndicator development or runtime
package. The Windows release build still uses `x86_64-w64-mingw32-gcc` for the
Wails application itself.

The existing AppImage packager traverses `ldd` for the desktop executable and
copies non-glibc runtime libraries; it does not require a tray-specific shared
library.

## Public README and product screenshots

The workspace-root `README.md` and `README.ru.md` supplied by the maintainer
become the public repository documents: the English source replaces this
repository's `README.md`, and the Russian source is committed as
`README.ru.md`. Their existing language links and public-release instructions
are retained.

Three screenshots are captured from the real desktop application using the
test vault, then committed under `docs/screenshots/`:

1. `overview.png` — returning to recent work and useful next actions;
2. `workspace-files-notes.png` — ordinary vault files and Markdown notes in a
   workspace;
3. `activity-journal.png` — review of a factual activity session as a Journal
   entry.

Both README variants include the same three images with localized alt text.
They are factual UI captures, not generated illustrative mockups. Capture data
must be limited to the disposable test vault and inspected before commit so no
credentials or personal content are published.

## Test and manual verification

Automated tests cover:

- schedule replacement, cancellation, rescheduling, persistence, one-time
  overdue delivery, failed-send retry, and permission/capability rejection;
- Todo desired-list derivation and calls after create/edit/status/delete;
- close policy: ordinary close hides only after tray readiness, while explicit
  quit permits shutdown;
- tray controller action wiring, readiness/failure fallback, idempotent stop,
  left-click reveal, and second-instance window reveal;
- true multi-resolution Windows ICO data, Linux PNG data, and Linux/Windows
  build and package dependency expectations.

Manual smoke tests are required because neither unit tests nor Playwright can
assert a real desktop notification area or OS toast:

1. On Linux and Windows, start Verstak, verify the tray icon and tooltip, use
   one left click to reveal the window, use the right-click menu to reveal it,
   close the window, and use **Quit** to terminate it.
2. Set a Todo reminder for a near future time, hide the window in the tray, and
   observe one native notification.
3. Quit before a future reminder, relaunch after it expires, and observe one
   overdue notification with no duplicate on the next scheduler scan.
4. Inspect all three README screenshots at their committed size and confirm
   that they show only intended test-vault data and explain the feature named
   in their localized captions.
