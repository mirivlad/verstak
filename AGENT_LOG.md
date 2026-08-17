# Verstak audit — agent continuity log

> Этот файл — каноническая точка продолжения ревизии Verstak между чатами.
> Перед любой новой работой по ревизии сначала прочитать этот файл и сверить указанные ветки/PR/коммиты с GitHub. После каждого законченного среза или смены рабочей точки обновлять файл.

## Цель ревизии

Цель — не «дотянуть до релиза», а последовательно поднять качество всего проекта: пройти UI/UX, функциональные шероховатости, архитектурные хвосты, старые сущности и незавершённые миграции во всех репозиториях Verstak.

Репозитории:

- `mirivlad/verstak` — desktop/core/shell;
- `mirivlad/verstak-official-plugins` — официальные плагины;
- `mirivlad/verstak-sdk` — plugin SDK/contracts;
- `mirivlad/verstak-sync-server` — sync server;
- `mirivlad/verstak-browser-extension` — Firefox extension;
- `mirivlad/verstak-docs` — документация.

## Состояние на 2026-08-17

### Уже завершено и в `main`

#### Инфраструктура реального UI-аудита

`mirivlad/verstak` PR #1 — `ci: add real GUI audit for Verstak`, merge `01429cd`.

- real Wails/WebKitGTK GUI audit через Actions;
- deterministic Playwright visual audit;
- shipping-like набор official plugins;
- screenshot/artifact evidence;
- исправлены старые CI gaps вокруг plugin build, SVG/ImageMagick и тестовых зависимостей.

#### Overview — UX и информационная архитектура

`mirivlad/verstak` PR #2 — `fix: separate Overview resume and attention signals`, merge `728c338`.

- `Продолжить работу` больше не дублирует pending/attention сигналы;
- attention остаётся отдельной областью;
- исправлена stale localization;
- Browser Inbox labels уточнены.

`mirivlad/verstak` PR #3 — `refactor: prioritize Overview work context`, merge `8a74b3d`.

- главный порядок чтения: где остановился → что требует реакции → навигация/сводка → недавнее;
- `Продолжить работу` поднято выше;
- убрана дублирующая карточка attention.

Cross-repo provider migration:

- `verstak-sdk` `4659f13` — Overview provider SDK contract;
- `verstak-official-plugins` `d5aebca` — official Overview providers;
- `verstak-official-plugins` `e8bdfca` — attention priority;
- `verstak` PR #4, merge `9691858` — shell потребляет generic Overview providers вместо знания внутренних storage keys official plugins.

`verstak` PR #5 — `refactor: finish Overview shell cleanup`, merge `ad4f894`.

- `TodaySurface` → `OverviewSurface`;
- удалены Today-era имена/классы;
- generic navigation contract доведён до конца.

#### Release tooling

`verstak` PR #6 — `ci: build and publish desktop releases on GitHub`, merge `840f4b8`.

- GitHub Actions собирает Linux `.deb`, `.AppImage`, Windows portable `.zip`;
- release smoke/checksums;
- публикация GitHub Release только после проверок.

#### Global Search — provider migration

`verstak-sdk`:

- `8b62494` — Search provider result contract;
- `182ccbc` — workspace path resolution API.

`verstak-official-plugins`:

- `29d1599` — Search provider остаётся активен в background;
- `7fc2cfc` — domain-owned Search providers;
- `5764916` — generic/responsive provider navigation;
- `532fa29` — provider сам владеет labels категорий File/Folder.

`verstak` PR #7 — `refactor: consume generic Search providers`, merge `8ff7390`.

- GlobalSearch больше не читает storage official plugins напрямую;
- domain results приходят через generic `searchProviders`;
- Deals индексируются из semantic workspace tree;
- provider actions generic (`workspace`, `workspace-item`, `view`, `resource`);
- partial provider failure не ломает общую выдачу;
- сохранены ranking/dedupe и RU/EN keyboard correction.

#### E2E ↔ official plugin runtime alignment, первый большой срез

`verstak` PR #8 — `test: use official plugin manifests in E2E`, merge `3ffcab0`.

- 13 existing official plugin fixtures используют настоящие `plugin.json`;
- asset loader следует `manifest.frontend.entry/style`;
- убраны вторые копии manifest metadata;
- Plugin Manager risk regression сверяется с реальными permissions;
- полный Playwright на момент PR: 166/166.

Найденные тогда хвосты:

- `file-preview` был shipping plugin, но отсутствовал в базовой E2E plugin model;
- `folder-appearance` оказался незавершённой миграцией в core;
- в `wails-mock.js` оставались synthetic UI bundles для `Trash`, `Sync`, `Platform Test`.

#### Folder appearance — миграция полностью завершена

`verstak` PR #9 — `refactor: move folder appearance fully into core`, merge `eaf4b8c`.

- WorkspaceTree использует core Wails API;
- shell больше не impersonate'ит `verstak.folder-appearance`;
- legacy plugin data мигрируются best-effort/idempotent в core storage;
- existing core values выигрывают;
- legacy source сохраняется для recoverability.

`verstak-official-plugins` PR #8 — `refactor: retire folder appearance plugin`, merge `7381a94`.

- obsolete plugin удалён;
- build теперь отклоняет неизвестные `contributes.*`, чтобы неподдерживаемые contribution points не проходили молча.

#### CI stabilization, найденный во время ревизии

`verstak` PR #10 — `test: stop asserting sync scan wall-clock ordering`, merge `2d622e5`.

- убран flaky assertion сравнения двух одиночных wall-clock scan samples;
- functional scan verification оставлена;
- performance ordering должен проверяться benchmark'ами, а не noisy shared runner timing.

