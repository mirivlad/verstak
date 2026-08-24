# Projects MVP UX v2 — implementation task

Status: **ready for implementation**  
Target: Verstak desktop + `verstak.projects` official plugin  
Baseline: v0.1.4

## 1. Goal

Rework the current Projects MVP UI into a native Verstak experience without changing the successful capability-driven architecture introduced in v0.1.4.

This is not a feature-expansion task and not a Jira/Trello clone. The purpose of this pass is to make the existing Projects MVP pleasant, obvious, compact, and consistent with the rest of Verstak.

The current data model and provider-independent integrations should remain unless a small compatibility change is required for correct Deal linking.

## 2. Problems observed in v0.1.4

The first Projects MVP proved the data model and capability bridge, but the presentation layer is not acceptable as the final UX.

### 2.1 Native/browser-looking controls

The project editor uses controls that visually fall out of the Verstak theme, especially status and priority selects. On a dark Verstak window they appear as foreign light/native widgets.

Required result:

- no browser/OS-default looking form controls in Projects;
- status, priority, tree pickers, menus, inputs and focus states must use existing Verstak visual patterns/tokens;
- do not invent an isolated Projects-only design system when an existing shell/plugin pattern can be reused.

### 2.2 Linked Deal is exposed as an internal path

The editor currently asks the user for `Путь связанного дела` / linked workspace root path.

This is the wrong abstraction. A user works with Deals, not internal filesystem/root-path conventions. It is also error-prone: entering an absolute path or a path relative to the vault does not reliably establish the expected link.

Required result:

- remove the raw path text field from the user-facing form;
- replace it with a proper Deal picker;
- show the same logical hierarchy the user recognizes from the left sidebar;
- the user chooses a Deal by name/location, e.g. `Projects / Creatures2.0`;
- internal `workspaceRootPath`, stable id, or another canonical identity is resolved and stored by the application/plugin, not typed by the user;
- include an explicit `Не связывать с Делом` / `No linked Deal` option;
- when Projects is opened inside a Deal and a new project is created, preselect the current Deal unless the user changes it.

### 2.3 Deal picker architecture

Do not solve the picker by scraping the sidebar DOM, parsing visible labels, or walking the vault directly from Projects.

Preferred contract:

1. use an existing public SDK/Core workspace/Deal enumeration API if it can return all Deals visible to the user;
2. preserve hierarchy information needed to render the tree;
3. resolve selection to the canonical identity/path in Core/SDK;
4. keep Projects dependent on public APIs only.

If the current `api.workspaces.list()` contract only returns workspaces where the calling plugin is already active and therefore cannot represent all user-visible Deals, introduce the smallest read-only SDK/Core addition required to enumerate/select Deals correctly (for example an all-visible-workspaces/tree query). Do not weaken plugin boundaries or give Projects a privileged filesystem path.

A Projects project linked to a Deal that does not itself contain the Projects workspace tool must still be representable as a link if the product model allows that Deal to exist in the sidebar.

### 2.4 Main Projects screen wastes space and overuses cards

The current screen is visually dominated by nested bordered cards:

- project list is a small floating card on the left;
- the detail area leaves large unused regions;
- Status / Milestones / Tasks / Notes are presented as four equal KPI cards although they are not equal concepts;
- unavailable integrations consume large cards that only say `Недоступен`;
- linked Deal path and history are additional full-width cards.

This creates many rectangles but little information hierarchy.

Required result: redesign the page as a real **master-detail** workspace.

## 3. Target layout

### 3.1 Master-detail structure

Use a stable two-pane layout on normal desktop widths.

Left pane:

- dedicated project list, approximately 280–340 px or equivalent responsive sizing;
- local project search/filter;
- clear `Новый проект` / `New project` action rather than a visually isolated ambiguous control where space permits;
- rows optimized for scanning rather than mini dashboards.

Right pane:

- selected project header;
- project content tabs;
- content fills available width without a card-inside-card dashboard shell.

The list pane should feel like part of Projects, not a floating card embedded inside a mostly empty canvas.

### 3.2 Project row

Each project row should prioritize:

1. project name;
2. compact status indication;
3. optional priority/tag signal only when useful.

Do not print a raw workspace path inline beside status/priority. If a linked Deal is useful in the row, show a human-readable secondary label such as `Projects / Creatures2.0` with restrained styling.

Selected, hover, keyboard-focus and disabled states must match the rest of Verstak.

### 3.3 Project header

The detail header should contain:

- project name as the primary title;
- description immediately below when present;
- compact status badge;
- compact priority badge when meaningful;
- tags;
- human-readable linked Deal (if any), preferably as a navigable/context action rather than a raw data field;
- Edit action in a predictable location.

Avoid duplicating the same information again in large Overview cards.

### 3.4 Tabs

Keep the current conceptual tabs:

- Overview;
- Milestones;
- Tasks;
- Notes;
- Files;
- Activity;
- Links.

Improve their visual hierarchy and active/focus states using existing Verstak tab patterns.

