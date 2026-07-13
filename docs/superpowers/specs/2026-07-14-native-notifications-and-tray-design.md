# Native Notifications and System Tray Design

**Status:** approved for implementation on 2026-07-14

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

On Windows and Linux, a tray icon is registered before the Wails event loop
starts. Its menu contains exactly two actions:

1. **Show Verstak** — shows and focuses the existing main window.
2. **Quit** — exits the process deliberately.

Closing the main window with its window-manager close control hides the window
and keeps the process, plugins, local browser receiver, and reminder scheduler
alive. It does not terminate the application. The quit action temporarily
allows the close lifecycle to finish and then exits normally.

The app has a single-instance lock. If a user launches the executable while an
instance is hidden in the tray, the existing instance shows its window instead
of creating a second process.

The implementation uses `github.com/getlantern/systray` through a small
`internal/shell/tray` adapter. It uses `Register`, rather than its blocking
`Run`, so Wails remains the owner of the GUI event loop. A compact PNG derived
from the tracked project logo is encoded in the tray package, so a clean build
does not depend on ignored Wails-generated icon files. The library accepts PNG
icon bytes on both target platforms.

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

`getlantern/systray` requires CGO. The Windows release build already uses
`x86_64-w64-mingw32-gcc`; the Windows packaging tests must compile it with the
tray dependency included. Linux build instructions add
`libayatana-appindicator3-dev`. The Debian package declares the corresponding
runtime dependency `libayatana-appindicator3-1`.

The existing AppImage packager traverses `ldd` for the desktop executable and
copies non-glibc runtime libraries. Its verification is extended to prove that
the appindicator library is present in the AppDir when the tray implementation
is compiled in.

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
- close policy: ordinary close hides, explicit quit permits shutdown;
- tray controller action wiring and second-instance window reveal;
- Linux/Windows build scripts and package dependency expectations.

Manual smoke tests are required because neither unit tests nor Playwright can
assert a real desktop notification area or OS toast:

1. On Linux and Windows, start Verstak, close its window, use the tray menu to
   reveal it, and use **Quit** to terminate it.
2. Set a Todo reminder for a near future time, hide the window in the tray, and
   observe one native notification.
3. Quit before a future reminder, relaunch after it expires, and observe one
   overdue notification with no duplicate on the next scheduler scan.
4. Inspect all three README screenshots at their committed size and confirm
   that they show only intended test-vault data and explain the feature named
   in their localized captions.