### Текущая незавершённая работа, реально сохранённая на GitHub

Репозиторий: `mirivlad/verstak`

Ветка: `audit/file-preview-e2e-catalog`

Состояние относительно `main` (`eaf4b8c`): **ahead by 3, behind by 0**, PR не открыт.

Коммиты:

1. `453f0fb` — `test: cover shipped file preview plugin`;
2. `9d906f1` — `ci: stage file preview E2E catalog alignment`;
3. `ad922b8` — `ci: verify file preview E2E catalog alignment`.

В ветке пока только staging/verification слой:

- `frontend/e2e/file-preview.spec.js`;
- `.github/scripts/patch_file_preview_e2e_catalog.py`;
- `.github/workflows/one-shot-file-preview-e2e-catalog.yml`.

Permanent изменения `wails-mock.js`/source-contract ещё НЕ были закоммичены.

Последний one-shot run `31992774370` завершился failure на шаге **Apply File Preview catalog alignment**.

Точный blocker:

```text
manual vault plugin inventory still present
```

Patcher успевает подготовить переход к единому `officialPluginFixtures`, но после своих замен всё ещё находит `vaultPluginState.enabledPlugins.push(...)`. Значит в текущем `wails-mock.js` есть ещё один ручной путь изменения inventory, который patcher не учитывает.

### Следующий конкретный шаг

1. На ветке `audit/file-preview-e2e-catalog` найти все оставшиеся `vaultPluginState.enabledPlugins.push(...)` и понять, что это за runtime path (не удалять механически).
2. Исправить модель так, чтобы shipped File Preview входил в canonical E2E catalog, а runtime enable/install behavior не ломался.
3. Запустить focused File Preview E2E + полный E2E + shell source-contract.
4. После успешного permanent commit удалить one-shot patcher/workflow из итогового diff.
5. Прогнать обычные `check`, `visual-audit`, `gui-audit`.
6. Открыть PR и после зелёного merge зафиксировать результат здесь.

### Более поздняя рабочая мысль из зависшего чата, НЕ сохранённая на GitHub

После File Preview обсуждение/работа успели перейти к отдельному срезу `official-plugin-runtime-alignment`.

В чате было зафиксировано намерение/локальная работа:

- ветка `audit/official-plugin-runtime-alignment`;
- workflow `.github/workflows/official-plugin-runtime-alignment.yml`;
- checker `scripts/check-official-plugin-runtime-alignment.mjs`;
- checker должен был проверять:
  - отсутствие живых synthetic bundle implementations в `wails-mock.js`;
  - использование official plugin sources/manifests asset-loader'ом;
  - отсутствие unused legacy bundle functions;
  - отсутствие ручной подмены manifests при наличии official manifest;
  - только допустимые host/API mocks для каждого official plugin.

Также было отмечено отдельное расхождение mock/runtime: browser mock-контекст делает вид, что `window.go` готов, хотя при старте реального Wails runtime это не так; lazy initialization маскировал проблему в обычных тестах.

**Важно:** на момент этого лога ветки `audit/official-plugin-runtime-alignment` и файла `check-official-plugin-runtime-alignment.mjs` нет ни в `verstak`, ни в `verstak-official-plugins` на GitHub. Поэтому это НЕ каноническое сохранённое состояние и продолжать с него вслепую нельзя. Сначала надо завершить/разобрать сохранённую `audit/file-preview-e2e-catalog`, затем при необходимости заново создать runtime-alignment срез на актуальном `main`.

## Что ещё не пройдено полностью в текущей ревизии

- оставшиеся synthetic UI bundles `Trash`, `Sync`, `Platform Test` в desktop E2E/mock слое — повторно проверить после File Preview среза;
- полный screen-by-screen UI/UX audit остальных surfaces после Overview/Search;
- `verstak-sync-server` — отдельный системный/UX/API audit в этой ревизии ещё не завершён;
- `verstak-browser-extension` — отдельный audit ещё не завершён;
- `verstak-docs` — сверка документации с новой архитектурой после продуктовых срезов ещё не завершена;
- `verstak-sdk` и `verstak-official-plugins` проверять не изолированно, а каждый раз на cross-repo contract drift с desktop.

## Правила ведения этого лога

После каждого заметного действия добавить короткую запись в раздел ниже. Обязательно фиксировать:

- repo + branch;
- что проверено/изменено;
- commit/PR, если появился;
- результат tests/CI;
- blocker, если работа остановилась;
- **один конкретный следующий шаг**, чтобы новый чат не занимался повторной разведкой.

Не писать сюда длинный поток команд. Это continuity log, а не terminal transcript.

## Хронология агента

### 2026-08-17 — восстановление после зависшего чата

- Сверены все 6 репозиториев Verstak и recent commits 16–17 августа.
- Подтверждено, что PR #1–#10 desktop и связанные SDK/official-plugin срезы выше уже находятся в `main`.
- Найдена реально незавершённая ветка `audit/file-preview-e2e-catalog` (3 commits ahead of main, no PR).
- Проверен последний one-shot Actions run; blocker — `manual vault plugin inventory still present`.
- Проверено, что упоминавшийся позднее `audit/official-plugin-runtime-alignment` не был сохранён на GitHub; его нельзя считать восстановимой рабочей веткой.
- Создан этот `AGENT_LOG.md` как постоянная точка продолжения.
- Следующий шаг: разобрать оставшийся manual vault plugin inventory на `audit/file-preview-e2e-catalog` и довести File Preview E2E catalog alignment до PR.
