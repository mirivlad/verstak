# DokuWiki and Obsidian Import Plugin Design

Date: 2026-07-23

## Goal

Add one official `verstak.import` plugin that imports user content from a
DokuWiki installation or an Obsidian vault selected as a directory, ZIP, TAR,
TAR.GZ, or TGZ archive. The plugin must analyze the source without changing the
vault, propose an editable Verstak folder/Deal/note/file structure, and apply
only the user-approved plan under a new isolated import directory.

The delivered feature includes compatible SDK, desktop, official-plugin, and
documentation changes; focused and end-to-end verification with the supplied
local backup archives; commits and pushes after significant completed blocks;
and published GitHub release artifacts produced by the repositories' release
scripts.

## Product Decisions

- There is one Plugin Manager entry named `Import`, implemented by
  `verstak.import`, with DokuWiki and Obsidian adapters inside it.
- The settings contribution opens a wide four-step modal wizard using the
  existing Verstak visual language, spacing, controls, colors, responsive
  behavior, and Lucide icon set.
- The importer proposes an adaptive structure. Organizational branches become
  folders, coherent bodies of work become Deals, ordinary documents become
  notes, and cohesive mixed project trees remain files. Every proposed node
  except the fixed import containers can be changed to folder, Deal, note,
  file, or skip before applying the plan.
- Every run is isolated under:

  ```text
  Импортировано/
    DokuWiki — YYYY-MM-DD HH-mm-ss/
    Obsidian — YYYY-MM-DD HH-mm-ss/
  ```

  A same-second collision receives a stable ` (2)`, ` (3)`, and so on suffix.
  Existing user content is never overwritten or merged automatically.
- If `Импортировано` already exists as a compatible organizational folder, it
  is reused. A conflicting file, unmarked directory, or Deal at that path is a
  blocking conflict rather than a reason to rename or overwrite user data.
- DokuWiki imports only current page versions. Old revisions are neither
  imported nor mentioned in the result.
- DokuWiki syntax is converted to Markdown. Unsupported plugin syntax remains
  visible in the converted page and produces a page warning; it is never
  silently discarded.
- Obsidian's `.obsidian/` tree is ignored silently. Only user content is
  analyzed and imported.
- Credentials, recovery codes, passwords, and similar content are processed as
  ordinary pages. The importer does not classify, modify, block, delete, or
  move them to Secrets. The confirmation step displays one general warning so
  the user can review imported material afterward.
- Nested archives found inside an imported source are ordinary files and are
  not recursively unpacked.

## Architecture Boundary

The implementation follows the platform's plugin boundary:

- `verstak-sdk` defines generic import capability, permissions, DTOs, and the
  frontend `api.imports` contract.
- `verstak-desktop` implements a permission-scoped external-source session,
  safe directory/archive access, generic plan validation, staging, and physical
  plan application.
- `verstak-official-plugins` owns source detection, DokuWiki conversion,
  Obsidian link handling, graph analysis, classification, the editable plan,
  and all importer UI.
- `verstak-docs` documents the platform contract and the official plugin.

Desktop core must not contain DokuWiki or Obsidian names, parsing rules,
classification heuristics, or UI copy. The plugin must not call Wails bindings
directly or receive unrestricted filesystem access.

## Capabilities and Permissions

Desktop registers the core capability:

```text
verstak/core/import/v1
```

The plugin provides:

```text
verstak/import/v1
```

The SDK adds two permissions:

- `imports.readExternal`: open a user-selected external directory or supported
  archive and read entries through an opaque source session. This is dangerous
  and visible in Plugin Manager.
- `imports.apply`: stage and publish an approved import plan inside the current
  vault. This is dangerous and visible in Plugin Manager.

The plugin also declares `ui.register` and `storage.namespace`. It contributes
one settings panel. The planned generic `importers` contribution point is not a
prerequisite for this first UI because the requested entry point is the plugin's
Settings button.

## Generic Import API

The frontend API exposes an `imports` namespace scoped to the calling plugin:

```text
imports.selectDirectory()
imports.selectArchive()
imports.listEntries(sourceHandle, cursor)
imports.readText(sourceHandle, entryID)
imports.applyPlan(sourceHandle, plan)
imports.closeSource(sourceHandle)
```

