# Verstak audit — agent continuity log

> Каноническая точка продолжения ревизии между чатами. Перед новой работой сначала прочитать этот файл и сверить указанные ветки/PR/коммиты с GitHub. После каждого законченного среза или смены рабочей точки обновлять файл.

## Цель

Последовательно довести качество всего Verstak: UI/UX, функциональные шероховатости, старые сущности, незавершённые миграции, архитектурный drift и cross-repo contracts.

Репозитории:

- `mirivlad/verstak` — desktop/core/shell;
- `mirivlad/verstak-official-plugins` — official plugins;
- `mirivlad/verstak-sdk` — plugin SDK/contracts;
- `mirivlad/verstak-sync-server` — sync server;
- `mirivlad/verstak-browser-extension` — Firefox extension;
- `mirivlad/verstak-docs` — документация.

## Завершённые срезы 2026-08-16 — 2026-08-17

### Desktop / shell

- PR #1 `01429cd` — real Wails/WebKitGTK GUI audit + deterministic visual audit.
- PR #2 `728c338` — Overview: разделены resume и attention signals.
- PR #3 `8a74b3d` — Overview IA: work context выше secondary summary.
- PR #4 `9691858` — Overview shell переведён на generic provider contributions.
- PR #5 `ad4f894` — `TodaySurface` окончательно очищен/переименован в Overview.
- PR #6 `840f4b8` — GitHub release build/publish pipeline.
- PR #7 `8ff7390` — Global Search потребляет generic Search providers и semantic workspace tree.
- PR #8 `3ffcab0` — E2E использует настоящие manifests official plugins вместо вторых копий metadata.
- PR #9 `eaf4b8c` — folder appearance полностью принадлежит core, legacy plugin data мигрируются.
- PR #10 `2d622e5` — убран flaky wall-clock ordering assertion sync scan test.
- PR #11 `1fbddce` — shipped File Preview включён в canonical E2E official plugin catalog.

### SDK / official plugins

- `verstak-sdk` `4659f13` — Overview provider contract.
- `verstak-sdk` `8b62494` — Search provider result contract.
- `verstak-sdk` `182ccbc` — workspace path resolution API.
- `verstak-official-plugins` `d5aebca` / `e8bdfca` — Overview providers + attention priority.
- `verstak-official-plugins` `29d1599` / `7fc2cfc` / `5764916` / `532fa29` — Search provider lifecycle/domain ownership/navigation/labels.
- `verstak-official-plugins` PR #8, `7381a94` — obsolete `verstak.folder-appearance` удалён; build rejects unknown `contributes.*`.

## Последний завершённый срез: File Preview / canonical E2E catalog

Desktop PR #11 merged as `1fbddce26bc6fa1364e67691bd7062199362936c`.

Что исправлено:

- shipping `verstak.file-preview` добавлен в базовую E2E plugin model;
- official plugin runtime state и vault enabled/desired state строятся из одного `officialPluginFixtures`;
- File Preview использует реальный `plugin.json`, frontend source и RU/EN catalogs;
- реальный image `openProvider` проверяется отдельным E2E, включая disable flow;
- `no-provider` test теперь использует действительно неподдерживаемое расширение, а не PNG;
- Plugin Manager tests больше не фиксируют исторические числа 13/14, а выводят expectations из текущего catalog;
- temporary migration patchers/workflows удалены из финального diff.

Проверка финального head:

- focused File Preview E2E: 2/2;
- полный Playwright: 169/169;
- штатный `check`: success;
- `visual-audit`: success;
- real `gui-audit`: success.

### Cross-repo drift, найденный этим срезом

Штатный contract check после File Preview сначала упал на stale generated icons. Причина оказалась не PR #11, а порядок предыдущих cross-repo merges:

1. desktop folder-appearance PR #9 был зелёным, пока `verstak.folder-appearance` ещё существовал в official-plugins;
2. затем official-plugins PR #8 удалил plugin с manifest icon `palette`;
3. desktop `scripts/generate-icon-assets.mjs` читает manifests соседнего `verstak-official-plugins`, поэтому canonical core icon set изменился уже после desktop check;
4. generated `core.js` / `sprite.js` стали stale.

Assets регенерированы штатным генератором: `palette` перемещён из first-paint core в lazy sprite, core count 36 → 35. Generator + `--check` и все обычные CI после этого зелёные.

