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

// Vault capability — регистрируется отдельно после vault initialization
capRegistry.Register("verstak-desktop", []string{"verstak/core/vault/v1"})
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

Плагины могут регистрировать UI contributions через поле `contributes` в `plugin.json`.

### Реализованные contribution points (Milestone 5a)

| Тип | Поле manifest | Описание | Frontend host |
|---|---|---|---|
| Боковая панель | `sidebarItems` | Элементы в sidebar слева | ✅ Sidebar.svelte (из ContributionRegistry) |
| Основные панели | `views` | Полноценные страницы/панели | ✅ ViewContainer.svelte (PluginBundleHost — real frontend bundle) |
| Панели настроек | `settingsPanels` | Панели в Plugin Manager | ✅ PluginManager.svelte (кнопка Settings, открывает modal) |
| Команды | `commands` | Команды для command palette | ✅ ContributionRegistry (UI command palette не реализован) |

### Планируемые contribution points

| Тип | Поле manifest | Статус |
|---|---|---|
| Действия над файлами | `fileActions` | Registry готов, UI не реализован |
| Действия над заметками | `noteActions` | Registry готов, UI не реализован |
| Контекстное меню | `contextMenuEntries` | Registry готов, UI не реализован |
| Провайдеры поиска | `searchProviders` | Registry готов, UI не реализован |
| Провайдеры активности | `activityProviders` | Registry готов, UI не реализован |
| Элементы status bar | `statusBarItems` | Registry готов, UI не реализован |

### Структура contribution points в manifest

```json
{
  "contributes": {
    "sidebarItems": [
      {
        "id": "mypanel.sidebar",
        "title": "My Panel",
        "icon": "📌",
        "view": "mypanel.view",
        "position": 100
      }
    ],
    "views": [
      {
        "id": "mypanel.view",
        "title": "My Panel View",
        "icon": "📌",
        "component": "MyPanelComponent"
      }
    ],
    "settingsPanels": [
      {
        "id": "mypanel.settings",
        "title": "My Settings",
        "component": "MySettingsPanel"
      }
    ],
    "commands": [
      {
        "id": "mypanel.cmd",
        "title": "Do Something",
        "icon": "⚡",
        "handler": "doSomething"
      }
    ]
  }
}
```

### Contribution lifecycle

1. Plugin `Register(pluginID, contributions)` — все contributions регистрируются
2. `Unregister(pluginID)` — удаляет все contributions указанного plugin
3. Reload: `Unregister → Register` (предотвращает дублирование)
4. Disable plugin → `Unregister` (contributions исчезают из UI)
5. Enable plugin → `Register` при следующем Reload
6. Registry idempotent: Register удаляет старые записи перед добавлением новых

### Error boundary

- Ошибка в plugin view/settings placeholder не роняет shell
- ViewContainer показывает "⚠️ Plugin UI failed" fallback
- Error канал: `console.error` + видимый fallback в UI

## Frontend Bundle Contract

### Регистрация компонентов

Плагин регистрирует frontend компоненты через глобальную функцию `window.VerstakPluginRegister`:

```javascript
window.VerstakPluginRegister('plugin.id', {
  components: {
    'ComponentName': {
      mount: function(containerEl, props, api) {
        // containerEl — div, созданный PluginBundleHost
        // api — ограниченный VerstakPluginAPI
        containerEl.innerHTML = '<h1>Hello from plugin!</h1>';
      },
      unmount: function(containerEl) {
        // Очистка при смене view
        containerEl.innerHTML = '';
      }
    }
  }
});
```

### VerstakPluginAPI

API объект передаётся в `mount()` и содержит только ограниченный набор методов:

| Свойство | Статус | Описание |
|---|---|---|
| `api.pluginId` | ✅ Работает | ID плагина |
| `api.capabilities.has(id)` | 🔧 Stub | Запрос capability registry (planned) |
| `api.events.publish(type, payload)` | 🔧 Stub | Публикация события (planned) |
| `api.events.subscribe(type, handler)` | 🔧 Stub | Подписка на события (planned) |
| `api.settings.read(key)` | 🔧 Stub | Чтение настроек плагина (planned) |
| `api.settings.write(key, value)` | 🔧 Stub | Запись настроек плагина (planned) |
| `api.commands.execute(id, args)` | 🔧 Stub | Выполнение команды (planned) |

### Загрузка бандла

