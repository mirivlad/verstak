# Verstak audit — agent continuity log

> Каноническая точка продолжения ревизии между чатами. Перед новой работой сначала прочитать этот файл и сверить указанные ветки/PR/коммиты с GitHub. После каждого законченного среза или смены рабочей точки обновлять файл.

## UX-правки Actions по проверке владельцем: 2026-10-27

- Тот же этап, три ветки `feat/actions-2`; main и релизы не меняются.
- Постоянная памятка убрана. Компактная карточка запускает действие; кнопки
  редактирования, удаления, порядка и отзыва — иконками при hover/focus.
- Общий IconPicker выделен из дерева папок и доступен через `api.ui.pickIcon`;
  DOM-иконки используют тот же lazy sprite. SDK контракт и mock обновлены.
- Аргументы вводятся строкой с кавычками; хранилище остаётся argv-массивом.
  Старые argv при редактировании остальных полей сохраняются без разбора.
- Целевые UI проверки: 15/15; SDK: 39/39; smoke Actions и locale check — passed.
  LSP новых ошибок в контрактах не нашёл; две прежние диагностики SDK test file.
- Следующий шаг: полный `scripts/verstak-check.sh`, пакеты `0.3.5-actions.3`,
  физические клики в WebKitGTK/Xvfb и передача владельцу.

## Текущая точка: 2026-10-27, Actions 2.0 — локальные сборки, ожидание проверки владельцем

- Три ветки `feat/actions-2` отправлены в origin; main не менялся, тегов и релизов нет.
  Desktop: `7bf67b8` (общая process infrastructure), `7f0dbb1` (Actions E2E).
  SDK: `a0963e5` (контракты), `5ba9a34` (generated dist из актуального src).
  Official plugins: `ee923e7` (новый `verstak.actions`).
- Core: `verstak/core/processes/v1`, literal executable/argv/cwd, wait/detached,
  bounded stdout/stderr, timeout/cancel, HTTP(S) system handler. Permission
  `process.spawn` переиспользуется, `external.openURL` добавлен. Все Go вызовы
  проверяют permission/capability и актуальный disable. Native exact-request
  approvals хранятся вне vault и не синхронизируются. Linux Wails patch исключает
  printf-интерпретацию текста, задаёт отказ по Enter и scroll для длинного review.
  Shared frontend остаётся trusted context, не sandbox. Core не содержит Actions ID/логику.
- Actions: CRUD/order, global/Deal UUID, sidebar + вкладка каждого Дела,
  palette launcher, 10 Lucide иконок, RU/EN, обе темы, output/Stop/revoke.
  Данные — существующий settings/plugin-record sync. Загрузка/import/sync/enable
  не запускают действий. Локальных path mappings нет; file associations не
  предоставляются (можно явное приложение + аргумент файла). SSHKeeper не начат.
- Local gate `scripts/verstak-check.sh`: 9/9 групп, 220/220 E2E, SDK 39/39;
  opt-in Go SKIP: real-server smoke, supplied backups, synthetic cost test.
  Focused Go `-race` passed; реальный Linux printf/no-shell, stderr/exit/cwd,
  cancellation/time/output/descendants и device-local approvals checked.
  Sync record-op replay двух временных vault checked; реальный server/два
  физических устройства не проверялись. Windows/macOS process tests — compile only.
  gopls чистый; две старые TS LSP test-file diagnostics совпали с baseline.
- Build checked: SDK, plugins, desktop, Deb и AppImage. Готовые пакеты:
  `release/verstak_0.3.5-actions.2_amd64.deb` и
  `release/verstak-linux-x86_64-0.3.5-actions.2.AppImage`.
  Бинарники: `0.3.5-actions.2 (7f0dbb1)`. В обоих 18 shipping plugins,
  checksums и Actions source match passed. SHA-256 — `build/actions-2/artifacts.json`.
- Real desktop GUI checked на development binary, извлечённом Deb и самом AppImage:
  private HOME/Xvfb/software, physical buttons, native No/Yes, literal printf,
  one-click repeat, long review, missing executable, cancelled sleep, xmessage
  GUI (возврат ~1.3s до закрытия), темы и Deal tab с Notes/Files. Четыре записи
  подтверждены на диске; аргументы отсутствуют в app.log. Личный vault и
  установленное приложение не менялись; пользовательский GPU не проверялся.
