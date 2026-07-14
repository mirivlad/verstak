# Plugin Runtime

Описание системы плагинов Verstak Desktop: discovery, lifecycle, capabilities, states.

## Plugin Discovery

### Discovery Directories

Plugins ищутся через единый resolver `internal/core/plugin.ResolveDiscoveryDirs`.
Порядок приоритета:

| Путь | Назначение | Коммитится |
|---|---|---|
| `VERSTAK_PLUGIN_DIR` | Override для тестов/dev; можно передать несколько путей через OS path separator | Нет |
| `./plugins/` | Dev plugins относительно текущей рабочей директории/repo | Нет (`.gitignore`) |
| `<binary-dir>/plugins/` | Packaged plugins рядом с desktop binary | Зависит от дистрибутива |
| `~/.config/verstak/plugins/` | User-installed plugins | Нет (user home) |

Resolver нормализует пути, удаляет дубликаты и передает discovery только канонический
список директорий. Отсутствующие директории просто пропускаются на этапе scanning.

Discovery сканирует **все** resolved директории в указанном порядке. Если один и тот
же `plugin.id` найден несколько раз, применяется правило **first plugin wins**:
первый найденный plugin загружается, последующие plugins с тем же id пропускаются.
Конфликт логируется и возвращается как discovery warning с двумя путями: путь
пропущенного duplicate и путь уже загруженного winner.

### ./plugins/ как Dev/Install Target

Директория `./plugins/` от текущей рабочей директории используется как:

- **Dev target** — `install-dev-plugins.sh` коприрует сюда собранные пакеты из `verstak-official-plugins/dist/`.
- **Local override** — при запуске desktop из repo позволяет быстро проверять packaged bundles.

В packaged-сборке bundled plugins должны лежать в `plugins/` рядом с executable.
Для тестов и локальных сценариев можно задать `VERSTAK_PLUGIN_DIR=/path/to/plugins`.

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
    "verstak/core/files/v1",
    "verstak/core/workbench/v1",
}
capRegistry.Register("verstak-desktop", coreCaps)

// Vault capability — регистрируется отдельно после vault initialization
capRegistry.Register("verstak-desktop", []string{"verstak/core/vault/v1"})
```

Это гарантирует, что любой plugin с `requires: ["verstak/core/plugin-manager/v1"]` будет загружен.

### Plugin capability resolution

```
1. Исключить disabled plugins.
2. Пока есть неразрешённые plugins:
   - plugin с уже доступными `requires` атомарно регистрирует `provides`;
   - при конфликте `provides` plugin получает `failed`;
   - если за проход ничего не разрешилось, остальные получают
     `missing-required-capability`.
3. После разрешения всех обязательных зависимостей проверить
   `optionalRequires`: отсутствующие переводят plugin в `degraded`.
4. Зарегистрировать `contributes` только для `loaded` и `degraded` plugins.
```

Таким образом, порядок discovery не влияет на обычные зависимости между
plugins. При конкурирующих providers одного capability порядок discovery служит
tie-breaker. `provides` плагина с неразрешённым `requires` или ошибкой
регистрации не попадают в registry.

### Capability конфликт

Два plugin не могут предоставлять один и тот же capability. При попытке повторной регистрации второй plugin получает `failed`; регистрация атомарна, поэтому ни один capability такого plugin не остаётся в registry.

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
| `icon` | string | Имя иконки из встроенного Lucide-набора shell |
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
  "permissions": ["vault.read", "events.publish", "ui.register", "workbench.open"],
  "frontend": { "entry": "frontend/dist/index.js" },
  "contributes": {
    "views": [{ "id": "my.view", "title": "My View", "component": "MyPanel" }],
    "commands": [{ "id": "my.cmd", "title": "Run", "handler": "run" }]
  }
}
```

## Contribution Points

Плагины могут регистрировать UI contributions через поле `contributes` в `plugin.json`.

Icon fields use shell icon names rendered through the bundled Lucide SVG wrapper. Plugins must not rely on emoji, Unicode pictographs, or system icon fonts; if a plugin needs its own icon font, it must bundle the font and reference it from its own frontend bundle.

### Реализованные contribution points (Milestone 5a)