Optional capability absence must not break navigation.

## 4. Overview redesign

Overview should become a compact project summary, not a KPI card grid.

Suggested information order:

1. project description/context if not already fully visible in the header;
2. milestone progress and nearest/open milestones;
3. Tasks summary when Todo is available;
4. Notes summary/recent notes when Notes is available;
5. recent meaningful project activity;
6. useful links or linked Deal context where appropriate.

The exact visual arrangement can follow existing Verstak UX conventions, but must preserve information hierarchy and density.

### 4.1 Status

Status is a project property, not a peer dashboard metric next to Tasks and Notes. Show it in the project header or compact metadata area.

### 4.2 Milestones

On Overview, show useful progress, for example:

- `3 / 5` completed;
- progress indicator if consistent with Verstak;
- next/open milestones.

Do not dedicate a large empty card just to `Этапы: 0/0` when there is no useful content.

### 4.3 Optional integrations

When an optional capability is unavailable:

- do **not** render a large Overview card saying only `Недоступен`;
- omit that Overview summary block, or use a very small contextual indicator only if needed;
- if the user explicitly opens the corresponding tab, show a proper local empty/degraded state explaining what is unavailable and which plugin/capability would provide it;
- Projects itself remains fully usable.

Preserve the existing lifecycle rule: missing optional capability is degraded behavior, not plugin failure.

## 5. Project editor redesign

Keep create/edit flows concise and native to Verstak.

Fields:

- Name;
- Description;
- Status;
- Priority;
- Tags;
- Linked Deal picker.

Requirements:

- all controls use Verstak styling;
- labels, spacing, keyboard focus and error states are consistent;
- no raw internal paths exposed to the user;
- Deal picker supports keyboard and mouse operation;
- selection is readable after the picker closes (`Projects / Creatures2.0`, not an opaque id);
- long Deal names and deep hierarchies must remain usable;
- editing an existing project must resolve and display the currently linked Deal correctly.

## 6. Deal tree picker UX

The picker should be recognizably equivalent to the Deal hierarchy already shown in the sidebar.

Example:

```text
Связанное дело
[ Projects / Creatures2.0                 ▾ ]

  Не связывать с Делом
  ─────────────────────────────────────────
  Поиск дела…
  ─────────────────────────────────────────
  ▾ Projects
      AI-server
      Creatures2.0                         ✓
      LearnManagementS…
      RetroESP
      ServerMon
      Sshkeeper
      Synthesis
      Верстак
      Домовой
  ▸ Импортирова…
  ▸ Литература
```

This is illustrative, not a pixel-exact mandate.

Required behavior:

- collapsible hierarchy when useful;
- search filters Deals while preserving enough parent context to understand location;
- current selection clearly marked;
- Enter selects; Escape closes; arrows provide reasonable navigation if the common Verstak picker pattern supports it;
- no filesystem terminology in user-facing labels;
- picker works for nested Deals, not only one level.

## 7. History / recent activity

Current entries such as repeated `Проект изменён · ЛЛМ` are low-value.

Make project-owned history meaningful.

Examples:

- `Приоритет: Обычный → Высокий`;
- `Статус: Активный → Приостановлен`;
- `Связано с Делом Projects / Creatures2.0`;
- `Добавлен этап: Подготовить прототип`;
- `Этап завершён: Архитектура`;
- `Добавлена ссылка: Репозиторий`.

Requirements:

- do not log meaningless duplicate edit events when no user-visible field changed;
- retain a lightweight bounded history rather than turning Projects into an audit subsystem;
- existing stored old-format events must not crash rendering.

## 8. Search clarity

The shell already has global search. Projects also needs a local list filter.

Make the distinction visually and semantically clear:

- global search remains shell-owned;
- project-list search filters Projects only;
- placeholder/placement should make scope obvious;
- do not add another global search implementation inside the plugin.

## 9. Responsive behavior

Desktop is the primary target, but the redesigned layout must not collapse badly at narrower widths.

At minimum:

- no horizontal page overflow caused by project list, tabs or picker;
- master-detail can stack or switch modes at narrow widths;
- tabs remain usable;
- dialogs/pickers fit the viewport;
- project list remains navigable.

Follow the repository's existing responsive/UI verification conventions.

## 10. Architecture constraints

Preserve the v0.1.4 architecture:

- Projects owns only project metadata, milestones, links and lightweight project history;
- Notes remain owned by Notes;
- Tasks remain owned by Todo;
- Files remain owned by Files;
- Activity remains owned by Activity;
- Projects consumes integrations through capabilities/contributions, not hardcoded provider plugin IDs;
- do not reintroduce `commands.executeFor(targetPluginId, ...)` coupling;
- optional capability absence stays a degraded UI state;
- disabling a provider must not delete Projects data;
- do not move project business logic into the desktop shell for visual convenience.

Core/SDK changes are allowed only when required to expose a clean generic workspace/Deal selection contract that belongs at platform level.

## 11. Data compatibility

Existing v0.1.4 Projects data must remain readable.

