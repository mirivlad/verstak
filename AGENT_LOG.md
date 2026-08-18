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

## Завершённый срез 2026-08-18: детерминированность контрактов и runtime alignment

### Поправка к записи выше

Правило «перепрогонять consumer после cross-repo merges» описывало симптом, а не причину, и потому не работало. Настоящая причина найдена 2026-08-18.

`verstak-desktop/plugins/` — gitignored install copy. И `scripts/generate-icon-assets.mjs`, и `frontend/tests/icon-contract-test.mjs` читали её **наравне** с sibling-репозиторием исходников. Значит сгенерированные `core.js` / `sprite.js` считались актуальными относительно того, что случайно установлено на конкретной машине:

- ретайрнутый плагин, забытый в install copy, затягивает свою иконку обратно в core set;
- ещё не установленный плагин молча из него выпадает, и его иконка коммитится как «актуальная».

Один и тот же коммит красный на одной машине и зелёный на другой. `palette` жил только в устаревшей локальной установке плагина, которого в upstream уже нет — отсюда ощущение, что drift возвращается.

Исправлено в PR #12: sibling-репозиторий — единственный источник истины, install copy остаётся fallback'ом только для одиночного desktop checkout, а `--check` отказывается по ней судить и говорит об этом вслух.

**Правило вместо прежнего:** contract check не имеет права читать gitignored пути. Если проверка зависит от состояния машины, она не проверка.

### Что сделано

- desktop PR #12 `031bf3f` — детерминированный icon contract + nightly `schedule` на `check.yml`.
- official-plugins PR #10 `68c0cf5` — job `desktop-contract`: изменение манифестов прогоняет desktop-контракт **на том PR, который его ломает**. 13 секунд, секретов не требует.
- desktop PR #13 `246cf18` — срез official plugin runtime alignment закрыт.
- desktop PR #14 `bf4aff8` — Wails binding contract выводится из Go API.

### Срез official plugin runtime alignment

Sync — последний из четырёх — переведён на реальный frontend. Единственный, кто поставляет собранный вывод, поэтому mock грузит `frontend/dist`, как уже делает Import.

Разница оказалась содержательной: синтетический бар рисовал захардкоженный `Synced` и собирал open-settings CustomEvent руками; настоящий выводит подпись из статуса через plugin API и зовёт `api.ui.openSettings()`. Второе — прямо предмет `settings-section-request.spec.js`, который покрывает регрессию с вызовом **без** panel id; mock её имитировал, а не воспроизводил.

Удалено пять мёртвых фабрик (`defaultEditorBundle`, `simplePluginBundle`, `syncPluginBundle`, `browserInboxBundle`, `todoBundle`) — у каждой была ровно одна ссылка, собственное определение. `wails-mock.js`: 2790 → 1856 строк. Source contract называет их поимённо.

### Wails binding contract

Проверка была списком из десяти имён, записанных руками, — то же усилие памяти, что и не забыть пересобрать биндинги, поэтому не ловила ничего. Теперь множество выводится из Go: 133 метода вместо 10.

Дрейф в самих биндингах оказался слабее, чем выглядел: `models.ts` — только типы, и шелл (обычный JS) его не импортирует; в `App.js` расхождение было в порядке сортировки, то есть запись когда-то добавили руками. Функционально сломано не было ничего.

### Ветки

Удалена 41 смёрженная ветка в шести репозиториях. Важно для следующего чата: `git branch --merged` показывал смёрженными 6 из них, потому что PR #1–#11 шли squash'ем и их коммиты не предки `main`. Сверять надо по состоянию PR, иначе чистка оставляет почти весь мусор.

Намеренно оставлены как несмёрженные:

- `verstak` / `verstak-official-plugins` — `rescue/folder-experiment-2026-07-18`;
- `verstak-sdk` / `verstak-browser-extension` — `fix/beta-readiness-2026-07-20`, одиночные коммиты `build: update vulnerable * tooling` от 20 июля. Похоже на брошенные обновления уязвимых зависимостей — проверить отдельно.

## Текущая рабочая точка

Релиз v0.1.3 — desktop + official-plugins синхронно. До него последний desktop-релиз был v0.1.2 от 28 июля, а official-plugins — beta от 23 июля, при том что оболочка с тех пор научилась потреблять Overview- и Search-провайдеры, которых в опубликованных плагинах нет. Три недели работы ревизии до владельца не доезжали.

## Что ещё не пройдено полностью

Приоритеты уточнены по факту осмотра 2026-08-18.

**Повышено:**

- продуктовый UX. Из 11 desktop PR ревизии пользователь увидел два (#2, #3). Остальное — тесты, контракты, CI. В `docs/UX_UI_REFACTOR_PLAN.md` не доделаны пункты 4 (эргономика Plugin Manager) и 5 (Files/Workbench: контекстное меню, клавиатура, выделение);
- `verstak-docs` — всё ещё «Today flow» вместо Overview, ни слова про переезд folder appearance в core;
- `internal/api/app.go` — 5335 строк, 205 методов на одной структуре. Wails требует одну структуру, но не один файл; разнести на `app_plugins.go` / `app_workspace.go` / `app_sync.go` / `app_files.go` механически безопасно;
- `StatusIncompatible` объявлен в `internal/core/plugin/plugin.go:241` и **не присваивается нигде**. `apiVersion` обязателен к заполнению, но ни разу не сверяется с версией хоста. Для платформы с плагинами это значит, что плагин под старый SDK продолжит «загружаться» и упадёт в рантайме с невнятной ошибкой вместо честного статуса.

**Понижено:**

- `verstak-sync-server` — осмотрен, в лучшей форме, чем предполагал прежний план: 72 теста, среди них изоляция арендаторов, CSRF, доверие proxy-заголовкам, SMTP header injection, ограниченность памяти rate-limiter'а;
- `verstak-browser-extension` — 11 тестовых скриптов, свой CI, свежий релиз 2.1.0.

## Метод

Отдельно для следующего чата: прошлый агент правил файлы, пуша одноразовые GitHub Actions workflow, которые патчат исходник в CI, а потом workflow, который патчер удаляет. Три коммита строительных лесов на одно содержательное изменение, коммиты авторства `github-actions[bot]`, diff нельзя отревьюить до применения. Если правки можно делать локально — делать локально.

## Правила ведения лога

После каждого заметного действия фиксировать:

- repo + branch;
- что проверено/изменено;
- commit/PR;
- tests/CI;
- blocker;
- **один конкретный следующий шаг**.

Не хранить здесь terminal transcript; только информацию, необходимую следующему чату.

Отдельно: этот файл — не архив. Запись, которая описывала симптом как причину, дезориентировала следующего агента сильнее, чем её отсутствие. Найдя настоящую причину, править запись, а не дописывать рядом.