| Тип | Поле manifest | Описание | Frontend host |
|---|---|---|---|
| Боковая панель | `sidebarItems` | Элементы в sidebar слева | ✅ Sidebar.svelte (из ContributionRegistry) |
| Основные панели | `views` | Полноценные страницы/панели | ✅ ViewContainer.svelte (PluginBundleHost — real frontend bundle) |
| Панели настроек | `settingsPanels` | Панели в Plugin Manager | ✅ PluginManager.svelte (кнопка Settings, открывает modal) |
| Команды | `commands` | Команды для command palette | ✅ ContributionRegistry + CommandPalette UI |
| Open/edit providers | `openProviders` | Провайдеры viewer/editor для Workbench routing | ✅ ContributionRegistry + минимальный Workbench host |
| Действия над файлами | `fileActions` | Provider actions for Files surface | ✅ Files plugin context menu |
| Действия над заметками | `noteActions` | Provider actions for Notes surface | ✅ Notes plugin row actions |
| Контекстное меню | `contextMenuEntries` | Provider context menu entries | ✅ Files plugin context menu |
| Провайдеры поиска | `searchProviders` | Search provider discovery | ✅ Contribution summary + Search plugin |
| Провайдеры активности | `activityProviders` | Activity event subscriptions | ✅ Backend activity recorder |
| Элементы status bar | `statusBarItems` | Status bar labels/actions | ✅ StatusBar.svelte host |

### Планируемые contribution points

| Тип | Поле manifest | Статус |
|---|---|---|
| Sidecar-backed command handlers | `commands` | Planned; bundled frontend handlers are current runtime |
| Import/export providers | `importers` / `exporters` | Planned |
| Protocol receivers | `protocol.receivers` | Planned |

### Структура contribution points в manifest

```json
{
  "contributes": {
    "sidebarItems": [
      {
        "id": "mypanel.sidebar",
        "title": "My Panel",
        "icon": "puzzle",
        "view": "mypanel.view",
        "position": 100
      }
    ],
    "views": [
      {
        "id": "mypanel.view",
        "title": "My Panel View",
        "icon": "puzzle",
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
        "icon": "gear",
        "handler": "doSomething"
      }
    ]
  }
}
```

### Contribution lifecycle

1. Plugin `Register(pluginID, contributions)` — все contributions регистрируются
2. `Unregister(pluginID)` — удаляет все contributions указанного plugin
3. Reload сначала удаляет contributions всех plugins предыдущего discovery, затем регистрирует только `loaded`/`degraded` plugins
4. Disable plugin → `Unregister` (contributions исчезают из UI)
5. Enable plugin → `Register` при следующем Reload
6. Registry idempotent: Register удаляет старые записи перед добавлением новых

## Bundled Frontend Plugin API

Bundled frontend plugins получают API от host через `createPluginAPI(pluginId)`.
Обычный plugin code не передает `pluginId` в методы API: scope закрепляется в
host при mount компонента. Это защищает нормальный cooperative path от случайного
доступа к чужому namespace.

Текущая модель безопасности честно ограничена:

- bundled frontend plugins исполняются в общем JS-контексте приложения;
- проверки permissions/capabilities сейчас являются contract/policy checks, а не
  полноценной security boundary;
- malicious JS в общем контексте теоретически может обойти frontend wrapper;
- настоящая изоляция будет только после отдельного sidecar/sandbox milestone.

## Workbench Open/Edit Routing

Files and Notes plugins do not import or embed a concrete editor plugin. They
call `api.workbench.openResource(request)` or `api.workbench.editResource(request)`.
The backend requires the source plugin to be enabled, loaded/degraded, and to
declare `workbench.open`. This is a policy/contract check, not a security
boundary.

`OpenResourceRequest`:

```ts
type OpenResourceRequest = {
  kind: "vault-file";
  path: string;
  mode?: "view" | "edit";
  mime?: string;
  extension?: string;
  context?: {
    sourcePluginId?: string;
    sourceView?: "files" | "notes" | string;
    isInsideNotesFolder?: boolean;
    notesScopePath?: string;
    notesMode?: boolean;
  };
};
```

