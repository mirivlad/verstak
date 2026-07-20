# Search and Rename Beta Hotfix Plan

**Goal:** ship a patch prerelease that restores file and note renaming, removes stale Browser Inbox results after conversion, and makes global search results usable.

## Scope

- Replace metadata preflights in Files and Notes with atomic non-overwriting move operations.
- Apply the same correction to Files duplication, which shared the broken metadata preflight.
- Distinguish Wails data arrays from the tuple-shaped test compatibility response when indexing files.
- Exclude archived Browser Inbox captures from global search while retaining the converted vault file or note.
- Widen the results panel to roughly three search-input widths and reserve room for five results without overflowing a small viewport.
- Add regression coverage for all affected behavior.

## Verification and release

- Run focused plugin smoke tests and global-search Playwright tests.
- Run the full desktop and official-plugin checks and production builds.
- Publish `v0.1.0-beta.20260721.1` desktop and official-plugin prereleases with bilingual release notes; the unchanged sync server is not republished.