1. `PluginBundleHost` получает pluginId и componentId
2. Вызывает `App.GetPluginFrontendInfo(pluginId)` — получает entry/style/rootPath
3. Вызывает `App.GetPluginAssetContent(pluginId, entry)` — получает JS контент
4. Выполняет контент через `new Function(content)` — bundle вызывает `VerstakPluginRegister`
5. Находит компонент по componentId и вызывает `mount(container, props, api)`
6. При смене view — вызывает `unmount(container)` для старого компонента

### Безопасность asset path

| Правило | Проверка |
|---|---|
| Нет абсолютных путей | Пути, начинающиеся с `/` или `\`, отклоняются |
| Нет path traversal | Пути, содержащие `..`, отклоняются |
| Нет выхода за root | После `filepath.Join` проверяется, что путь внутри plugin root |
| Только существующие файлы | `os.ReadFile` возвращает ошибку если файл не существует |

### manifest frontend config

```json
{
  "frontend": {
    "entry": "frontend/dist/index.js",
    "style": "frontend/style.css"
  }
}
```

## Reload

`ReloadPlugins()` в `internal/api/app.go` позволяет перезагрузить plugins без перезапуска приложения:

1. Unregister all non-core capabilities.
2. Re-register core capabilities + vault + workspace (если открыт).
3. Re-scan discovery directories.
4. For each plugin: re-run capability resolution.
5. **Unregister contributions** before re-registering (предотвращает дубли).
6. Register contributions for loaded/degraded plugins (disabled/failed — не регистрируются).
7. Update plugins list.

Frontend вызывает это при нажатии "Reload" в Plugin Manager.

## Vault Core Capability

- `verstak/core/vault/v1` — регистрируется в `main.go` после остальных core capabilities, когда vault инициализирован.
- Vault layout: `<base>/VerstakVault/.verstak/` с подпапками (см. ниже).
- Plugin namespace paths: `plugin-data/<id>`, `plugin-settings/<id>`, `plugin-cache/<id>`.
- Vault events: `vault.created`, `vault.opened`, `vault.closed`, `vault.error`.
- Vault status: `not-created`, `closed`, `open`, `error`.
- Path traversal protection через `ResolveSafePath`.

### Vault Directory Layout

```
<base>/
  VerstakVault/           ← vault root (создаётся CreateVault)
    .verstak/
      vault.json          ← VaultMeta (schemaVersion, vaultId, createdAt, app)
      plugin-data/        ← per-plugin data namespaces
        <plugin-id>/
      plugin-settings/    ← per-plugin settings namespaces
        <plugin-id>/
      plugin-cache/       ← per-plugin cache namespaces
        <plugin-id>/
      trash/              ← soft-deleted items
      logs/               ← vault-scoped logs
```

### Vault API

| Метод | Описание |
|---|---|
| `CreateVault(path)` | Создаёт `VerstakVault/` с `.verstak/` layout и `vault.json`. Публикует `vault.created`. |
| `OpenVault(path)` | Открывает существующий vault, валидирует `vault.json`. Публикует `vault.opened`. |
| `CloseVault()` | Закрывает vault, сбрасывает path/meta. Публикует `vault.closed`. |
| `GetVaultStatus()` | Возвращает текущий статус: `not-created`, `closed`, `open`, `error`. |
| `GetVaultPath()` | Возвращает путь к vault root. |
| `GetVaultMeta()` | Возвращает `VaultMeta` (vaultId, schemaVersion, timestamps). |
| `ResolveSafePath(rel)` | Безопасно резолвит относительный путь внутри vault. Блокирует path traversal. |
| `GetPluginDataPath(id)` | Возвращает (и создаёт) `plugin-data/<id>/`. |
| `GetPluginSettingsPath(id)` | Возвращает (и создаёт) `plugin-settings/<id>/`. |
| `GetPluginCachePath(id)` | Возвращает (и создаёт) `plugin-cache/<id>/`. |

### Vault Events

| Event | Когда публикуется | Payload |
|---|---|---|
| `vault.created` | После успешного `CreateVault` | `path`, `vaultId` |
| `vault.opened` | После успешного `OpenVault` | `path`, `vaultId` |
| `vault.closed` | После `CloseVault` | `vaultId` |
| `vault.error` | При ошибках операций | `error` |

### Vault Status Flow

```
not-created ──CreateVault──▶ open ──CloseVault──▶ closed
                                │                    │
                                └──OpenVault─────────┘