Routing contexts are fixed as `generic-text`, `generic-markdown`, and
`notes-markdown`. `.md`/`.markdown` inside canonical `Notes/` folders uses
`notes-markdown`; markdown outside Notes uses `generic-markdown`; ordinary text
uses `generic-text`. Milestone 6b derives context from request fields; future
Files/Notes integrations can centralize canonical Notes folder auto-detection in
the Workbench helper.

`contributes.openProviders` extends the existing contribution registry:

```json
{
  "contributes": {
    "openProviders": [
      {
        "id": "verstak.platform-test.markdown-diagnostic",
        "title": "Platform Test Markdown Diagnostic",
        "priority": 100,
        "component": "MarkdownDiagnosticProvider",
        "supports": [
          {
            "kind": "vault-file",
            "extensions": [".md", ".markdown"],
            "contexts": ["generic-markdown", "notes-markdown"],
            "modes": ["view"]
          },
          {
            "kind": "vault-file",
            "mime": ["text/plain"],
            "extensions": [".txt", ".log"],
            "contexts": ["generic-text"]
          }
        ]
      }
    ]
  }
}
```

Selection uses enabled loaded/degraded provider plugins, resource kind,
request mode, extension/mime, context, user preference, priority, then deterministic
`pluginId/providerId` fallback. If nothing matches, Workbench returns
`status: "no-provider"` and shows the fallback view instead of a core editor.

Disabled/failed/missing-required-capability plugins are excluded from provider
selection at request time by `activeOpenProviders()`. Their contributions may
remain in the registry until the next `ReloadPlugins()` cycle, but they never
match during routing.

Draft app-global preferences are `defaultTextEditorProvider`,
`defaultMarkdownEditorProvider`, and `defaultNotesMarkdownEditorProvider`.
Vault-scoped and per-extension overrides are deferred.

### Default Editor Plugin

The official `verstak.default-editor` plugin (`verstak-official-plugins/plugins/default-editor/`)
provides openProviders for text, generic markdown, and notes-context markdown files.
It uses `api.files.readText` / `api.files.writeText` for file I/O and mounts through
the standard `PluginBundleHost` / provider host mechanism. Core does not import or
reference this plugin directly.

Provider plugins may have no sidebar item — openProviders are contribution points
for workbench routing, not navigation. Plugin Manager displays openProviders in the
contributions summary.

### API methods

`settings`

- `settings.read()` — читает весь settings namespace текущего plugin.
- `settings.read(key)` — читает один ключ.
- `settings.write(key, value)` — обновляет один ключ и пишет namespace обратно.
- `settings.writeAll(settings)` — заменяет settings namespace.
- Backend требует plugin exists, enabled, status `loaded`/`degraded` и permission
  `storage.namespace`.

`capabilities`

- `capabilities.list()` — возвращает текущий capability registry.
- `capabilities.get(name)` — возвращает `{ available, name, pluginId, status }`.
- `capabilities.has(name)` — boolean wrapper над `get`.
- Backend требует, чтобы plugin был enabled/loaded и декларировал dependency на
  `verstak/core/capability-registry/v1` в `requires` или `optionalRequires`.

`commands`

- `commands.register(commandId, handler)` — регистрирует bundled frontend handler.
  Возвращает `Promise<unsubscribe>`.
- `commands.execute(commandId, args)` — backend сначала проверяет plugin status,
  permission `commands.register` и что command объявлен в `contributes.commands`
  именно этим plugin. Затем frontend registry вызывает зарегистрированный handler.
- `commands.executeFor(pluginId, commandId, args)` — то же выполнение, но для
  command другого plugin-provider. Используется host surfaces вроде Files/Notes,
  которые читают contribution action и вызывают объявленный provider handler.
- Если command объявлен в manifest, но handler не зарегистрирован, API возвращает
  понятную ошибку `declared-but-unhandled`.
- Handler registry очищается при component unmount, reload/disable flow и
  `api.dispose()`.
- Shell command palette открывается через `Ctrl+K` / `Cmd+K` или
  `Ctrl+Shift+P` / `Cmd+Shift+P`, показывает commands enabled plugins,
  фильтрует по title/id/plugin и вызывает зарегистрированные bundled frontend
  handlers.

`statusBarItems`

- Shell status bar renders enabled plugin `statusBarItems` contributions.
- Items support `left`, `center`, and `right` positions. Missing position
  defaults to `left`.
