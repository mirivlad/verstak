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
