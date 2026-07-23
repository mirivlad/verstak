# Verstak Design System

## Purpose

Stage 2 turns the visual conventions already present in the desktop shell into
an explicit, testable design-system contract. The result must make core
surfaces and official plugins look and behave consistently without moving
plugin business logic into the shell or introducing a component framework.

This is a visual-architecture refactor. Backend APIs, Wails bindings, plugin
manifests, contribution contracts, persisted data, commands, and user-visible
feature behavior remain unchanged.

## Current State

`frontend/src/App.svelte` owns a useful but implicit collection of
`--vt-*` tokens and global `vt-*` classes. Shell components use part of it.
Official plugins independently reproduce the same toolbar, button, field,
row, card, alert, empty-state, menu, and modal CSS with token fallbacks.

The existing styling is visually close, but the contract is not isolated or
tested:

- application bootstrap styling and reusable primitives live inside
  `App.svelte`;
- several frequently repeated primitives are incomplete, especially fields,
  control variants, modal structure, status tones, and separators;
- legacy `btn-*` classes and newer `vt-*` classes overlap without an explicit
  compatibility rule;
- hard-coded palette values still appear in shell and official-plugin
  surfaces even where a semantic token exists;
- no focused check prevents accidental removal or renaming of the classes and
  variables used across repositories.

## Chosen Architecture

The desktop host owns one CSS contract in
`frontend/src/lib/ui/design-system.css`. `frontend/src/main.js` imports it
before creating the Svelte application.

The contract has three layers:

1. **Foundations** — semantic color, spacing, typography, radius, control
   size, focus, elevation, transition, and overlay tokens.
2. **Primitives** — composable `vt-*` classes for buttons, fields, headers,
   toolbars, tabs, rows, cards, split panes, empty states, alerts, badges,
   menus, modals, and scroll surfaces.
3. **Compatibility aliases** — existing `btn-primary`, `btn-secondary`,
   `btn-danger`, `btn-ghost`, and `btn-icon` selectors share the same rules as
   their `vt-*` equivalents for this stage.

Component-specific geometry and exceptional behavior remain local. For
example, the Files grid owns its column layout, and the tree owns drag target
indicators. Local CSS may compose or refine the shared primitives, but it
must use semantic tokens for common color, focus, control, and state rules.

Official plugins consume the host contract through inherited CSS variables
and semantic `vt-*` class names. Their injected CSS retains token fallback
values so a package can still render legibly in isolated smoke fixtures.
This does not create a new public SDK or plugin manifest capability.

## Alternatives Considered

### Shared Svelte or npm component package

This would provide strong component-level encapsulation but would force
plain-JS official plugins into a new build/runtime model. It adds release and
version coordination that is not justified by a visual refactor.

### Token-only normalization

This would reduce hard-coded colors with minimal churn, but repeated controls
would still drift in height, focus treatment, disabled state, spacing, and
semantic tone. It does not solve the missing reusable primitive contract.

### Host-owned CSS contract

This is the selected approach because it matches the current DOM mounting
model, preserves plugin boundaries, allows incremental adoption, and can be
verified without changing application behavior.

## Foundation Contract

Existing token names remain valid. The stylesheet adds only the semantic
values needed by repeated current surfaces:

- control backgrounds and hover/active foregrounds;
- input background and placeholder color;
- success, warning, and danger foreground/border variants;
- control heights for compact and standard density;
- transition timing;
- modal backdrop and overlay z-index.

Tokens use the `--vt-` prefix. No theme switcher or alternate palette is
introduced in this stage.

## Primitive Contract

The system supports these class families:

- `vt-page`, `vt-page-header`, `vt-page-title`, `vt-page-subtitle`;
- `vt-toolbar`, `vt-toolbar-group`, `vt-toolbar-spacer`,
  `vt-toolbar-status`;
- `vt-button`, with `primary`, `secondary`, `ghost`, `danger`, `icon`, and
  `compact` modifiers;
- `vt-field`, `vt-field-label`, `vt-input`, `vt-select`, `vt-textarea`,
  `vt-control`, and `vt-control-group`;