- The host refreshes on plugin reload/enable/disable through
  `verstak:plugins-changed`.
- `handler` is preserved in the contribution summary for future action routing;
  current host only renders status labels.

`contributions`

- `contributions.list()` — возвращает весь flattened contribution summary.
- `contributions.list(point)` — возвращает массив contribution records для
  конкретного поля (`fileActions`, `noteActions`, `contextMenuEntries`, etc).
- Runtime records include `pluginId`, so consumer surfaces can call
  `commands.executeFor(record.pluginId, record.handler, args)`.

`events`

- `events.subscribe(eventName, handler)` — frontend-local subscription с backend
  validation permission `events.subscribe`. Возвращает `Promise<unsubscribe>`.
- `events.publish(eventName, payload)` — backend проверяет `events.publish`, затем
  событие dispatch'ится в bundled frontend event bus.
- Handler получает envelope `{ name, pluginId, payload, timestamp }`.
- Subscriptions очищаются при component unmount, reload/disable flow и
  `api.dispose()`.
- Core publishes `file.changed` for Files API mutations and live vault watcher
  changes. Watcher payloads use `operation: "external.create"`,
  `"external.update"`, or `"external.delete"`, include vault-relative `path`,
  `type`, `workspaceRootPath`, and `external: true`.

`files`

- `files.list(relativeDir)` — list directory using a vault-relative path.
- `files.metadata(relativePath)` — returns file/folder/symlink metadata.
- `files.readText(relativePath)` — reads a UTF-8 regular file, with a size limit.
- `files.readBytes(relativePath)` — reads a regular file up to 8 MB and returns
  `{ relativePath, size, mimeHint, dataBase64 }` for bounded binary preview use.
- `files.writeText(relativePath, content, options)` — atomically writes text via
  temp-file-and-rename. `options.createIfMissing` and `options.overwrite`
  control conflicts.
- `files.writeBytes(relativePath, dataBase64, options)` — decodes a base64
  payload up to 8 MB and atomically writes bytes via temp-file-and-rename.
  `options.createIfMissing` and `options.overwrite` control conflicts.
- `files.createFolder(relativePath)` — creates one folder when the parent exists.
- `files.move(from, to, options)` — moves a file or folder; rejects moving a
  folder into itself and conflicts unless `options.overwrite` is true.
- `files.trash(relativePath)` — moves a file/folder into internal
  `.verstak/trash/files/<trashId>/...` and returns trash metadata.
- `files.listTrash()` — returns internal file trash metadata, newest first.
- `files.restoreTrash(trashId, options)` — restores a file/folder from internal
  trash to its original vault-relative path, or to `options.targetPath`.
  Conflicts are rejected unless `options.overwrite` is true.
- `files.deleteTrash(trashId)` — permanently removes a file/folder already in
  internal trash. This operation cannot be undone.
- UI policy: the Deal `Files` plugin displays only live files and folders.
  The global official `verstak.trash` plugin owns listing, restoring, filtering,
  and permanently deleting deleted items across the vault.
- `files.openExternal(relativePath)` — opens a vault-relative file/folder in
  the OS default application.
- `files.showInFolder(relativePath)` — reveals a vault-relative file/folder in
  the OS file manager where the platform supports it.
- Backend requires plugin exists, enabled, status `loaded`/`degraded`, open
  vault, and `files.read`, `files.write`, `files.delete`, or
  `files.openExternal`.
- All paths are canonical vault-relative slash paths. Backslashes, POSIX
  absolute paths, Windows drive paths, UNC/network paths, `..`, null bytes,
  symlink traversal, and public access to `.verstak/` are rejected.
- `.verstak` is reserved case-insensitively: `.verstak`, `.Verstak`, and any
  first path segment with that spelling are internal-only.
- `files.metadata` may report a final symlink as `type: "symlink"`, but
  `files.list` through a symlink directory and all read/write/move/trash
  operations through symlinks are forbidden in Milestone 6a.
- `readText` is limited to UTF-8 regular files up to 2 MB. `readBytes` and
  `writeBytes` are bounded byte contracts up to 8 MB; chunked streaming is
  deferred.
