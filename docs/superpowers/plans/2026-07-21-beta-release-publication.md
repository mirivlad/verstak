# Beta Release Publication Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish the verified Verstak desktop, official plugins, and sync server as coordinated `v0.1.0-beta.20260721` GitHub prereleases with bilingual release notes.

**Architecture:** Each repository receives a release-notes file tailored to its artifacts. The existing feature branch is fast-forwarded to `main`, tagged at the verified commit, and published with the repository's existing release assets and checksums.

**Tech Stack:** Git, GitHub CLI, Bash release scripts, Markdown, SHA-256.

## Global Constraints

- Publish version `v0.1.0-beta.20260721` in all three repositories.
- Include English and Russian notes in every GitHub Release.
- Use fast-forward updates only; `origin/main` must remain an ancestor of the release commit.
- Do not modify or remove unrelated files in the normal repository checkouts.
- Mark every GitHub Release as a prerelease.

---

### Task 1: Add bilingual release notes

**Files:**
- Create: `release-notes/v0.1.0-beta.20260721.md` in each repository.

**Interfaces:**
- Consumes: verified fixes and artifact lists from the beta-readiness branch.
- Produces: Markdown passed to `gh release create --notes-file`.

- [ ] **Step 1: Add English and Russian notes**

Use the established `Highlights`, `Главное`, and `Packages / Пакеты` structure and mention only verified changes.

- [ ] **Step 2: Validate the notes**

Run in each repository:

```bash
test -s release-notes/v0.1.0-beta.20260721.md
rg '^## (Highlights|Главное|Packages / Пакеты)$' release-notes/v0.1.0-beta.20260721.md
```

Expected: all three headings are present.

- [ ] **Step 3: Commit and push the notes**

```bash
git add release-notes/v0.1.0-beta.20260721.md docs/superpowers/plans/2026-07-21-beta-release-publication.md
git commit -m "docs: add beta release notes"
git push origin fix/beta-readiness-2026-07-20
```

Expected: each feature branch matches its upstream.

### Task 2: Promote the release commits to main

**Files:**
- No file changes.

**Interfaces:**
- Consumes: clean, pushed feature-branch commits.
- Produces: the same commits at `origin/main`.

- [ ] **Step 1: Recheck ancestry and remote state**

```bash
git fetch origin main --tags
git merge-base --is-ancestor origin/main HEAD
git status --porcelain
```

Expected: ancestry check succeeds and the release worktree is clean.

- [ ] **Step 2: Fast-forward remote main**

```bash
git push origin HEAD:main
```

Expected: Git reports a fast-forward update.

### Task 3: Build, tag, and publish the coordinated prereleases

**Files:**
- Generated, ignored artifacts under each repository's `release/` directory.

**Interfaces:**
- Consumes: repository release scripts, bilingual notes, and GitHub authentication.
- Produces: three public GitHub prereleases with platform assets.

- [ ] **Step 1: Build versioned artifacts**

```bash
./scripts/release.sh v0.1.0-beta.20260721
```

Expected: the required Linux and Windows artifacts and `SHA256SUMS` are generated.

- [ ] **Step 2: Verify artifacts**

```bash
(cd release && sha256sum -c SHA256SUMS)
```

Expected: every asset reports `OK`.

- [ ] **Step 3: Create and push the annotated tag**

```bash
git tag -a v0.1.0-beta.20260721 -m "Release v0.1.0-beta.20260721"
git push origin refs/tags/v0.1.0-beta.20260721
```

Expected: the tag points at the release commit now present on `origin/main`.

- [ ] **Step 4: Publish each GitHub prerelease**

Run `gh release create` with the repository's required assets, `--notes-file release-notes/v0.1.0-beta.20260721.md`, `--generate-notes`, `--verify-tag`, and `--prerelease`.

Expected: one public release URL for each repository.

### Task 4: Verify public release state

**Files:**
- No file changes.

**Interfaces:**
- Consumes: published GitHub Releases.
- Produces: final evidence for URLs, prerelease flags, tags, and asset names.

- [ ] **Step 1: Inspect release metadata**

```bash
gh release view v0.1.0-beta.20260721 --json url,isPrerelease,tagName,assets
```

Expected: `isPrerelease` is true, the tag matches, and all required assets are listed.

- [ ] **Step 2: Confirm repository cleanliness and remote equality**

```bash
git diff --check
git status --short --branch
git rev-list --left-right --count HEAD...@{upstream}
```

Expected: no tracked changes and `0 0` divergence.