`selectDirectory()` and `selectArchive()` use native dialogs. Archive selection
accepts `.zip`, `.tar`, `.tar.gz`, and `.tgz`. A successful selection creates a
source session and returns:

- opaque `sourceHandle`;
- source kind (`directory` or `archive`);
- display path and basename;
- stable source fingerprint;
- entry and byte summaries available after indexing.

`listEntries()` is a paginated generic entry listing. It returns normalized
entry IDs, relative paths, kind, declared size, modified time, and media hint.
It never returns an absolute path usable by the plugin. The plugin performs
format and candidate-root detection from these entries.

`readText()` returns UTF-8 text only for a bounded regular entry. It rejects
binary data, oversized text, links, and an entry outside the source session.
Binary and unchanged file payloads stay behind the source handle. An approved
plan refers to their entry IDs so desktop can stream them directly into staging
without routing the bytes through the WebView.

`closeSource()` is idempotent. Desktop also closes sessions when the settings
host unmounts, the plugin is disabled/reloaded, the vault closes, or the session
expires.

## Source Safety

Directory sessions are rooted at the exact selected directory. They do not
follow symbolic links. Paths are canonicalized before a session is created and
again before reads.

Archive sessions index entries without extracting them. ZIP/TAR handling
rejects:

- absolute paths and drive-qualified paths;
- `..` traversal after normalization;
- NUL and device paths;
- symbolic links, hard links, and non-regular special entries;
- duplicate normalized paths, including case-folded duplicates that would
  collide on common target filesystems.

The first implementation uses fixed platform limits rather than user-facing
configuration:

- at most 250,000 source entries;
- at most 20 GiB declared uncompressed content;
- at most 2 GiB for one copied entry;
- at most 16 MiB for one text entry delivered to plugin JavaScript;
- a maximum 1,000:1 declared expansion ratio for compressed archive entries;
- a free-space check before staging begins.

Limit errors are fatal, explain the violated limit, and make no vault changes.
The supplied DokuWiki and Obsidian backups must pass these limits.

## Source Session and Fingerprint

A source session belongs to one plugin, one open vault, and one selected source.
Every call checks plugin status, permissions, session ownership, and expiry.

The source fingerprint covers the normalized entry inventory and stable source
metadata. `applyPlan()` verifies it immediately before staging. If a selected
directory or archive changed after analysis, the import stops and requires a
new analysis. This prevents a reviewed plan from applying to different bytes.

## Plugin Analysis Model

The plugin builds one format-neutral graph:

- source nodes for directories, pages, and files;
- source entry IDs and original relative paths;
- titles, modified times, media types, and content roles;
- edges for page links, heading/block references, embeds, and attachments;
- proposed target nodes and confidence/reason metadata;
- warnings attached to the exact source and target node.

The graph is held in the mounted import session. Plugin storage retains only the
last summary and non-content diagnostics, not full page text or file bytes.

The planner first chooses candidate organizational boundaries, then proposes
Deals and content roles. Its output is deterministic for the same source
fingerprint. User edits update the target mapping and rerun link resolution
without rescanning the source.

## DokuWiki Adapter

### Detection

The adapter searches selected entries for a DokuWiki user-data layout. It
accepts a full installation containing `data/pages/`, a backup containing
`pages/` with an optional sibling `media/`, or the selected data directory
itself. Wrapper directories in an archive are allowed. If multiple valid roots
exist, the source step presents them for explicit selection.

Only current `pages/**/*.txt` and referenced `media/**` regular files enter the
content graph. `attic`, `media_attic`, `cache`, `index`, `locks`, `tmp`, PHP
application code, configuration, authentication files, and generated metadata
are ignored silently.

Bundled DokuWiki help pages are excluded only when their path and content match
known stock documentation fingerprints. A customized or unrelated `wiki`
namespace is treated as user content.

### Identity and Titles

The page ID is derived from its path relative to `pages/`. A principal heading
becomes the proposed display title; otherwise the final page-ID segment is used.
Namespace and page IDs remain stable graph keys even when display names change.

Within a proposed Deal, namespace `start.txt` becomes `Notes/Overview.md`.
Other pages receive safe human-readable Markdown filenames derived through the
same title/filename policy as Verstak Notes. Every rename is recorded before
links are rewritten.

