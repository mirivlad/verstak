# Plugin Runtime

Описание системы плагинов Verstak Desktop: discovery, lifecycle, capabilities, states.

## Plugin Discovery

### Discovery Directories

Plugins ищутся в двух директориях (порядок приоритета):

| Путь | Назначение | Коммитится |
|---|---|---|
| `~/.config/verstak/plugins/` | User-installed plugins | Нет (user home) |
| `./plugins/` | Bundled / dev plugins | Нет (`.gitignore`) |

### ./plugins/ как Dev/Install Target

Директория `./plugins/` в корне `verstak-desktop` используется как:

- **Dev target** — `install-dev-plugins.sh` коприрует сюда собранные пакеты из `verstak-official-plugins/dist/`.
- **Bundled plugins** — при дистрибутиве core может поставлять плагины здесь.

Директория **не коммитится**. Каждый разработчик устанавливает плагины через `install-dev-plugins.sh`.

### Discovery Process

1. `DiscoverPlugins(dirs []string)` сканирует каждую директорию.
2. Для каждой поддиректории читает `plugin.json`.
3. `ValidateManifest()` проверяет: `schemaVersion == 1`, `id`, `name`, `version`, `apiVersion`, минимум 1 `provides`, минимум 1 `permissions`.
4. Дубликаты `id` отбрасываются с warning.