- Evidence/report: `build/actions-2/REPORT.md`, `verification.json`, `all-checks.log`,
  `native{,-deb,-appimage}/result.json`, PNG, package logs/metadata/checksums.
  GitHub CI не является доказательством комплекта: workflows берут соседей из main.
- Сборки переданы для ручной проверки. Остановиться, ждать обратной связи;
  никаких следующих этапов, merge/release или SSHKeeper без нового задания.

## Предыдущая точка: 2026-10-22, v0.3.4 опубликован; остановка по просьбе пользователя

- [Desktop v0.3.4](https://github.com/mirivlad/verstak/releases/tag/v0.3.4)
  опубликован: tag `3483033bd2cedb0c5058f0ddeacdfe2209122a15`.
  [Official plugins v0.2.8](https://github.com/mirivlad/verstak-official-plugins/releases/tag/v0.2.8):
  tag `6fbc9745f2d271b7e57fddb49d07cca49b7c6d71`. SDK `2cd620e`, docs `06a31a1`
  отправлены в main. Source feature commit desktop: `3221c42`.
- Browser Inbox принимает ссылку HTTP(S), текст или оба поля через «Добавить
  материал». Название необязательно. Глобально можно выбрать Дело по UUID
  или оставить без привязки; во вкладке используется текущее Дело. Введённое
  остаётся при ошибке, стабильный ID исключает дубликат при повторе. Новая
  запись и поступления из расширения сохраняются атомарно. Очередь остаётся
  device-local; созданные заметки и файлы синхронизируются обычным способом.
- Core добавляет только общий `settings.upsertRecord`: namespace permission,
  bounded array, полная замена по строковому ID, сохранение соседних ключей,
  sync tracking для объявленных коллекций. SDK feature-detect optional method.
  LSP definitions/references checked; gopls diagnostics пустые, JS hints only.
- Полный local `scripts/verstak-check.sh`: 9/9 групп, 210/210 E2E, без ошибок
  и пропусков. SDK 38 tests, Go race concurrent writers и bridge contracts
  прошли. Local build, Deb и AppImage v0.3.4 собраны штатными scripts.
- [CI точного релизного коммита](https://github.com/mirivlad/verstak/actions/runs/39755412117)
  прошёл, включая Windows Go и E2E. [Публикация](https://github.com/mirivlad/verstak/actions/runs/39803330089)
  завершилась успешно. Main был заморожен до завершения publish job.
- Все три опубликованных установщика полностью скачаны; SHA-256 совпали с
  SHA256SUMS и серверными digest. Deb/Windows metadata: `v0.3.4`/`3483033`;
  Windows ZIP CRC и по 17 plugin checksums прошли. Все 17 checksum manifests
  в Deb/Windows совпали с фактически опубликованными plugins v0.2.8.
- Real desktop GUI checked на опубликованном Deb: три записи на диске,
  nested Deal UUID, text-only/unassigned, вкладка Дела, конвертация в заметку,
  навигация, валидация URL, границы диалога, белый action text, обе темы.
  Опубликованный AppImage запущен обычным способом без inspector: AT-SPI
  подтвердил ручную запись на диске, отмену без записи, обе темы, собственные
  WebKit helpers, живой binary `v0.3.4`/`3483033` и 17 plugin checksums.
  Скриншоты обеих сборок просмотрены. Windows GUI/macOS не запускались.
- Evidence: `build/manual-capture-2026-10-22/RESULT.md`, `all-checks.log`,
  `{candidate,release}-ci-status.json`, `release-ci.log`, `local-packages.log`,
  `native-{complete,published-deb,published-appimage}/result.json`,
  `plugins-published/verification.json`; фактические desktop assets и metadata:
  `build/releases-2026-10-22/desktop-v0.3.4/`.
- Все шесть checkout чистые, main соответствует origin/main после fetch.
  Личный vault и установленное приложение не менялись.
- Пользователь явно попросил «завершай и останавливайся». Текущий этап
  завершён, дальнейшую разработку остановить и поставить активную цель на
  паузу после фиксации этого отчёта. Новые этапы — только по новой команде.

## Предыдущий выпуск: 2026-10-22, v0.3.3 опубликован и проверен

- [Desktop v0.3.3](https://github.com/mirivlad/verstak/releases/tag/v0.3.3)
  опубликован как стабильный latest. Tag: `8237a714a26028eff95f877d1c7380845797a61a`;
  official plugins v0.2.7: `74379f6f2640ec4191b734528e1e305827526291`.
  [CI точного коммита](https://github.com/mirivlad/verstak/actions/runs/39743121615)
  и [публикация](https://github.com/mirivlad/verstak/actions/runs/39744860938)
  завершились успешно. Финальный local gate: 9/9 групп, 206/206 E2E,
  без ошибок и пропусков. Windows Go CI также прошёл.
- Исправлена подтверждённая потеря последней правки при закрытии до draft timer
  150 ms. Core принимает Promise из component unmount, сохраняет API до flush
  и ждёт его перед новым mount того же плагина (включая разные keyed host).
  Синхронная очистка остаётся немедленной; ошибка cleanup освобождает API.
  Editor возвращает завершение существующей очереди draftWrites.
- Исправлено самопроизвольное сохранение draft при открытии Markdown для чтения:
  восстановленная правка открывается для редактирования, включая ручное
  восстановление после внешнего конфликта. Само восстановление не меняет файл.
  Новые сценарии сначала упали на старом коде, затем прошли. LSP definitions,
  references и diagnostics редактора/lifecycle checked; diagnostics пустые.
- Windows CI выявил гонку в тесте PhysicalWorkspaceMove: он видел новый tree
  до публикации event. Изменён только тест — ожидание требуемого уведомления.
  Production Go-код не менялся. 20 targeted повторов прошли; gopls check пустой.
- Все три опубликованных установщика полностью скачаны, SHA-256 совпали с
  SHA256SUMS и серверными digest. Windows ZIP CRC, Deb/Windows metadata
  `v0.3.3`/`8237a71` и по 17 наборов plugin checksum проверены. Plugins v0.2.7
  тоже скачаны в обоих форматах: SHA, server digest, CRC и 17 payloads прошли;
  удалённый annotated tag указывает на `74379f6`.
- Real desktop GUI checked на фактических опубликованных Deb и AppImage,
  private HOME/Xvfb/D-Bus и синтетический vault. Deb прошёл четыре draft-сценария
  text/Markdown × pause/rapid: Markdown переоткрывается для VIEW, восстановленная
  правка доступна для edit, JSON draft на диске, исходный файл не меняется после
  autosave interval. AppImage прошёл те же control/rapid шаги заметки из Notes UI
  без inspector; проверены metadata живого binary и 17 live plugin packages.
- Фактические Deb и AppImage также прошли Secrets regression: ссылка над
  заметкой, один master prompt по Enter, системный clipboard, изменение в том
  же окне с сохранением ID/scope, обе темы и физическое закрытие. Полный Deb gate
  проверил archive/reset, nested Deal, выбранную подпись и re-lock после OpenVault.
  AppImage подтвердил собственные WebKit helpers и отсутствие dummy credentials
  в app.log. Скриншоты восстановленного draft и окна секрета просмотрены.
- Evidence: `build/drafts-2026-10-22/{all-checks-final.log,candidate-full-ci.log,
  release-publish-ci.log,plugins-published/verification.json}` и
  `native-published-{deb,appimage}-{drafts,secrets}/result.json`;
  `build/releases-2026-10-22/desktop-v0.3.3/{verification,binary-metadata}.json`.
  Docs `76cf1c1` описывает восстановление; link check 25/25. Windows GUI/macOS
  не запускались. Личный vault и установленное приложение не менялись.
- Активная цель развития с релизами после значимых этапов продолжается. Один
  конкретный следующий срез из существующего продуктового плана: ручное
  добавление ссылки/текста в Browser Inbox. Сейчас UI принимает receiver events;
  нового кода следующего среза ещё нет. Сначала проверить его текущий storage
  и scope flow; реализация остаётся в плагине, очередь — device-local.

## Предыдущая точка: 2026-10-22, v0.3.2 опубликован и проверен

- [Desktop v0.3.2](https://github.com/mirivlad/verstak/releases/tag/v0.3.2)
  — стабильный latest release. Tag: `187f06a9360fec32981df469f0a410dd7a55be95`;
  official plugins v0.2.6: `149e7465b625ba9c3a9299e397986a5a9990d70f`.
  [CI точного коммита](https://github.com/mirivlad/verstak/actions/runs/39697402815)
  и [публикация](https://github.com/mirivlad/verstak/actions/runs/39699222618)
  завершились успешно. Локальный финальный gate: 9/9 групп, 203/203 E2E,
  без пропусков. Build-only candidate CI 39697414056 проверен до публикации.
- Обзор предлагает до трёх последних изменённых заметок с фактическими mtime.
  Каждая карточка открывает собственный путь, счётчик — список. Удалённый или
  перенесённый файл оставляет список с локализованной ошибкой. Notes владеет
  выбором/открытием через `toolRequest`; Core не получил бизнес-функций заметок.
- Исправлен подтверждённый пустой экран опубликованного AppImage v0.3.1:
  встроенный release WebKit запускал системный NetworkProcess. Полный
  NUL-terminated helper directory релокируется с сохранением размера ELF;
  AppRun задаёт cwd `usr/lib`, собственные injected bundle и GIO modules.
  Настоящий ELF/launcher regression test включён в общий gate. Sandbox
  не выключается. Механизм соответствует
  [upstream Wails](https://github.com/wailsapp/wails/blob/73242d1eca0d7d940321e1a42fdf1df6605edab9/v3/internal/commands/appimage.go#L271).
- ResourceDialog использует естественную высоту общего PluginBundleHost:
  старый встроенный WebKit иначе схлопывал фон, поля выходили за окно и мешали
  физическому закрытию. На том же runtime исправленная сборка прошла сценарий.
- Все три опубликованных установщика скачаны полностью, SHA-256 совпали с
  SHA256SUMS и серверными digest. Windows ZIP CRC, Deb/Windows build metadata
  `v0.3.2`/`187f06a` и 17 наборов plugin checksums в обоих payload проверены.
- Real desktop GUI checked на фактических опубликованных .deb и AppImage:
  синтетический vault, private HOME/D-Bus/Xvfb. Deb прошёл полный reset/archive/
  nested Deal/secret-link сценарий. Обычный AppImage без инспектора прошёл
  ввод пароля по Enter, системный clipboard логина/значения, изменение в том
  же окне с сохранением ID, повторное открытие без пароля, сохранность заметки,
  границы контролов, физическое закрытие и обе темы. Проверены metadata живого
  исполняемого файла, 17 плагинов и запуск собственных WebKit helpers.
- Фактический опубликованный AppImage также прошёл Обзор: разные пути и
  содержимое карточек, общий список, точный Overview.md, удалённый файл и обе
  темы. Сообщение об удалении дополнительно прочитано на нативном снимке.
- Evidence: `build/overview-2026-10-22/RESULT.md`, `all-checks-final.log`,
  `native-published-{deb,appimage,overview}/result.json`;
  `build/releases-2026-10-22/desktop-v0.3.2/{verification,binary-metadata}.json`.
  Windows GUI и macOS не запускались. Личный vault и установленное приложение
  не менялись. Docs `b44b77a` опубликованы; все шесть checkout чистые main и
  совпали с origin/main после fetch (перед этим proof-only коммитом).
- Цель дальнейшего развития с релизами после значимых этапов остаётся активной.
  Следующий конкретный шаг: воспроизвести поведение черновика при очень быстром
  уходе из редактора. Обнаружен warning о storage write после API disposed;
  потеря последних правок пока не подтверждена, продуктовый код не менялся.

## Предыдущая точка: 2026-10-22, v0.3.1 опубликован

- [Desktop v0.3.1](https://github.com/mirivlad/verstak/releases/tag/v0.3.1)
  опубликован как обычный latest release с .deb, AppImage, Windows ZIP и
  SHA256SUMS. Tag указывает на `a3c7512c1f5c4f89325df206e95d374f38c30b38`;
  official plugins v0.2.5 — на `e73d52e654309a1c940d01bc127de0dfdb2e08b4`.
- Точный desktop candidate прошёл [CI](https://github.com/mirivlad/verstak/actions/runs/39664851844)
  и [release workflow](https://github.com/mirivlad/verstak/actions/runs/39692921181).
  Локальный финальный `scripts/verstak-check.sh`: 9/9 групп, 201 E2E, 0
  пропусков. Серверные digest трёх установщиков совпали с SHA256SUMS.
- .deb и Windows ZIP скачаны и проверены побайтово; ZIP CRC и 17 наборов plugin
  checksum в каждом формате прошли. Build metadata обоих бинарников:
  `v0.3.1`, `a3c7512`. Payload скачанного .deb прошёл полный настоящий Wails/
  WebKitGTK сценарий секретов, включая создание во вложенном Деле сразу после
  сброса, копирование, редактирование и один unlock. Windows GUI не запускался.
- Доказательства: `build/releases-2026-10-22/all-checks-final.log`,
  `desktop-v0.3.1/verification.json`, `binary-metadata.json`,
  `native-published-deb/result.json` и CI/release logs. Полный клиентский
  AppImage позже скачан, hash совпал; обычный запуск выявил пустой экран.
  Причина и исправление v0.3.2 описаны в текущей точке выше.
- Docs `66ad7a1` описывает новый сценарий; relative-link check прошёл. Личный
  vault и установленное приложение не менялись. Активная цель продолжается.
  Следующий срез — Обзор Дела: недавние заметки и точное открытие конкретной
  заметки из карточки, с сохранением перехода общего счётчика к списку заметок.

## Предыдущая точка: 2026-10-22, подготовка релиза v0.3.1

- Владелец явно поручил выпустить готовые изменения, затем самостоятельно
  продолжать улучшение удобства, существующих функций и плагинов с релизом
  после каждого значимого этапа. Это разрешает последовательные этапы без
  прежней остановки на локальных пакетах для проверки владельцем.
- Публикуется desktop v0.3.1 с official plugins v0.2.5. Release notes и
  `OFFICIAL_PLUGINS_VERSION` согласованы; выпуск пока не объявляется завершённым.
- После публикации проверить assets, checksum и фактическое содержимое
  установщика, обновить verstak-docs. До завершения публикации не двигать main
  новым функциональным этапом: publish job берёт текущий main.
- Активная цель — дальнейшее развитие с законченными проверенными этапами и
  релизами. Успешная публикация одного выпуска не завершает всю эту цель.
- Plugins v0.2.4 опубликованы и скачаны: оба SHA-256 и серверные digest
  совпали, Windows ZIP CRC и 17 наборов plugin checksum проверены. Desktop
  candidate `b149494` прошёл CI (включая E2E и Windows Go), но release workflow
  ещё не запускался. Перед публикацией обнаружен блокирующий сценарий создания
  секрета: настоящий Wails показал после первого unlock только пустую опцию,
  после переоткрытия — `Clients` вместо Дела `Clients/Site`.
- Причины: после unlock не загружались workspace options; перечислялись
  папки корня, а не семантические Дела. Plugins v0.2.5 переводит выбор на
  `api.workspaces.list()`, загружает его при unlock и сохраняет текущее Дело
  и привязки существующих записей при недоступности списка. Старый tag v0.2.4
  не переписывается. Desktop release не запускать до проверки исправления.
- Регрессионный smoke сначала упал на прежнем коде, затем прошёл после
  исправления. Нативный сценарий дополнен созданием секрета во вложенном Деле
  сразу после сброса: `scope-fixed/native-verified/result.json` подтвердил его
  и весь прежний сценарий ссылок/копирования/редактирования. Первый новый прогон
  выявил гонку автоматического выделения текста: теперь тест дожидается полного
  текста и выделенной подписи перед нажатием кнопки. Plugin checks прошли.
  Новый полный gate и публикация ещё впереди; доказательства —
  `build/releases-2026-10-22/`.

## Предыдущая точка: 2026-10-22, архив секретов и ссылки из заметок

- Владелец поручил сначала безопасно начинать новое хранилище при забытом
  мастер-пароле, затем вставлять ссылки на секреты в заметки и открывать их
  поверх заметки: один пароль на сессию, просмотр, копирование и редактирование.
- Desktop `c752db2` — архивирование, блокировка старых Store, защита от нового
  ключа рядом с осиротевшими записями, проверка metadata, API с Go error,
  правом `secrets.write` и явным подтверждением. `f23d37a` — общий workbench
  `presentation: dialog`, оболочка, проверки и документация.
- Official plugins `0761c01` — «Забыли мастер-пароль?», подтверждение архива,
  компактное окно секрета, копирование, редактирование без потери значений
  при ошибке, Enter. `613c250` — выбор секрета Дела/общих секретов в редакторе,
  подпись из выделенного текста, открытие ссылки в диалоге. Редактор объявляет
  `secrets.read`; выбор читает только список без значений.
- SDK `51ebc53` — контракт `secrets.archiveAndReset(true)` и dialog presentation.
- Весь старый каталог сохраняется в
  `.verstak/secrets-archives/<timestamp>-<unique>/secrets`, старый пароль остаётся
  необходимым для архива. Новое хранилище получает отдельный пароль. Криптоформат
  и алгоритм получения ключа не менялись. В заметке остаётся только стабильная
  ссылка; редактирование сохраняет ID и привязку к Делу.
- Unit tests checked, `gopls` references/diagnostics checked. SDK 37/37,
  браузерный smoke редактора 27/27. Финальный `scripts/verstak-check.sh` — 9/9
  групп, 201/201 E2E, 0 пропусков. Первый полный прогон был 8/9, 200/201:
  ожидание числа карточек по secrets.read обновлено с 3 на 4, поскольку доступ
  теперь объявляет редактор. После исправления — focused 8/8 и полный 201/201.
- Real desktop GUI checked на обоих готовых пакетах: payload .deb и настоящий
  AppImage с его библиотеками. Изолированные HOME/Xvfb/private D-Bus, настоящий
  Wails bridge и системный clipboard. Проверены побайтовый архив, новое пустое
  хранилище, один запрос пароля, редактирование с сохранением scope/ID, URL с %,
  подпись ссылки, отсутствие паролей в Markdown/settings/logs, блокировка после
  переоткрытия vault и все 17 shipping-плагинов loaded. Светлая/тёмная темы
  просмотрены. Дополнительно Chromium проверил первый unlock в picker и
  последующее открытие секрета без повторного пароля.
- [Desktop CI f23d37a](https://github.com/mirivlad/verstak/actions/runs/39660687218)
  прошёл, включая E2E и Windows Go tests; plugins `613c250` и SDK `51ebc53`
  также прошли свои CI. Windows GUI/macOS не запускались.
- Локальные пакеты `v0.3.1-secrets.20261022`, оба с build commit `f23d37a`:
  `release/verstak_0.3.1-secrets.20261022_amd64.deb` и
  `release/verstak-linux-x86_64-v0.3.1-secrets.20261022.AppImage`.
  SHA-256 совпали с `release/SHA256SUMS`; 17 наборов plugin checksums проверены,
  все 87 файлов plugin payload совпадают между форматами.
- Доказательства: `build/secrets-2026-10-22/RESULT.md`, `packages.json`,
  `all-checks-verified.log`, `native-package/`, `native-appimage/`,
  `locked-picker.json`. Ранние неполные/упавшие прогоны не выдаются за зелёные.
- Изменения в трёх репозиториях отправлены. Личный vault и установленное
  приложение не менялись. Это локальная сборка для проверки владельцем;
  новый публичный release/tag не создавался. Перед будущим релизом надо
  выпустить новый набор official plugins и обновить `OFFICIAL_PLUGINS_VERSION`
  (сейчас всё ещё v0.2.3); локальные пакеты уже содержат новые плагины.

## Предыдущая точка: 2026-10-21, «Студия» v0.3.0 опубликована

- Владелец утвердил современное направление без зелёного акцента, тёмную тему,
  внедрение и явно поручил закончить с выпуском релиза. Это сняло предыдущую
  границу ожидания проверки пакетов перед следующим визуальным этапом.
- Desktop `4e7a132` — «Студия»; `858e589` — pin/notes; `7966716` — ожидание
  асинхронной смены CSS в тесте и стабилизация нативных снимков. Tag v0.3.0
  указывает на `79667166bf4e2f417009741783e7fb9dfa4e5eab`.
- [Desktop v0.3.0](https://github.com/mirivlad/verstak/releases/tag/v0.3.0)
  опубликован как обычный release: .deb, AppImage, Windows ZIP, SHA256SUMS.
- Official plugins `82b0d51`/`102e180`: общая палитра, редактор, размер иконки
  создания заметки и контраст опасных действий. Tag/release v0.2.3 указывает
  на `102e1801b411cb97658ea2da4279f918b1dddb2f`; Linux/Windows архивы опубликованы,
  их серверные digest совпали с локальными загруженными байтами.
- Docs `d830b18`: актуальная компоновка и темы; проверка ссылок прошла.
- Реализованы нейтральные палитры и индиго с белым текстом, переключатель в
  шапке, сохранение light/dark/system, настройки оформления; обновлены панель,
  вкладки и Обзор, настройки находятся внизу навигации, состав инструментов
  открывается рядом с вкладками. Узкое окно использует drawer. Modal
  портируется в body, сохраняет фокус и не восстанавливает устаревший inert.
- Core не получил бизнес-функций Notes/Files. «Документ», закрепления и общая
  навигация по материалам остаются исследовательскими предложениями.
- Проверки финального состояния: `scripts/verstak-check.sh` — 9/9 групп,
  197/197 E2E, 0 пропусков. Production frontend build и release smoke прошли.
  Точный tag прошёл [check CI](https://github.com/mirivlad/verstak/actions/runs/39493861910)
  (197 E2E без retry, Linux checks и Windows Go tests) и
  [release CI](https://github.com/mirivlad/verstak/actions/runs/39493919398).
- Real desktop GUI checked: корень/папка, UUID/metadata, старые/свежие рецепты,
  фокус/Tab/возврат фокуса, ошибки, реальное сохранение темы и reload, список
  Notes, Markdown preview и CodeMirror. Все 17 shipping-плагинов loaded.
  Пройден и обычный запуск без инспектора/JSC override. Повторно весь нативный
  сценарий прошёл на payload опубликованного .deb с build commit `7966716`.
- Локальные пакеты имели commit `858e589`; с tag отличается только код
  проверок, shipping source совпадает. Linux-пакеты распакованы, проверены
  17 наборов plugin checksum, Windows ZIP CRC и build metadata. Бинарник
  локального AppImage совпал с нативно протестированным Linux bundle.
- У опубликованных трёх установщиков серверные SHA-256 совпали с опубликованным
  SHA256SUMS. .deb и Windows ZIP дополнительно скачаны, проверены побайтово,
  их metadata указывает на `7966716`. Полная клиентская передача AppImage не
  требовалась; частичный файл имеет суффикс `.part`.
- Доказательства: `build/studio-2026-10-21/RESULT.md`, `all-checks-release.log`,
  `native-published/`, `native-plain/`, `release-smoke.log`, `packages.json`,
  `published-digest-verification.json`, `published-client-verification.json`.
  Ранние прерванные прогоны и старые неоднозначные селекторы исправлены и не
  выдаются за зелёный результат. Первый CI retry цвета заменён ожиданием CSS.
- Все изменения отправлены; перед этой записью все шесть checkout были clean
  main и совпадали с origin/main. Личный vault и установленное приложение не
  менялись. Windows GUI, macOS и установка поверх личного vault не проверялись.
- Blocker: нет. Следующий шаг — обратная связь владельца после использования
  опубликованной v0.3.0; архитектурные расширения не начинать автоматически.

## Предыдущий этап: создание Дела и предложение редизайна

- Репозитории/ветки: desktop и official-plugins, оба `main`. Перед этапом
  свежий fetch подтвердил равенство HEAD/origin/main во всех шести репозиториях;
  четыре остальных не менялись.
- Причины бага подтверждены в настоящем Wails: клик передавал
  `MouseEvent.detail=1` вместо parent UUID; второй результат `(T, string)` не
  становился reject Promise; сохранённые рецепты содержали устаревший Deal-tool
  `verstak.activity`.
- Official plugins `b5c5d01`: однократная точечная миграция рецептов,
  локализованное отображение seed-шаблонов, понятный missing-tool error,
  явное удаление недоступного инструмента в настройках. Прочие пользовательские
  поля и неизвестные ID сохранены. Push выполнен; GitHub check прошёл.
- Desktop `c786dd6`: корректный parent UUID и Go error, проверка ответа с UUID,
  открытие созданного Дела/раскрытие папки, расположение и ошибки в форме,
  начальный фокус/Tab/возврат фокуса, актуальный mock, регрессионные E2E и
  воспроизводимый `scripts/smoke-native-create-deal.sh`.
- Desktop `aa99693`: исправлено старое E2E-ожидание Wrap; `ae692c0`: Wails CLI
  закреплён по go.mod; `542eab7`: спецификация направления редизайна
  `docs/UX_REDESIGN_PROPOSAL.md`. Все отправлены в origin/main.
- Проверки: полный `scripts/verstak-check.sh` — 9/9 групп, 0 пропусков;
  Playwright 189/189; Go LSP без диагностики. Official plugins пересобраны и
  переустановлены. `scripts/build.sh` и обе Linux-упаковки прошли.
- Real desktop GUI checked на финальном бинарнике: физические клики/ввод,
  корень/папка, запись UUID и metadata, старые/свежие рецепты, ошибки и фокус.
  Отдельно создание прошло без WebKit-инспектора/JSC-настроек тестового режима.
- Локальная кандидатная версия `v0.2.11-fix.20261021`, исходный commit бинарников
  `542eab7` (последующие изменения лога не меняют код). Артефакты:
  `release/verstak_0.2.11-fix.20261021_amd64.deb` и
  `release/verstak-linux-x86_64-v0.2.11-fix.20261021.AppImage`.
  Оба распакованы, 17 shipping-плагинов и их checksum проверены; бинарник из
  AppImage побайтово совпадает с нативно протестированным. Релиз/тег не публиковался.
- Доказательства и подробный отчёт: `build/creation-fix-2026-10-21/RESULT.md`,
  `all-checks.log`, `native-final/`, `packages.json`. Первые невалидные запуски
  E2E с общим Vite/пересобираемыми bundle не выдаются за зелёную проверку.
- Редизайн пока предложение: «Рабочий стол» и «Фокус», графит/мята и светлая
  тема. Не переносить plugin-функции в Core ради макета. Закрепления и общая
  навигация по материалам потребуют отдельных контрактов.
- Blocker: технического нет; граница этапа — проверка владельцем на реальном
  vault. Windows/macOS GUI и установка поверх его версии не проверялись.
  Другие методы `(T, string)` требуют отдельного аудита.
- Следующий шаг: получить результат проверки создания Дела в переданных
  пакетах и отмашку на визуальный этап. До этого следующий этап не начинать.

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

2026-09-30. Выпущены desktop v0.2.5 (новый редактор заметок) и official plugins
v0.1.5; desktop v0.2.6 — первый релиз по пушу тега с закреплённой версией
плагинов (`OFFICIAL_PLUGINS_VERSION`, `scripts/resolve-official-plugins-ref.sh`).
Раньше тег desktop искал в репозитории плагинов тег с тем же именем, и с тех пор,
как desktop ушёл на v0.2.x, пуш тега не мог собрать релиз.

Идёт план улучшений из ревизии 2026-09-30, по пункту за релиз:

1. релизный процесс — сделано (v0.2.6);
2. гигиена: Svelte 5 / Vite 8, vitest 5, web-ext 10, `verstak-docs` — сделано
   (v0.2.7, плагины v0.2.0);
3. техдолг: `app.go` разнесён по файлам, `apiVersion` проверяется при загрузке,
   SDK сверяется с хостом тестом `sdk-surface-test` — сделано (v0.2.8).
   Отложено: монолитные `index.js` плагинов files/browser-inbox/journal/activity
   (по ~1 500 строк одного замыкания монтирования с общим состоянием) — это
   переделка состояния, а не механическое разрезание;
4. платформа: Go-тесты на Windows в CI (нашли два бага: локализация плагинов
   через ссылку/8.3-путь и переименование только по регистру на NTFS/APFS),
   проверка обновлений (выключена по умолчанию) — сделано (v0.2.9).
   Ждут решения пользователя: retention журнала операций sync (затрагивает
   сходимость клиентов, блобы и историю ревизий) и изоляция плагинов
   (iframe/worker/sidecar меняет модель монтирования);
5. продукт: Overview Дела как центр возобновления работы, синхронизация
   секретов, ручной захват в Browser Inbox.

Push в `mirivlad/verstak`: у OAuth-токена `gh` нет scope `workflow`, поэтому
коммиты, трогающие `.github/workflows/`, пушатся по SSH
(`git push git@github.com:mirivlad/verstak.git`).

## Что ещё не пройдено полностью

Приоритеты уточнены по факту осмотра 2026-08-18.

**Повышено:**

- продуктовый UX. Из 11 desktop PR ревизии пользователь увидел два (#2, #3). Остальное — тесты, контракты, CI. В `docs/UX_UI_REFACTOR_PLAN.md` не доделаны пункты 4 (эргономика Plugin Manager) и 5 (Files/Workbench: контекстное меню, клавиатура, выделение);
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