### Markdown Conversion

The converter handles at minimum:

- DokuWiki heading levels;
- bold, italic, underline, monospace, deleted, subscript, and superscript text;
- paragraphs, forced breaks, horizontal rules, and quotes;
- ordered, unordered, and nested lists;
- tables and alignment where Markdown can represent it;
- inline code, indented code, `<code>`, `<file>`, and nowiki regions;
- footnotes;
- external links, page links, labels, and section anchors;
- media links, embeds, captions, and link-only media.

Internal `[[namespace:page|label]]` and media `{{namespace:file}}` references are
resolved against the DokuWiki graph, then emitted as relative Markdown links to
the final target. Known interwiki targets may become ordinary URLs; unknown
interwiki and plugin constructs remain visible and generate warnings.

Unresolved and ambiguous references are not deleted. They remain readable in
the resulting Markdown and appear in the import report.

Only current pages are imported. Revision history is not represented in the
plan or final report.

## Obsidian Adapter

### Detection and Scope

The adapter recognizes a directory containing `.obsidian/` or a coherent
Markdown vault graph when settings were omitted from the backup. Wrapper
directories are allowed, and multiple candidate vaults require explicit user
selection.

`.obsidian/` is removed from the candidate graph before planning and is not
shown as skipped content. Everything else is user content unless rejected by
generic source safety rules.

### Markdown and Links

Markdown content is preserved except for path-dependent references that must be
rewritten after target mapping. YAML frontmatter, tags, tasks, callouts, code
blocks, headings, and block IDs remain intact.

The graph resolves:

- `[[target]]` and `[[target|alias]]`;
- heading and block references;
- `![[embed]]` attachments and note embeds;
- ordinary Markdown links and images;
- relative attachment paths.

Resolution follows Obsidian's path and basename behavior. A reference with
multiple equally valid targets remains visible and receives a warning rather
than selecting an arbitrary file. Ordinary wikilinks and heading references
become relative Markdown links. Image/file embeds become Markdown image or file
links. A note embed that cannot be represented as Markdown transclusion becomes
an ordinary link with a warning. For a valid block reference, the converter
adds a stable HTML anchor at the original block and links to that anchor; an
unresolved block reference remains visible and receives a warning.

### Content Classification

Ordinary Markdown-centric vault branches are proposed as notes. A branch with a
cohesive mixture of source code, build/project files, Markdown documentation,
and assets is proposed as a file subtree so the original project structure and
relative relationships are not split across `Notes/` and `Files/`. The user can
override every proposal.

Images, PDFs, source files, documents, and nested archives are copied unchanged.
Nested archives are never opened recursively.

## Editable Import Plan

The plan schema is versioned and format-neutral. It contains:

- source handle and fingerprint;
- fixed run-container name;
- target nodes with stable IDs and parent IDs;
- node kind: organizational folder, Deal, note, file, or skipped;
- proposed and user-approved names;
- default template ID for each Deal;
- converted text for notes or a source entry ID for copied payloads;
- original modified time;
- source-to-target mapping and warnings.

The fixed `Импортировано` and run containers cannot be removed, retargeted, or
merged with existing Deals. Other nodes are editable. Changing a node type
revalidates its descendants and link destinations immediately.

Before apply, the plugin ensures that every content node belongs to a proposed
Deal. Notes map below `Notes/`; files map below `Files/`. The plan UI blocks
confirmation only for invalid topology, duplicate final paths, missing source
entries, or another fatal validation error. Low confidence and conversion
warnings remain reviewable but non-blocking.

## Generic Plan Application

Desktop treats the submitted plan as untrusted input. It repeats permission,
session, fingerprint, schema, topology, path, reserved-name, and case-folded
uniqueness checks. Every target must resolve below the current run container.

Each proposed Deal is created with the existing `default` workspace template;
the importer does not hard-code another plugin's ID. Converted text is written
as UTF-8 Markdown. Unchanged payloads stream from the selected source entry.
Where the target filesystem supports it, source modified times are preserved.

Workspace marker files live inside each staged Deal, while the current desktop
also keeps UUID-keyed workspace-tool metadata under the vault's global
`.verstak/workspaces/` registry. The import applier must not call the existing
single-workspace helper in a way that publishes this registry metadata during
staging. It prepares registry files under transaction-only names and records
the exact final paths in an import transaction journal.

