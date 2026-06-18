# AGENTS.md — Verstak Desktop

## Назначение

`verstak-desktop` — Core Platform + UI Shell. Это минимальное ядро, которое запускает приложение, управляет плагинами и предоставляет общий интерфейс. **Core не содержит бизнес-функций пользователя.**

## Главные инварианты

- Core не импортирует official plugins как обязательные модули.
- Core не содержит notes/files/editor/activity/journal как внутренние фичи.
- Все пользовательские функции приходят через динамические плагины.
- Plugin Manager UI — обязательный компонент core с первого этапа.
- Capability registry и contribution registry — механизмы связи, а не жёсткие импорты.
- Плагины не могут обращаться к Wails backend методам напрямую — только через VerstakPluginAPI.

## Технологии

- **Backend:** Go (Wails v2)
- **Frontend:** Svelte (plain JS, без `lang="ts"`)
- **UI Shell:** окно, навигация, command palette, settings, plugin manager, dialogs/toasts

## Что НЕ входит в core

- Markdown editor (это плагин)
- File manager (это плагин)
- Notes workflow (это плагин)
- Activity / journal (это плагины)
- Browser inbox (это плагин)
- Search (это плагин)
- Secrets (это плагин)
- Templates (это плагин)

## Plugin Runtime

1. Discovery — сканирование plugin directories, чтение plugin.json
2. Validation — проверка schemaVersion, apiVersion, обязательных полей
3. State check — enabled/disabled
4. Capability resolution — проверка requires/optionalRequires
5. Permissions — запрос, отображение пользователю
6. Backend sidecar — launch, если нужен
7. Frontend bundle — загрузка, если есть
8. Registration — capabilities и contributions в registry
9. Status — loaded / degraded / failed / incompatible / missing-required-capability

## Структура репозитория

```
verstak-desktop/
  AGENTS.md
  go.mod
  main.go
  cmd/
  internal/
    core/
      plugin/
        discovery.go
        manifest.go
        state.go
        lifecycle.go
      capability/
        registry.go
      contribution/
        registry.go
      permissions/
        registry.go
      events/
        bus.go
      settings/
        registry.go
      vault/
        api.go
      storage/
        api.go
      diagnostics/
        api.go
      sync/
        boundary.go
    shell/
      app/
      navigation/
      window/
      command-palette/
      plugin-manager/
        ui/
      settings/
      dialogs/
    api/
      plugin.go
  frontend/
    src/
    wails.json
  ...
```

## Verification language policy

**Do not say "checked", "verified", "works", or "all good" unless a concrete
verification command or user scenario was executed.**

Use exact labels:
- Build checked
- Unit tests checked
- Backend behavior checked
- Frontend E2E checked
- Real desktop GUI checked
- Not checked / requires manual verification

If GUI behavior was not clicked and asserted, report:
"GUI behavior was not verified."

### GUI testing

Frontend E2E tests are in `frontend/e2e/`. Run with `npm run test:e2e`.
These tests use Playwright + mocked Wails bindings. They test Svelte component
logic and user interactions in a real Chromium browser.

**Limitations:** Playwright tests do NOT test the real WebKitGTK/Wails native
shell. For real desktop GUI verification, a separate AT-SPI/xdotool layer is
needed (not yet implemented).

See `docs/GUI_TESTING.md` for details.

## Debug logging

**Always use debug logging when investigating issues. Never rely on "it should work" — look at the logs.**

### Backend debug

Enable with `--debug` flag:
```bash
./verstak-desktop --debug
```

Logs go to: `~/.local/share/verstak/debug/verstak-YYYY-MM-DD-HHMMSS.log`

View in real-time:
```bash
tail -f ~/.local/share/verstak/debug/verstak-*.log
```

What's logged (when `--debug` is active):
- Plugin discovery: dirs scanned, each plugin found (id, name, version, source, root)
- Plugin lifecycle: capability registration, contribution registration, status transitions
- API calls: GetPlugins, GetContributions, GetCapabilities, ReloadPlugins, EnablePlugin, DisablePlugin
- Vault operations: open/close/status

### Frontend debug

Enable via **either**:
1. URL query param: `?debug` (not practical in Wails, use #2)
2. In browser console: `localStorage.setItem('verstak-debug', 'true')` then reload

Logs go to:
- Browser console (with `[debug]` prefix)
- localStorage buffer: `localStorage.getItem('verstak-debug-log')` (last 1000 entries)

Export frontend log from console:
```javascript
copy(JSON.parse(localStorage.getItem('verstak-debug-log')))
// or
window.__verstakDebug.exportLog()
```

What's logged:
- App startup: checkVault, GetAppSettings, GetVaultStatus
- Navigation: onNav, onOpenView, onOpenSettings, onCloseSettings
- PluginManager: loadAll (start/end, plugin count, each plugin status), reload, enable/disable
- Sidebar: onMount (plugins loaded, sidebar items count), handleNav, handleSidebarItem

### Debug workflow

1. User reports issue
2. Restart app with `--debug` and reproduce
3. Run `tail -f ~/.local/share/verstak/debug/verstak-*.log` and share output
4. For frontend issues: enable frontend debug view `exportLog()` output
5. Analyze logs, identify root cause, fix
6. Verify fix by asking user to reproduce again with debug on

**Never skip step 2-4. Always look at real logs before proposing fixes.**