```

## Файлы реализации

| Файл | Назначение |
|---|---|
| `internal/core/plugin/plugin.go` | Manifest, ValidateManifest, DiscoverPlugins, Status |
| `internal/core/capability/registry.go` | CapabilityRegistry |
| `internal/core/contribution/registry.go` | ContributionRegistry |
| `internal/core/permissions/registry.go` | PermissionsRegistry |
| `internal/core/events/bus.go` | EventBus |
| `internal/api/app.go` | Wails API, ReloadPlugins |
| `internal/core/vault/vault.go` | Vault service: CreateVault, OpenVault, CloseVault, ResolveSafePath, plugin namespace paths |
|| `internal/core/vault/vault_test.go` | Vault tests: layout creation, open/close, path traversal, events |
|| `internal/core/storage/api.go` | Plugin storage API: settings/data/cache JSON with namespace isolation |
|| `internal/core/storage/api_test.go` | Storage tests: write/read, path traversal, atomic write |
|| `internal/core/appsettings/manager.go` | App settings manager: Load/Save/Update, recent vaults, defaults |
|| `internal/core/appsettings/manager_test.go` | App settings tests: defaults, corrupt config, recent dedup |
|| `internal/core/pluginstate/manager.go` | Vault plugin state: enable/disable, desired plugins, missing-installed |
|| `internal/core/pluginstate/manager_test.go` | Plugin state tests: enable/disable, persist, corrupt, missing |

---

## App Settings

App settings хранятся **локально** (НЕ внутри vault) в `~/.config/verstak/config.json`.

### Поле | Назначение
---|---
`currentVaultPath` | Путь к текущему vault
`recentVaults` | Список недавних vault (max 10, без дублей)
`theme` | Тема (dark/light)
`devMode` | Режим разработки
`userPluginsDir` | Директория пользовательских плагинов
`windowState` | Состояние окна (размеры, максимизация)
`lastOpenedAt` | Время последнего запуска

### Правила
- Если config отсутствует — создаётся с defaults
- Если config битый — backup + создание defaults с понятной ошибкой
- `currentVaultPath` при запуске проверяется и vault открывается автоматически
- Secrets НЕ хранятся в app settings

---

## Vault Plugin State

Vault plugin state хранится **внутри vault** в `.verstak/plugins.json`.

### Структура

```json
{
  "schemaVersion": 1,
  "enabledPlugins": ["verstak.platform-test"],
  "disabledPlugins": [],
  "desiredPlugins": [
    {
      "id": "verstak.platform-test",
      "version": "0.1.0",
      "source": "official"
    }
  ],
  "updatedAt": "2026-06-17T..."
}
```

### Поле | Назначение
---|---
`enabledPlugins` | Плагины, которые активны в этом vault
`disabledPlugins` | Плагины, которые явно отключены
`desiredPlugins` | Плагины, которые нужны этому vault (для будущей синхронизации)
`updatedAt` | Время последнего обновления

### Правила
- Enabled/disabled состояние относится к конкретному vault
- Disabled plugin не регистрирует provides/contributions
- Plugin settings остаются в `.verstak/plugin-settings/<id>/settings.json`
- Отсутствие `plugins.json` → создаётся с defaults
- Битый `plugins.json` → backup + defaults с понятной ошибкой
- App settings НЕ хранятся внутри vault
- Plugin packages НЕ хранятся в vault settings

### Installed vs Enabled

- **Installed** — plugin package существует в discovery directory
- **Enabled** — plugin активен в vault plugin state
- **Disabled** — plugin установлен, но отключен в vault
- **Missing installed** — plugin listed в `desiredPlugins`, но package отсутствует локально

### Missing Installed Plugins

Состояние для будущей синхронизации:
- `desiredPlugins` может содержать plugin, которого нет локально
- Plugin Manager показывает "Missing installed plugin"
- Auto-install пока НЕ делается
- Показывается подсказка: "Install official plugin package"


## UI Layout

```
┌─────────────────────────────────────────────────────┐
│ App.svelte                                          │
│ ┌──────────┬──────────────────────────────────────┐ │
│ │ Sidebar  │ Content area                         │ │
│ │          │                                       │ │
│ │ Verstak  │ PluginManager | ViewContainer         │ │
│ │          │                                       │ │
│ │ 🧩 Plugin│ (padding: 1.5rem)                    │ │
│ │   Manager│                                       │ │
│ │          │                                       │ │
│ │ Plugins  │                                       │ │
│ │ 📌 item1 │                                       │ │
│ │ 📌 item2 │                                       │ │
│ │          │                                       │ │
│ │ ● Vault  │                                       │ │
│ └──────────┴──────────────────────────────────────┘ │
└─────────────────────────────────────────────────────┘
```

- **Sidebar** (220px): навигация (Plugin Manager), plugin sidebar items, vault status
- **Content**: Plugin Manager или ViewContainer в зависимости от выбранного view
- **Vault Selection**: полноэкранный экран, показывается когда vault не открыт

## Milestone 4b — UI Completion (2026-06-17)

Сделано:
- VaultSelection.svelte: исправлен flow (CreateVault → OpenVault → SetCurrentVault)
- Sidebar.svelte: полная навигация с отступами, plugin sidebar items, vault status
- App.svelte: обработка `verstak:nav` событий, global reset стилей
- PluginManager.svelte: исправлены отступы header

Проверки:
- `go test ./...` — 52 PASS
- `./scripts/check.sh` — ✅
- `./scripts/smoke-platform.sh` — ✅ (enable/disable/plugins.json)
- `./scripts/build.sh` — ✅

## Workspace / Cases Core Capability

Workspace — центральная модель Верстака вокруг "дел". Это НЕ notes/files — это фундамент.

### Ноды

| Поле | Тип | Описание |
|------|-----|----------|
| `id` | UUID | Стабильный идентификатор |
| `parentId` | string | ID родителя (пусто для root) |
| `type` | space/case/folder | Тип ноды |
| `title` | string | Название |
| `status` | active/sleeping/archived | Жизненный цикл |
| `tags` | string[] | Теги |
| `order` | int | Порядок среди siblings |
| `createdAt` | RFC3339Nano | Создан |
| `updatedAt` | RFC3339Nano | Обновлён |

### Хранение

`<vault>/.verstak/workspace.json` — атомарная запись (temp + rename).

### API

- `GetWorkspaceTree()` — полное дерево
- `CreateWorkspaceNode(parentID, type, title)` — создать
- `RenameWorkspaceNode(id, title)` — переименовать
- `MoveWorkspaceNode(id, newParentID)` — переместить
- `ArchiveWorkspaceNode(id)` — архивировать
- `SetCurrentWorkspaceNode(id)` — выбрать текущую
- `GetCurrentWorkspaceNode()` — получить текущую

### Capability

`verstak/core/workspace/v1` — регистрируется только когда vault открыт и workspace инициализирован.

### Правила

- Root node создаётся при создании vault
- Порядок children стабилен (sort by order)
- Нельзя переместить ноду в себя или в своего потомка
- Архивирование — soft delete (status = archived)
- Corrupt JSON → backup + defaults

### Типы нод

| Тип | Назначение |
|-----|-----------|
| `space` | Рабочее пространство (root) |
| `case` | Дело |
| `folder` | Папка |

НЕ добавляются: note, file, action, secret, worklog, link — это плагины.

### Lifecycle Events

**Planned (not yet implemented in runtime):**
- `workspace.node.created`
- `workspace.node.renamed`
- `workspace.node.moved`
- `workspace.node.archived`
- `workspace.node.selected`
- `workspace.error`

### UI

WorkspaceTree в sidebar:
- Дерево с expand/collapse
- Создание case/folder
- Выбор текущей ноды
- Индикатор статуса (active/archived/sleeping)

---

## Build Scripts

В `verstak-desktop/scripts/` есть два скрипта:

### `build.sh` — локальная детерминированная сборка

- Собирает **только** `verstak-desktop` (core platform).
- Не трогает другие репозитории.
- **Fail-fast**: любая ошибка (go vet, go test, frontend build, wails build) прерывает сборку.
- Проверяет: deps → frontend build → go mod download → go vet → go build → go test → wails build + plugin copy.
- Используется в CI и для повседневной работы над core.

### `update-and-build-all.sh` — dev helper для полной пересборки связки

- **Не для CI.** Только для разработки, когда нужно быстро собрать всё вместе.
- Шаги:
  1. `git pull --ff-only` во всех 6 репозиториях
  2. Сборка official plugins (frontend npm build + backend go build для каждого плагина)
  3. Копирование собранных плагинов в `verstak-desktop/plugins/`
  4. Запуск `build.sh` для сборки desktop
- Ошибки pull и сборки плагинов не прерывают скрипт (best-effort), но ошибка build.sh прерывает (fail-fast).