If linked Deal representation changes:

- migrate/normalize existing `workspaceRootPath` records safely;
- preserve unresolved legacy values rather than silently dropping them;
- show an understandable `linked Deal unavailable` state when an old target no longer exists;
- save back the new canonical representation only when safe;
- add a migration/normalization test.

Do not require users to recreate projects created in v0.1.4.

## 12. Implementation locations

Primary implementation is expected in:

- `mirivlad/verstak-official-plugins/plugins/projects/`

Likely supporting changes/tests may be needed in:

- `mirivlad/verstak-sdk` for a generic read-only Deal/workspace enumeration API, **only if the current public API is insufficient**;
- `mirivlad/verstak` Core/Wails bridge for that generic SDK contract;
- desktop E2E fixtures/contracts when public APIs or visible tab/layout behavior change.

Do not create a Projects-specific privileged Core API if a generic workspace API is the correct abstraction.

## 13. Testing requirements

Add/update tests for at least these flows.

### 13.1 Projects plugin smoke

- mount with all optional providers available;
- mount with Notes/Todo/Files/Activity unavailable;
- create and edit a project;
- select, change and clear a linked Deal through the picker contract;
- existing v0.1.4 linked-path project still opens;
- milestone changes persist;
- meaningful project history is generated;
- Notes/Todo still use `api.capabilities.invoke(...)`.

### 13.2 Deal picker contract

- nested Deal hierarchy;
- duplicate leaf names under different parents are distinguishable;
- search preserves parent context;
- unresolved/deleted linked Deal;
- keyboard close/select behavior;
- no raw path input is required.

### 13.3 Desktop E2E

Cover the real user path:

1. open Projects;
2. create a project;
3. choose `Projects / <nested Deal>` from the tree picker;
4. save;
5. verify human-readable linked Deal in project detail;
6. reopen Edit and verify the same Deal is selected;
7. switch project and return;
8. verify link persists;
9. verify Project Deal tab ordering still works;
10. verify no browser-default/light select appears in the dark UI through visual/manual evidence.

### 13.4 Regression gates

Run the relevant repository checks, including:

- official plugins `scripts/check.sh` and Projects smoke;
- desktop static/Go contracts when desktop/Core changes;
- full Playwright suite when shell/workspace APIs or canonical fixtures change;
- visual audit;
- real Wails/WebKitGTK GUI audit for the final integrated branch.

## 14. Manual visual review checklist

Before declaring the task complete, inspect screenshots at normal desktop size and at least one narrower width.

Reject the implementation if any of these remain:

- bright OS/browser-default select controls on the dark theme;
- raw `workspaceRootPath` field visible to the user;
- linked Deal must be guessed/typed manually;
- large Overview cards whose only content is `Недоступен`;
- excessive nested borders/card-on-card composition;
- large unused detail-area whitespace caused by the project list being a small floating card;
- project rows containing compressed technical metadata/path noise;
- repeated history entries that only say `Проект изменён` without explaining the change;
- broken layout on narrow windows.

## 15. Acceptance criteria

The task is complete only when all of the following are true:

1. Projects visually belongs to Verstak; controls do not look native/browser-default.
2. No user needs to know or type a workspace filesystem/root path.
3. Linked Deal selection uses a searchable hierarchical picker based on a public platform contract.
4. A nested Deal can be selected, persisted, reopened and changed reliably.
5. Existing v0.1.4 project data remains usable.
6. Projects uses a proper master-detail layout with substantially less wasted space and card clutter.
7. Overview prioritizes useful project information instead of four equal KPI cards.
8. Missing optional providers do not occupy large `Unavailable` Overview cards and do not fail Projects.
9. Project history describes meaningful changes.
10. Capability/provider independence introduced in v0.1.4 is preserved.
11. Relevant plugin, desktop, E2E, visual and real-GUI checks are green.
12. Final screenshots are manually reviewed rather than relying on CI alone.

## 16. Non-goals for this task

Do **not** expand scope into a full project-management suite.

Explicitly out of scope unless required by the UX rewrite itself:

- task dependency graphs;
- Gantt charts;
- resource planning;
- multi-user roles/permissions;
- advanced reports/analytics;
- arbitrary workflow builders;
- Jira/Linear/Trello parity;
- replacing Todo, Notes, Files or Activity with Projects-owned copies;
- redesigning the whole Verstak shell.

If an attractive idea does not directly fix the current Projects MVP UX or Deal linking, leave it for a later Projects feature stage.

## 17. Delivery

Prefer a focused set of PRs with green gates rather than one mixed architecture/UI dump if SDK/Core support is required.

Suggested split when a platform API change is necessary:

1. generic SDK/Core Deal/workspace enumeration contract + tests;
2. Projects UX v2 implementation + plugin tests;
3. desktop E2E/fixture integration if not naturally included with step 1.

If the existing public API is already sufficient, keep the work plugin-local plus the required integration tests.

The final result should be judged by the actual interactive UI, not merely by the fact that the Projects data model and capabilities continue to function.