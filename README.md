# Verstak Desktop

Verstak is a local-first desktop workspace for files, notes, browser captures,
activity and work journal entries. This repository contains the Go/Wails desktop
host and UI shell; user-facing functions are delivered by official plugins.

> **Alpha software.** Use a disposable vault while evaluating it. APIs, storage
> formats and packaging can change before the first stable release.

## What the alpha does

- keeps each workspace's identity independent of its current directory name;
- keeps browser captures in a reviewable Inbox, with archive and restore rather
  than a destructive “remove from inbox” action;
- records local file/note activity and, after review, turns it into journal
  entries without creating a workspace automatically;
- accepts optional browser domain-time batches. The browser extension is
  opt-in: it sends only normalized domain names and bounded durations, never
  URLs, page titles, page contents, keystrokes or navigation history;
- runs official plugins from a `plugins/` directory next to the executable.

## Components

Clone these repositories as siblings when building a complete local setup:

| Repository | Purpose |
| --- | --- |
| `verstak-desktop` | desktop host and UI shell |
| `verstak-official-plugins` | Files, Notes, Browser Inbox, Activity, Journal and other official plugins |
| `verstak-browser-extension` | Chromium/Firefox capture and optional domain-activity extension |
| `verstak-sdk` | plugin manifest schema and TypeScript API |

## Build from source (Linux)

Requirements: Go, Node.js with npm, Python 3, the [Wails v2 build
prerequisites](https://wails.io/docs/gettingstarted/installation/), and the
WebKitGTK development package for your distribution.

```bash
git clone https://git.mirv.top/verstak/verstak-sdk.git
git clone https://git.mirv.top/verstak/verstak-official-plugins.git
git clone https://git.mirv.top/verstak/verstak-desktop.git
git clone https://git.mirv.top/verstak/verstak-browser-extension.git

cd verstak-sdk && ./scripts/build.sh
cd ../verstak-official-plugins && ./scripts/build.sh
cd ../verstak-desktop
./scripts/install-dev-plugins.sh
./scripts/build.sh
```

The Linux executable is placed in `build/bin/`. Start it with `--debug` to
display internal plugin-provider IDs and write diagnostic logs.

## Release artifacts

Maintainers can create a Linux desktop tarball after the sibling repositories
above have been built:

```bash
cd verstak-desktop
./scripts/release.sh v0.1.0-alpha.1
```

It writes `release/verstak-desktop-linux-amd64-<version>.tar.gz` and a matching
`SHA256SUMS` file. The archive is self-contained: unpack it and run the
included executable. Browser and SDK packages have their own release scripts
in their repositories.

## Privacy and activity tracking

The extension's passive domain tracker is disabled by default. Enabling it
requires an explicit choice in the extension and lets the user maintain a
domain exclusion list. Manual page, selection and link captures are separate
actions; they enter Browser Inbox and never create a workspace or journal
entry by themselves.

## License

Copyright © 2026 Verstak contributors. Licensed under
[GNU AGPLv3 or later](LICENSE).
