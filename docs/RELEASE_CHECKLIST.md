# Release smoke checklist

What has to be true before a build goes to anybody who is not the person who
made it. Everything here is either a command that answers for itself or a thing
somebody has to look at, and the difference is marked.

The automated parts are one script:

```bash
scripts/release-smoke.sh
```

It fails loudly. What it cannot check is at the bottom, and that part is not
optional either — the failures that reached users were the ones no script was
watching for.

## 1. The tree

- [ ] every repository is committed and pushed; `git status` is clean in
      `verstak-desktop`, `verstak-official-plugins`, `verstak-sdk`,
      `verstak-docs`, `verstak-sync-server`, `verstak-browser-extension`
- [ ] `scripts/verstak-check.sh` passes for every repository, Playwright
      included — it runs by default now, and it was opt-in on the day four
      specs were red for a week

## 2. The packages

- [ ] `verstak-official-plugins/scripts/build.sh` — every plugin packaged, each
      with its `checksums.txt`
- [ ] `verstak-desktop/scripts/install-dev-plugins.sh` — the installed copy is
      the one that was just built. The build takes plugins from
      `verstak-desktop/plugins/`, not from the plugin repository, and shipping a
      version-old copy has happened
- [ ] `verstak-desktop/scripts/build.sh`
- [ ] `verstak-desktop/scripts/package-deb.sh` and `package-appimage.sh`
- [ ] `scripts/release-smoke.sh` — packages exist, contain what they claim,
      and every packaged plugin verifies against its own checksums

## 3. The application, actually started

Not a test double. `scripts/gui-probe.sh` runs the real WebKitGTK build on a
virtual display; Playwright runs Chromium and cannot see what the native engine
paints.

- [ ] `scripts/gui-probe.sh --shot release` starts and the screenshot shows the
      vault, not the vault-selection screen or an empty window
- [ ] every plugin is `loaded` in the Plugin Manager; nothing `degraded` or
      `failed` that was not degraded on purpose
- [ ] the status bar shows the version being released
- [ ] Settings → Diagnostics writes a report, and the report names this build

## 4. What only a person can check

- [ ] install the `.deb` (or run the AppImage) on a machine that is not the
      build machine, and open a real vault with it
- [ ] the release notes say what changed, in words a user recognises
- [ ] anything the notes promise, done once by hand
- [ ] a vault backup exists before anybody points a new build at real data

## 5. After

- [ ] tag pushed, release published, artifacts attached
- [ ] `verstak-docs` describes what shipped