- `vt-tabbar`, `vt-tab`;
- `vt-list-row`, `vt-list-meta`, `vt-row-actions`;
- `vt-card`, `vt-split-pane`, `vt-split-list`, `vt-split-detail`;
- `vt-empty-state`, `vt-empty-title`, `vt-empty-hint`;
- `vt-inline-alert`, with `error`, `warning`, and `success` modifiers;
- `vt-badge`, with `accent`, `warning`, `success`, and `danger` modifiers;
- `vt-menu`, `vt-menu-item`, `vt-menu-separator`;
- `vt-modal-overlay`, `vt-modal`, `vt-modal-header`, `vt-modal-body`,
  `vt-modal-actions`;
- `scroll-surface`.

Primitive selectors must provide consistent hover, focus-visible, disabled,
selected, active, and destructive states. They do not encode product copy or
plugin-specific actions.

## Adoption Scope

### Desktop

- Move global reset, foundations, base controls, scrollbars, and reusable
  primitives out of `App.svelte`.
- Keep only application-layout CSS in `App.svelte`.
- Align existing `Modal.svelte`, `Select.svelte`, Workspace Tree forms and
  context menu, shell empty/error states, and Plugin Manager controls with the
  contract.
- Preserve all existing data attributes, Svelte events, keyboard behavior,
  responsive rules, and test selectors.

### Official plugins

Normalize the repeated visual rules in Files, Notes, Search, Activity,
Journal, Secrets, and Browser Inbox:

- use shared class names on common controls and state surfaces;
- replace hard-coded common palette values with semantic tokens;
- keep plugin-specific layout selectors and responsive rules local;
- keep existing mount/unmount behavior, DOM data attributes, actions,
  translated copy, and API calls unchanged.

Other official plugins remain in scope only when a common primitive can be
adopted mechanically without changing their layout or behavior.

## Accessibility and Interaction

- Every interactive primitive has a visible `:focus-visible` state.
- Disabled controls remain visibly disabled and do not gain hover emphasis.
- Semantic state must not rely on color alone when the current UI already has
  text, an icon, or an accessible label.
- Compact workbench density is preserved; standard controls remain 32 pixels
  high and compact controls remain 26 pixels high.
- Responsive breakpoints and keyboard/mouse semantics are not changed by
  class migration.
- Reduced-motion users receive zero-duration shared transitions.

## Testing

The desktop repository gains a focused Node contract test that loads
`design-system.css` and asserts the stable foundation variables, primitive
selectors, legacy aliases, focus-visible states, disabled states, semantic
tones, and reduced-motion rule.

Existing Svelte and plugin smoke tests remain the behavioral regression
oracle. Focused Playwright scenarios cover shell, modal, plugin manager, and
the migrated official tools. The final gate runs:

- desktop binding/contract tests and frontend build;
- focused and then full desktop Playwright;
- desktop Go tests because release branches must remain integrated;
- official-plugin `scripts/check.sh`;
- SDK lint, tests, and build to confirm no public contract change.

Production screenshots at desktop and narrow widths are captured and
inspected from representative shell, Files, Search, and split-pane surfaces.

## Success Criteria

- The reusable design system is imported independently of `App.svelte`.
- Shared selectors and existing aliases have a focused automated contract.
- Common shell and targeted official-plugin surfaces consume the same
  foundation and primitive vocabulary.
- Common state colors, control sizes, focus rings, disabled states, modal
  structure, and empty/error treatment no longer drift across those
  surfaces.
- No backend, Wails, plugin manifest, SDK, persistence, or feature behavior
  changes.
- All existing checks pass and visual evidence shows no clipping, overflow,
  unreadable state, or density regression at desktop and narrow widths.

## Out of Scope

- Light theme or user-selectable themes.
- New UI dependencies or a shared Svelte/npm component package.
- Public SDK guarantees for CSS classes.
- Information-architecture, workflow, copy, or feature changes.
- Rebuilding official plugins into Svelte packages.
- Refactoring unrelated application or plugin logic.