- Live watcher refresh is active while Verstak is running and a vault is open.
  It performs an initial no-event snapshot, then publishes `file.changed` for
  external creates, updates, and deletes outside `.verstak/`. It does not keep a
  persistent snapshot or report what changed while Verstak was closed.

`sync`

- `sync.now()` pushes local operations, pulls remote operations, and returns
  `{ pushed, pulled, serverSequence, conflicts?, applyErrors? }`.
- `conflicts` is an array of server-reported sync conflicts. Conflict objects
  may include `op_id`, `entity_type`, `entity_id`, `reason`, and additional
  server fields. The Sync plugin must show conflict details instead of only a
  count, and it must not silently resolve or overwrite local data.
- `applyErrors` lists local apply failures for pulled operations. These are
  user-visible warnings and do not imply that sync was fully successful.
- Transport push/pull uses bounded retry/backoff for transient HTTP/network
  failures. Client/auth errors are not retried.

`dispose`

- `dispose()` вызывается host'ом при cleanup. Plugin code обычно не вызывает его
  напрямую. Он удаляет зарегистрированные command handlers и event subscriptions.

### Runtime boundaries

| Layer | Current status |
|---|---|
| Bundled frontend runtime | Functional for settings, capabilities, commands, events and text Files API |
| Backend validation | Checks plugin exists, enabled/loaded state, permissions and declarations |
| Security boundary | Not implemented; bundled plugins share the desktop frontend JS context |
| Sidecar/RPC/sandbox | Not implemented |

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

API объект передаётся в `mount()` и содержит plugin-scoped методы текущего
bundled runtime. Это реальный runtime contract для cooperative bundled plugins,
но не sandbox/security boundary.

| Свойство | Статус | Описание |
|---|---|---|
| `api.pluginId` | ✅ Работает | ID плагина |
| `api.settings.read(key?)` | ✅ Работает | Читает plugin-scoped settings через backend bridge |
| `api.settings.write(key, value)` | ✅ Работает | Пишет один settings key через backend bridge |
| `api.settings.writeAll(settings)` | ✅ Работает | Заменяет settings namespace плагина |
| `api.capabilities.list()` | ✅ Работает | Возвращает capability registry |
| `api.capabilities.get(id)` | ✅ Работает | Возвращает capability entry/status |
| `api.capabilities.has(id)` | ✅ Работает | Boolean wrapper над `get` |
| `api.commands.register(id, handler)` | ✅ Работает | Регистрирует bundled frontend handler для объявленной command |
| `api.commands.execute(id, args)` | ✅ Работает | Валидирует declaration/permission/backend state и вызывает bundled handler |
| `api.commands.executeFor(pluginId, id, args)` | ✅ Работает | Выполняет handler другого plugin-provider после backend validation |
| `api.contributions.list(point?)` | ✅ Работает | Возвращает flattened contribution summary или массив по point |
| Command Palette UI | ✅ Работает | `Ctrl/Cmd+K`, фильтр enabled plugin commands, вызов registered frontend handlers |
| `api.events.publish(type, payload)` | ✅ Работает | Валидирует permission и публикует во frontend event bus |
| `api.events.subscribe(type, handler)` | ✅ Работает | Валидирует permission и подписывает handler на frontend event bus |
| `api.files.list(relativeDir)` | ✅ Работает | Список vault-relative директории, `.verstak` скрыта |
| `api.files.metadata(relativePath)` | ✅ Работает | Metadata для файла/папки/symlink без чтения содержимого |
| `api.files.readText(relativePath)` | ✅ Работает | Читает UTF-8 regular file до 2 MB |
| `api.files.readBytes(relativePath)` | ✅ Работает | Читает regular file до 8 MB как base64 payload |
| `api.files.writeText(relativePath, content, options)` | ✅ Работает | Atomic text write с явным create/overwrite policy |
| `api.files.writeBytes(relativePath, dataBase64, options)` | ✅ Работает | Atomic bounded byte write до 8 MB с явным create/overwrite policy |
| `api.files.createFolder(relativePath)` | ✅ Работает | Создаёт vault-relative folder |
| `api.files.move(from, to, options)` | ✅ Работает | Move file/folder с conflict и path-policy checks |
| `api.files.trash(relativePath)` | ✅ Работает | Перемещает в internal trash и сохраняет metadata исходного объекта |
| `api.files.listTrash()` | ✅ Работает | Возвращает metadata internal file trash |
| `api.files.restoreTrash(trashId, options)` | ✅ Работает | Восстанавливает из internal trash, conflict-safe по умолчанию |
| `api.files.deleteTrash(trashId)` | ✅ Работает | Необратимо удаляет запись и payload из internal trash |
| `api.files.openExternal(relativePath)` | ✅ Работает | Открывает vault file/folder во внешнем приложении, требует `files.openExternal` |
| `api.files.showInFolder(relativePath)` | ✅ Работает | Показывает vault file/folder в системном файловом менеджере, требует `files.openExternal` |
| `api.workbench.openResource(request)` | ✅ Работает | Routes vault resources to `openProviders` |
| `api.workbench.editResource(request)` | ✅ Работает | Same routing, forcing `mode: "edit"` |
| `api.sync.now()` | ✅ Работает | Push/pull с bounded retry/backoff для transient HTTP/network failures |
| `api.sync.status()` | ✅ Работает | Возвращает configured/connected/error/revoked state, lastError, unpushed count |
| `api.dispose()` | ✅ Работает | Очищает command handlers и event subscriptions текущего API instance |

