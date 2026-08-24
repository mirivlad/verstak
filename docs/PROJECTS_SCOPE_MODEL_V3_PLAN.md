# Projects scope model v3 — implementation plan

Status: implementation plan for v0.1.5

## 1. Product model

Projects are not peer workspaces and must not behave like a second Deal.

Canonical hierarchy:

```text
Verstak
  Deal
    Project
      project-scoped resources
```

A **Deal** owns the complete working context. A **Project** is a bounded stream of work inside one Deal.

The global **Projects** sidebar entry is a portfolio/index across all Deals. It shows every project and its current state. Selecting a project must switch Verstak to the Deal that owns/contains that project and open that project there.

Inside a project, Tasks/Notes/Files/Activity views show only resources scoped to that project. The normal Deal-level Tasks/Notes/Files/Activity views continue to show all resources for the Deal, including project-scoped and unscoped resources.

## 2. UX contract

### 2.1 Global Projects view

The global Projects entry is a portfolio, not a project editor.

Each project is represented as a compact card with only useful portfolio information:

- project name;
- owning/linked Deal;
- state;
- priority when meaningful;
- progress derived from milestones/tasks when available;
- bounded next-step information when available.

The portfolio must avoid duplicating the Deal workspace UI. No nested master-detail editor, no internal Tasks/Notes/Files tabs in the global portfolio.

Selecting a card:

1. resolves the project's stable `workspaceId` / Deal UUID;
2. switches the active Deal;
3. opens the Projects contribution in that Deal;
4. selects the requested project by stable project UUID.

If the Deal was deleted or cannot be resolved, the card remains visible as an unresolved project and offers repair/edit rather than silently losing the relationship.

### 2.2 Project view inside a Deal

The project view is a scoped lens over the Deal.

It may contain:

- Overview;
- Tasks;
- Notes;
- Files;
- Milestones;
- Activity;
- Links.

Only project-owned metadata, milestones, links and project history are stored by Projects itself. Tasks, Notes, Files and Activity remain owned by their provider plugins.

The project Overview must not duplicate full Tasks/Notes/Files lists. It should be compact and answer: state, progress, next milestone/next action, and recent meaningful changes.

### 2.3 Deal-level provider views

Deal-level provider views remain complete:

```text
Deal Tasks   = all tasks where workspaceId == current Deal
Deal Notes   = all notes where workspaceId == current Deal
Deal Files   = all files where workspaceId == current Deal
```

Project views add an optional project scope:

```text
Project Tasks = workspaceId == current Deal && projectId == current Project
Project Notes = workspaceId == current Deal && projectId == current Project
Project Files = workspaceId == current Deal && projectId == current Project
```

Unscoped resources have no `projectId` and remain visible at Deal level only.

## 3. Data and API model

### 3.1 Stable identities

Projects must store and navigate by stable IDs:

```text
project.id       = stable project UUID
project.workspaceId = stable Deal UUID
```

`workspaceRootPath` remains legacy migration data only. It must not be the canonical relationship and must not be shown as a technical path in normal UI.

### 3.2 Resource scope

Provider-owned resources gain an optional project scope:

```text
workspaceId: string
projectId?: string
```

`projectId` is optional and MUST NOT imply separate storage owned by Projects.

Providers must be able to:

- create a resource with `{ workspaceId, projectId }`;
- list all Deal resources by `workspaceId`;
- list project resources by `{ workspaceId, projectId }`;
- preserve project scope through edit/move/update operations;
- degrade safely when the Projects plugin is disabled or a project is deleted.

### 3.3 Navigation contract

Projects needs a public, capability-safe way to request navigation to:

```text
Deal UUID + contribution/plugin view + project UUID
```

Do not use plugin-to-plugin private command dispatch. Core owns workspace selection and view navigation; Projects supplies the target IDs.

The selected project may be passed as transient navigation state and then persisted by Projects as its current selection for that Deal.

## 4. Ownership boundaries

### Projects owns

- project identity;
- Deal relationship (`workspaceId`);
- project status/priority/tags/description;
- milestones;
- project links;
- meaningful project history;
- portfolio projection.

### Provider plugins own

- Tasks owns tasks;
- Notes owns notes;
- Files owns files;
- Activity owns activity records.

Projects must consume public provider capabilities only. It must not copy provider data into its own project record and must not directly execute another plugin's private commands.

## 5. Migration

Existing v0.1.4/v0.2 project records must be preserved.

1. If a record already has a stable `workspaceId`, keep it.
2. If only `workspaceRootPath` exists, resolve it through the Deal tree and write the stable UUID when resolution is unique.
3. If resolution is ambiguous or impossible, preserve the legacy value and mark the relationship unresolved.
4. Existing project metadata, milestones, links and history remain intact.
5. Existing provider resources remain unscoped (`projectId` absent) unless the user explicitly associates them with a project; do not guess project membership from names or paths.

## 6. Implementation sequence

### Phase A — contract and navigation

- add/extend SDK/Core navigation API for `workspaceId + target view + state`;
- keep `workspaces.tree()` as the read-only complete Deal tree;
- add automated bridge tests proving navigation is available to normal plugin API consumers;
- no shell-component-only API shortcuts.

### Phase B — Projects portfolio

- replace the current global Projects master-detail layout with a compact all-project card portfolio;
- cards show Deal/state/priority/progress/next step without dashboard noise;
- card click navigates to owning Deal and opens that project;
- retain create/edit/repair flows and hierarchical Deal picker;
- fix Deal picker clipping in modal.

### Phase C — project-in-Deal view

- render Projects as a Deal contribution;
- when opened with a project navigation target, select that project;
- keep a compact project header and scoped tabs;
- Overview becomes summary-only, not a duplicate of provider tabs.

### Phase D — provider project scope

For Tasks, Notes and Files first; Activity follows the same contract.

- add optional `projectId` to provider records/APIs where required;
- support list/create/update by `{workspaceId, projectId?}`;
- Deal views do not filter out project resources;
- project views request the additional filter;
- provider absence yields a compact unavailable state, not a large placeholder card.

### Phase E — tests and visual gates

Automated coverage must include:

- portfolio contains projects from multiple Deals;
- card click changes active Deal and opens the correct project;
- duplicate Deal names still resolve by UUID;
- deleted/unresolved Deal relationship remains visible and repairable;
- project Tasks/Notes/Files show only matching `projectId`;
- Deal Tasks/Notes/Files show both scoped and unscoped resources;
- resources from another Deal never leak into either view;
- legacy `workspaceRootPath` migration;
- keyboard Deal picker and nested Deals;
- narrow layout;
- no browser-default bright selects;
- picker results are visibly rendered, not merely present in DOM.

Visual/manual gates:

- portfolio must read visually as a portfolio, not another Deal;
- project-in-Deal must read as a scoped lens, not a nested workspace;
- Overview must not repeat whole provider lists;
- real Wails/WebKitGTK GUI smoke remains green.

## 7. Release gate for v0.1.5

Do not publish v0.1.5 until:

- SDK/Core/provider contracts are coherent and backward-safe;
- all repository checks pass;
- targeted Playwright E2E passes;
- visual audit shows the portfolio and project-scoped view correctly;
- Deal picker clipping is fixed and manually verified from screenshot evidence;
- GUI audit passes;
- release notes describe the new Deal → Project → scoped resource model.

## 8. Non-goals for v0.1.5

- no automatic inference of project membership for old Tasks/Notes/Files;
- no separate project filesystem/database owned by Projects;
- no duplication of provider records into Projects storage;
- no independent Project-as-workspace hierarchy;
- no hidden coupling to provider plugin IDs beyond public capabilities/contributions.