Подробнее о формате manifest см. [Plugin Manifest Format](#plugin-manifest-format).

## Plugin Lifecycle States

```
discovered
  │
  ├─ disabled — plugin.json найден, но plugin отключён
  │
  ├─ loading — plugin начинает загрузку
  │
  ├─ loaded — все required и optional capabilities разрешены
  │
  ├─ degraded — required capabilities разрешены, но не хватает optional
  │
  ├─ missing-required-capability — не хватает хотя бы одной required capability
  │
  ├─ failed — ошибка при регистрации capabilities (дубликат, panic)
  │
  ├─ incompatible — schemaVersion или apiVersion не поддерживаются
  │
  └─ (Скрыто) — discovered используется как промежуточный статус в discovery
```

### Определения статусов

| Статус | Условие | Поведение |
|---|---|---|
| `discovered` | plugin.json прочитан и валиден | Промежуточный, до capability resolution |
| `disabled` | Plugin отключён пользователем | Не загружается |
| `loaded` | Все capabilities разрешены | Полная функциональность |
| `degraded` | Required OK, но не хватает optional | Работает, часть функций недоступна |
| `missing-required-capability` | Не хватает required capability | Не загружается, показать ошибку |
| `failed` | Ошибка регистрации capabilities | Не загружается |
| `incompatible` | Неподдерживаемая schemaVersion/apiVersion | Не загружается |

## Required / Optional Capabilities

### Правило

- **`requires`** — жёсткая зависимость. Если ни один plugin не предоставляет требуемый capability, плагин получает `missing-required-capability` и **не загружается**.
- **`optionalRequires`** — мягкая зависимость. Если capability нет, плагин переходит в `degraded`, но **продолжает работать**.

### Регистрация core capabilities

Core capabilities регистрируются в `main.go` ДО plugin discovery:

```go
coreCaps := []string{
    "verstak/core/plugin-manager/v1",
    "verstak/core/capability-registry/v1",
    "verstak/core/contribution-registry/v1",
    "verstak/core/permissions/v1",
    "verstak/core/events/v1",
}
capRegistry.Register("verstak-desktop", coreCaps)
```

Это гарантирует, что любой plugin с `requires: ["verstak/core/plugin-manager/v1"]` будет загружен.

### Plugin capability resolution

```
foreach plugin:
    1. зарегистрировать plugin.provides в capability registry
    2. проверить plugin.requires — если есть missing → missing-required-capability
    3. проверить plugin.optionalRequires — если есть missing → degraded
    4. иначе → loaded
    5. зарегистрировать plugin.contributes в contribution registry
```

### Capability конфликт

Два plugin не могут предоставлять один и тот же capability. При попытке повторной регистрации — ошибка и статус `failed` для второго плагина.

## Plugin Manifest Format

Файл `plugin.json` в корне директории плагина.

### Обязательные поля

| Поле | Тип | Описание |
|---|---|---|
| `schemaVersion` | int | Должен быть `1` |
| `id` | string | Уникальный идентификатор (regex: `[a-zA-Z0-9.-]+`) |
| `name` | string | Человекочитаемое имя |
| `version` | string | Semver (напр. `"0.1.0"`) |
| `apiVersion` | string | API версии plugin |
| `provides` | string[] | Список capabilities (мин. 1) |
| `permissions` | string[] | Список запрашиваемых permissions (мин. 1) |

### Опциональные поля

| Поле | Тип | Описание |
|---|---|---|
| `description` | string | Описание плагина |
| `source` | string | `"official"`, `"local"`, `"third-party"` |
| `icon` | string | Иконка (emoji или имя) |
| `requires` | string[] | Жёзкие capability-зависимости |
| `optionalRequires` | string[] | Мягкие capability-зависимости |
| `frontend` | object | `{ "entry": "path/to/index.js", "style": "path/to/style.css" }` |
| `backend` | object | `{ "type": "go", "entry": { "linux-amd64": "...", ... }, "healthCheck": {...} }` |
| `migrations` | object | `{ "path": "migrations/" }` |
| `contributes` | object | UI contributions (см. ниже) |
| `sync` | object | `{ "namespaces": [...], "participate": bool }` |

### Пример

```json
{
  "schemaVersion": 1,
  "id": "verstak.platform-test",
  "name": "Platform Test",
  "version": "0.1.0",
  "apiVersion": "0.1.0",
  "provides": ["verstak/platform-test/v1"],
  "requires": ["verstak/core/plugin-manager/v1"],
  "optionalRequires": ["verstak/core/vault/v1", "verstak/core/sync/v1"],
  "permissions": ["vault.read", "events.publish", "ui.register"],
  "frontend": { "entry": "frontend/dist/index.js" },
  "contributes": {
    "views": [{ "id": "my.view", "title": "My View", "component": "MyPanel" }],
    "commands": [{ "id": "my.cmd", "title": "Run", "handler": "run" }]
  }
}
```

## Contribution Points

Плагины могут регистрировать UI contributions через поле `contributes`:

| Тип | Описание |
|---|---|
| `views` | Панели/страницы (component — Svelte) |
| `commands` | Команды command palette |
| `settingsPanels` | Панели в Settings |
| `sidebarItems` | Элементы боковой панели |
| `fileActions` | Действия над файлами |
| `noteActions` | Действия над заметками |
| `contextMenuEntries` | Пункты контекстного меню |
| `searchProviders` | Провайдеры поиска |
| `activityProviders` | Провайдеры активности |
| `statusBarItems` | Элементы status bar |

## Reload

`ReloadPlugins()` в `internal/api/app.go` позволяет перезагрузить plugins без перезапуска приложения:

1. Unregister all capabilities (кроме core).
2. Re-register core capabilities.
3. Re-scan discovery directories.
4. Re-run capability resolution.
5. Re-register contributions.

Frontend вызывает это при нажатии "Reload" в Plugin Manager.

## Файлы реализации

| Файл | Назначение |
|---|---|
| `internal/core/plugin/plugin.go` | Manifest, ValidateManifest, DiscoverPlugins, Status |
| `internal/core/capability/registry.go` | CapabilityRegistry |
| `internal/core/contribution/registry.go` | ContributionRegistry |
| `internal/core/permissions/registry.go` | PermissionsRegistry |
| `internal/core/events/bus.go` | EventBus |
| `internal/api/app.go` | Wails API, ReloadPlugins |
| `main.go` | Инициализация, lifecycle orchestration |