Ограничения:

- permissions/capabilities checks являются contract/policy checks;
- bundled frontend plugins исполняются в общем JS-контексте;
- malicious JS не изолирован;
- sidecar process lifecycle, RPC transport и sandbox enforcement ещё не
  реализованы.
- Files paths are slash-only vault-relative contract paths; backslashes,
  Windows absolute paths, UNC paths, `.verstak` variants, traversal and symlink
  operations are rejected by backend policy checks.

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

1. Unregister contributions всех plugins предыдущего discovery.
2. Unregister all non-core capabilities.
3. Re-register core capabilities + vault + Deal manager (если открыт).
4. Re-scan discovery directories и повторить capability resolution.
5. Register contributions для loaded/degraded plugins (disabled/failed — не регистрируются).
6. Update plugins list.

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
│ │ [icon]   │ (padding: 1.5rem)                    │ │
│ │ Plugin   │                                      │ │
│ │   Manager│                                       │ │
│ │          │                                       │ │
│ │ Plugins  │                                       │ │
│ │ [icon] item1                                    │ │
│ │ [icon] item2                                    │ │
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

## Дело: core capability

Дело — это физическая папка верхнего уровня внутри vault root. Filesystem
является source of truth для списка Дел.

Пример:

```
<vault>/
  Deal/
    Notes/
  Project/
  ClientA/
  .verstak/
```

Нет единого `<vault>/Deal/` контейнера для всех Дел. Папка `Deal/` может быть
обычным Делом, но `Project/` и `ClientA/` являются такими же Делами на том же
уровне.

### Хранение

Существование и список Дел хранятся только в filesystem:

- `ListWorkspaces()` читает top-level directories из vault root.
- `.verstak`, reserved/internal directories, top-level files и symlinks не
  считаются Делами.
- `.verstak/workspace*.json` не является source of truth для списка Дел.
- Нет virtual workspace tree, которое
  мапится на произвольные папки.

`.verstak` может хранить только metadata, которая не заменяет filesystem:

- UI state: выбранное Дело, expanded folders, sort/pin state, preferences.
- Semantic snapshot: applied template snapshot, enabled feature areas, exact
  `workspaceTools`, and folder conventions.

При создании Дела runtime генерирует UUID и сохраняет его в
`.verstak/workspace.json`. `workspaceId` — постоянная личность Дела;
`workspaceRootPath` и `workspaceName` — изменяемый адрес и presentation fields.
Переименование обновляет путь, но не ID. Новая папка с прежним именем получает
другой ID и не может перехватить старые привязки.

```json
{
  "workspaceId": "1eb7cc69-52e8-4a18-ae2a-50d5229c5b60",
  "workspaceName": "Project",
  "createdFromTemplate": {
    "templateId": "project",
    "templateName": "Project",
    "templateVersion": 1,
    "appliedAt": "2026-06-19T12:00:00Z",
    "workspaceTools": [
      "verstak.notes",
      "verstak.files",
      "verstak.todo",
      "verstak.journal",
      "verstak.activity",
      "verstak.browser-inbox"
    ]
  },
  "features": {
    "notes": true,
    "files": true,
    "todo": true,
    "journal": true,
    "activity": true,
    "browser-inbox": true
  },
  "folders": {
    "notes": "Notes",
    "files": "Files"
  },
  "workspaceTools": [
    "verstak.notes",
    "verstak.files",
    "verstak.todo",
    "verstak.journal",
    "verstak.activity",
    "verstak.browser-inbox"
  ]
}
```