**Новое правило ревизии:** после серии cross-repo merges повторно прогонять consumer repo на уже объединённом состоянии зависимостей. Зелёность каждого PR по отдельности не гарантирует зелёность итоговой комбинации.

## Текущая рабочая точка

Следующий срез: **official plugin runtime alignment**.

Рабочая ветка: `audit/official-plugin-runtime-alignment` (создавать от актуального `main` после этого log commit).

Подтверждённый drift в `frontend/src/lib/test/wails-mock.js`:

E2E уже использует реальные manifests всех 14 official plugins, но asset loader всё ещё исполняет четыре собственные synthetic frontend implementations вместо shipping code:

1. `Platform Test` → synthetic `platformTestBundle()`;
2. `Trash` → synthetic `trashPluginBundle()`;
3. `Sync` → synthetic `syncPluginBundle()`;
4. `Search` → synthetic `searchPluginBundle()`.

`Search` — дополнительный хвост, который не был перечислен в раннем плане ревизии.

Фактические shipping entries:

- Platform Test: `frontend/src/index.js` — raw source можно грузить напрямую;
- Trash: `frontend/src/index.js` — raw source можно грузить напрямую;
- Search: `frontend/src/index.js` — raw source можно грузить напрямую;
- Sync: `frontend/dist/index.js` — должен использовать результат реальной сборки official plugins.

Host/backend/state mocks сами по себе допустимы и нужны. Цель — убрать **вторые реализации frontend plugins**, а не эмулировать весь Wails backend настоящим сервером.

Также из зависшего прошлого чата сохранён отдельный вопрос для проверки в этом срезе: mock может делать `window.go` доступным раньше, чем настоящий Wails runtime; lazy initialization мог скрывать startup-order bug. Не считать это подтверждённым дефектом, пока не будет воспроизведения/контракта.

## Следующие конкретные шаги

1. Создать `audit/official-plugin-runtime-alignment` от текущего `main`.
2. Добавить source-contract/checker, который сначала фиксирует четыре живых synthetic frontend bundles как drift.
3. Переводить plugins по одному на real source/dist, сохраняя существующие backend/API mocks:
   - Trash;
   - Search;
   - Platform Test;
   - Sync.
4. После каждого перехода запускать focused E2E соответствующего plugin/domain, затем полный E2E.
5. Удалить ставшие мёртвыми bundle functions и запретить их возврат source-contract'ом.
6. Проверить startup ordering `window.go` отдельно, не смешивая с frontend-source migration, если потребуется продуктовая правка.
7. Прогнать `check`, `visual-audit`, `gui-audit`, открыть/merge PR и обновить этот лог.

## Что ещё не пройдено полностью

- полный screen-by-screen UI/UX audit surfaces после Overview/Search;
- `verstak-sync-server` — отдельный системный/API audit;
- `verstak-browser-extension` — отдельный audit;
- `verstak-docs` — финальная сверка документации с изменившейся архитектурой;
- `verstak-sdk` / `verstak-official-plugins` — продолжать проверять на cross-repo contract drift с desktop.

## Правила ведения лога

После каждого заметного действия фиксировать:

- repo + branch;
- что проверено/изменено;
- commit/PR;
- tests/CI;
- blocker;
- **один конкретный следующий шаг**.

Не хранить здесь terminal transcript; только информацию, необходимую следующему чату.

## Хронология агента

### 2026-08-17 — восстановление continuity

- Восстановлены PR #1–#10 и cross-repo состояние.
- Найдена незавершённая `audit/file-preview-e2e-catalog` и точный старый blocker.
- Создан `AGENT_LOG.md`.

### 2026-08-17 — File Preview срез завершён

- Исправлен ложный guard вокруг runtime `enabledPlugins.push(...)` — runtime mutation сохранена, убраны только duplicate default inventories.
- Исправлен новый E2E: актуальный Settings helper и существующий parent path.
- Убраны stale assumptions PNG=no-provider и fixed plugin counts.
- Full E2E 169/169.
- Найден и исправлен cross-repo generated icon drift после retirement folder-appearance.
- Финальные `check` / visual / real GUI зелёные.
- PR #11 merged as `1fbddce`.
- Следующий шаг: создать `audit/official-plugin-runtime-alignment` и убрать четыре synthetic frontend implementations из E2E runtime.