Application proceeds as follows:

1. Resolve a unique run-directory name and verify the `Импортировано` parent.
2. Build the complete folder, Deal marker, template, note, and file tree under
   `.verstak/import-staging/<session-id>` and prepare UUID registry metadata
   under transaction-only filenames.
3. Verify written counts, sizes, marker files, registry files, and final paths;
   then persist the transaction journal.
4. Publish the complete run directory with a final same-filesystem rename. If
   the `Импортировано` folder does not exist, publish a staged compatible
   organizational folder containing the run instead.
5. Promote every prepared UUID registry file to its final name, mark the
   transaction committed, refresh workspace-tree/file-watcher baselines, and
   publish the existing tree/file change signals.

The user may cancel through staging. The short final publish phase is
non-cancellable. Failure before publish removes staging and leaves the visible
vault unchanged. Failure while promoting registry metadata removes promoted
metadata and renames the complete visible run back to staging before returning
an error. The transaction journal lets startup finish that rollback after a
process or power failure. A failed post-commit index refresh reports an
explicit warning but does not misreport the already committed filesystem tree
as rolled back.

Startup resolves transaction journals before scanning the workspace tree, then
removes abandoned import staging directories after confirming they are not
owned by an active session. Recovery deletes only registry files named by an
uncommitted journal and moves only that journal's newly published run; existing
visible content is never deleted as generic cleanup.

## Wizard UX

The Plugin Manager Settings button opens a wide modal with four steps.

### 1. Source

The first step offers separate native actions for a directory and an archive,
shows the selected display path, lists supported archive formats, and enables
Analyze only after a valid selection. Multiple detected source roots are shown
for explicit selection.

### 2. Analysis

The modal reports progress through structure discovery, page reading, link
analysis, classification, and plan generation. Analysis is cancellable and
makes no vault changes.

### 3. Structure

The left pane shows the complete proposed target tree and counts. The right
pane edits the selected node's name and type and explains the proposal,
relationships, and warnings. Filters expose low-confidence and warning nodes.
The responsive narrow layout stacks the tree above the inspector.

Before confirmation the UI always warns that imported material may contain
credentials or other sensitive information and that the user should review,
move to Secrets, or delete it afterward. No page is singled out by content
inspection.

### 4. Import

The last step shows a final count summary, staging progress, and the point after
which cancellation is unavailable. Completion reports created folders, Deals,
notes, files, user-skipped nodes, conversion warnings, unresolved links, and
errors. Actions close the wizard or open the imported run in the workspace
tree.

Closing before apply closes the source session and changes nothing. Closing
during analysis cancels it. Closing during staging requires confirmation.

## Visual Contract

The implementation reuses the current Plugin Manager modal sizing and host
surface rules. It follows existing Verstak design tokens and patterns for:

- colors, typography, radii, borders, spacing, and control heights;
- focus, hover, disabled, progress, error, and warning states;
- select controls and responsive filter groups;
- Lucide icons rendered through the shell's supported icon contract.

The importer must not introduce emoji icons, a separate theme, arbitrary inline
palette values, or a private copy of shell components. If a reusable public
plugin UI token is missing, it is added as generic host infrastructure rather
than encoded as an importer-only approximation.

## Errors, Warnings, and Reporting

Fatal errors stop analysis or application: corrupt archive, unsafe path,
unsupported archive, source mutation, resource limit, insufficient space,
invalid plan, staging write failure, or final target conflict.

Warnings do not silently change data: unsupported DokuWiki syntax, unresolved
or ambiguous links, safe filename projection, unsupported metadata, or failed
post-commit index refresh. Informational entries cover explicit user skips and
accepted type changes.

Frontend and backend logs contain operation IDs, normalized paths, counts, and
error codes, but never page bodies, credentials, or binary payloads. Plugin
storage retains the last summary and warnings only; a new import replaces it.

## Repository Changes

Expected change ownership is:

- `verstak-sdk`: capability/permission registries, schemas, DTO/types, frontend
  API contract, and contract tests.
- `verstak-desktop`: generic source/archive service, generic plan applier,
  permission-checked Wails API, frontend API wrapper, modal host integration,
  lifecycle cleanup, tests, and runtime documentation.