Если original template удалён или изменён позже, существующее Дело
открывается по сохранённому snapshot и не мутирует автоматически. Template
update/migration может быть только явной future feature. Если metadata
отсутствует, Дело открывается как generic Deal минимум с `files: true`.

### API

- `ListWorkspaces()` — список top-level папок Дел с их `workspaceId`.
- `ListWorkspaceTemplates()` — selectable built-in templates с presentation
  metadata и `workspaceTools`.
- `CreateWorkspace(name, templateId?)` — создать `<vault>/<name>/`, применить
  template один раз, сохранить snapshot metadata.
- `RenameWorkspace(oldName, newName)` — физически переименовать top-level папку
  Дела и обновить metadata name, не меняя `workspaceId`.
- `TrashWorkspace(name)` — перенести весь top-level folder Дела в internal
  trash policy.
- `GetWorkspaceMetadata(name)` — прочитать metadata или вернуть generic fallback.
- `UpdateWorkspaceMetadata(name, patch)` — обновить metadata без влияния на
  существование Дела.

Deprecated compatibility APIs:

- `GetWorkspaceTree()` — flat view, derived from top-level folders. Не дерево.
- `CreateWorkspaceNode(...)` — wrapper over `CreateWorkspace`.
- `RenameWorkspaceNode(...)` — wrapper over `RenameWorkspace`.
- `ArchiveWorkspaceNode(...)` — wrapper over `TrashWorkspace`.
- `MoveWorkspaceNode(...)` — unsupported; old nested/mapped moves are rejected.
- `GetCurrentWorkspaceNode()` / `SetCurrentWorkspaceNode(...)` — wrappers over
  selected top-level Deal UI state.

Эти методы существуют только для постепенного frontend/Wails cleanup. Они не
должны создавать или сохранять nested workspace tree и не должны восстанавливать
`WorkspaceNode.path` mapping.

### Capability

`verstak/core/workspace/v1` — техническое имя capability; оно регистрируется
только когда vault открыт и Deal manager инициализирован.

### Правила

- Имя Дела — один safe folder name, не path.
- Reject: empty, slash, backslash, absolute-looking paths, `..`, null byte,
  `.verstak`, reserved/internal names, symlink Дел, conflicts.
- WorkspaceItems получают техническое поле `workspaceRootPath`, равное имени top-level папки
  (`Project`, `ClientA`, etc). Files plugin показывает именно эту папку.
- Files API остаётся raw vault-relative API: `Project/Notes/example.md`,
  `Project/docs/file.md`, `Test/readme.md`.
- Notes are ordinary Markdown files under `<Deal>/Notes/`; нет
  `.verstak/notes`, UUID note storage или второго source of truth для note
  content.

### Lifecycle Events

Runtime publishes lifecycle events Дел после успешных операций:

- `workspace.created` — payload includes `operation: "create"`, `workspaceId`,
  `workspaceRootPath`, `workspaceName`, and optional `templateId`.
- `workspace.renamed` — payload includes `operation: "rename"`, `workspaceId`,
  new `workspaceRootPath` / `workspaceName`, and previous path fields.
- `workspace.trashed` — payload includes `operation: "trash"`, `workspaceId`,
  `workspaceRootPath`, `workspaceName`, `trashId`, `trashPath`, and `deletedAt`.
- `workspace.selected` — payload includes `operation: "select"`, `workspaceId`,
  `workspaceRootPath`, and `workspaceName`.

Official Activity subscribes to these events and stores them through the normal
activity provider path.

### UI

Список Дел в sidebar:
- Flat list of top-level папок Дел.
- Создать Дело, переименовать Дело, переместить Дело в корзину.
- Selection is stored as selected Deal name.
- No expand/collapse tree and no case/folder node creation in core.

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