- `verstak-official-plugins`: `verstak.import` manifest, RU/EN catalogs,
  frontend implementation, DokuWiki/Obsidian adapters, fixtures, unit/smoke/E2E
  coverage, packaging, and README updates.
- `verstak-docs`: platform/plugin reference, official-plugin list, user import
  guide, and release notes/roadmap status.

Every significant, internally complete block is committed and pushed in its
own repository before the next dependent block is considered complete.

## Verification

### Synthetic Fixtures

Small repository-owned fixtures cover:

- DokuWiki pages, media, formatting, tables, code/file blocks, links, sections,
  missing targets, plugin syntax, wrapper directories, and ignored revisions;
- Obsidian frontmatter, aliases, headings, block IDs, embeds, attachments,
  duplicate basenames, and a mixed project subtree;
- safe ZIP/TAR/TAR.GZ sources and malicious traversal/link/size cases.

Fixtures contain no personal backup data.

### Automated Checks

- SDK schema validation, TypeScript compile, and API unit tests.
- Go unit and integration tests for sessions, archive formats, safety limits,
  fingerprints, permission checks, plan validation, staging, cancellation,
  rollback, workspace metadata, and cleanup.
- Importer unit/golden tests for conversion, graph resolution,
  classification, normalization, and deterministic planning.
- Official-plugin manifest, localization, public API boundary, responsive
  layout, select-style, and user-facing error checks.
- Playwright flows for all wizard steps, candidate selection, plan edits,
  warning behavior, cancellation, completion, repeated import, and plugin
  disable/re-enable cleanup.
- Full SDK, desktop, and official-plugin builds.

Non-trivial Go and TypeScript changes receive LSP diagnostics before and after
where the environment exposes the language servers.

### Supplied Backup Smoke Test

The local `wiki.tar.gz` and `Obsidian.tar.gz` are never added to git. A focused
integration/smoke workflow imports each into a temporary vault and verifies:

- detected format and candidate root;
- expected current-page/content inventory;
- representative converted notes and copied files;
- rewritten internal links and media references;
- absence of DokuWiki revisions and Obsidian `.obsidian` data;
- isolated unique run directories;
- successful repeated import without overwriting the first run.

Test output contains counts and safe synthetic assertions, not personal page
content or sensitive filenames.

### Real GUI Check

Run the real Wails desktop with packaged development plugins. Use native dialogs
to select both supplied archives, inspect and edit the plan, complete both
imports into a disposable vault, open representative results, repeat one
import, and disable/re-enable the plugin. Playwright alone is not sufficient for
this gate.

## Release and Publication

After every repository is clean, pushed, and verified together:

1. Determine the next compatible version from existing tags and release-script
   conventions.
2. Run the official-plugin and desktop release checks and packaging scripts in
   their required dependency order.
3. Publish GitHub releases with generated archives/installers and checksums.
4. Verify tags, release pages, asset names, non-zero sizes, checksums, and the
   update metadata/channel used by the application.
5. Install or update from the published artifacts and perform a final startup
   and Plugin Manager availability check.

The task is not complete when local builds pass. It is complete only after the
published release can be used to update and exercise the importer.

## Success Criteria

- Plugin Manager shows one enabled/disabled official `Import` plugin with a
  Settings button and accurate permissions/status.
- A directory or supported archive containing DokuWiki or Obsidian content can
  be selected with a native dialog.
- Analysis writes nothing to the vault and produces a deterministic editable
  structure proposal.
- DokuWiki current pages become Markdown with working representative links and
  media; revision history and service data do not appear.
- Obsidian user data imports without `.obsidian`; representative wikilinks,
  embeds, attachments, and mixed project trees remain usable.
- Sensitive material receives only the agreed general warning and is otherwise
  imported as ordinary content.
- Every run is isolated below `Импортировано`, never overwrites existing data,
  and either publishes a complete run or leaves no visible partial run.
- Cancellation, plugin disable/re-enable, source mutation, malicious archives,
  and repeated imports behave as specified.
- Documentation matches the implemented contract.
- Significant changes are committed and pushed, all required checks and real
  GUI scenarios pass, and verified GitHub release artifacts are published for
  update and user testing.
