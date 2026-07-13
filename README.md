# Verstak

Verstak is a local-first workspace for files, notes, browser captures,
activity and work-journal entries. This repository is the desktop application:
the Go/Wails host and UI shell that loads plugins from a `plugins/` directory
next to its executable.

> **Alpha software.** Use a disposable vault while evaluating it. APIs, storage
> formats and packaging can change before the first stable release.

## Components

The public alpha is split into small repositories. Keep their `main` branches
in the same release line when building from source.

| Component | Repository | What it is for |
| --- | --- | --- |
| Desktop | [mirivlad/verstak](https://github.com/mirivlad/verstak) | This application: vault UI, local host and plugin runtime. |
| Official plugins | [mirivlad/verstak-official-plugins](https://github.com/mirivlad/verstak-official-plugins) | Files, Notes, Browser Inbox, Activity, Journal, Sync, Todo and other first-party plugins. |
| Browser extension | [mirivlad/verstak-browser-extension](https://github.com/mirivlad/verstak-browser-extension) | Manual browser captures and opt-in domain-time activity. |
| Sync server | [mirivlad/verstak-sync-server](https://github.com/mirivlad/verstak-sync-server) | Optional self-hosted synchronization between devices. |
| Plugin SDK | [mirivlad/verstak-sdk](https://github.com/mirivlad/verstak-sdk) | TypeScript API, JSON schemas and contract tests for plugin authors. |
| Architecture documentation | [mirivlad/verstak-docs](https://github.com/mirivlad/verstak-docs) | Product, platform and plugin-system design documents. |

No server account or browser extension is needed for a local desktop vault.

## Build the desktop and official plugins

To build from source, install Go 1.24+, Node.js 20+ with npm, Python 3, the [Wails v2 build
prerequisites](https://wails.io/docs/gettingstarted/installation/), and your
distribution's WebKitGTK development package.

Clone the repositories as siblings. The directory names below are intentional:
the desktop helper finds the official-plugin checkout at
`../verstak-official-plugins`.

```text
verstak-workspace/
├── verstak/
├── verstak-sdk/
├── verstak-official-plugins/
└── verstak-browser-extension/   # optional for browser integration
```

```bash
git clone https://github.com/mirivlad/verstak.git verstak
git clone https://github.com/mirivlad/verstak-sdk.git verstak-sdk
git clone https://github.com/mirivlad/verstak-official-plugins.git verstak-official-plugins
git clone https://github.com/mirivlad/verstak-browser-extension.git verstak-browser-extension

cd verstak-sdk && ./scripts/build.sh
cd ../verstak-official-plugins && ./scripts/build.sh
cd ../verstak
./scripts/install-dev-plugins.sh
./scripts/build.sh
```

`install-dev-plugins.sh` copies the packages from
`../verstak-official-plugins/dist/` into this repository's `plugins/`
directory. `build.sh` then copies them to `build/bin/plugins/`, beside the
desktop executable:

```bash
./build/bin/verstak-desktop
```

For a manually assembled installation, place each unpacked plugin directory
directly in `plugins/` beside `verstak-desktop`; for example,
`plugins/browser-inbox/plugin.json`. Do not put the release archive itself in
that directory. The desktop release archive already includes its matching
`plugins/` directory.

Start with `--debug` to show internal plugin-provider identifiers and write
diagnostic logs:

```bash
./build/bin/verstak-desktop --debug
```

## Portable test artifacts

These commands make local alpha artifacts in `release/`. They do not create a
GitHub Release.

```bash
# Debian 13 / Ubuntu 24.04 or later. APT installs WebKitGTK dependencies.
./scripts/package-deb.sh v0.1.0-alpha.1

# Portable Linux x86_64 AppImage with bundled WebKitGTK runtime and plugins.
./scripts/package-appimage.sh v0.1.0-alpha.1
```

Install the Debian package with `sudo apt install ./release/verstak_*.deb`,
then launch `verstak`. Run the AppImage directly after making it executable:

```bash
chmod +x release/verstak-linux-x86_64-v0.1.0-alpha.1.AppImage
./release/verstak-linux-x86_64-v0.1.0-alpha.1.AppImage
```

On distributions without FUSE support, use
`APPIMAGE_EXTRACT_AND_RUN=1 ./release/verstak-linux-x86_64-v0.1.0-alpha.1.AppImage`.

Build the Windows portable ZIP on Linux with MinGW:

```bash
sudo apt install gcc-mingw-w64-x86-64 zip
./scripts/package-windows-portable.sh v0.1.0-alpha.1
```

The ZIP uses the x64 Evergreen Microsoft WebView2 Runtime installed in Windows
and writes `release/verstak-windows-amd64-<version>.zip`. Extract it to a local
disk (not a network share) and launch `Verstak.cmd`. Windows 11 and most
Windows 10 installations already include this runtime. If Verstak does not
start, install the official [Microsoft WebView2 Runtime x64 standalone
installer](https://go.microsoft.com/fwlink/p/?LinkId=2124701), then launch it
again.

Each format is listed in `release/SHA256SUMS` after packaging.

## First local vault

1. Launch the desktop application and choose or create a writable vault folder.
   Verstak stores its local metadata in that vault; do not point it at a
   read-only directory.
2. Create a Дело (workspace) in the Files plugin before assigning captures or
   activity to it.
3. Open Notes, Files, Activity and Journal as needed. Activity candidates are
   only suggestions: a Journal entry and a new Дело are always created by the
   user.

## Browser extension

The extension is optional. Build it locally with `npm ci && npm test && npm run
build` in `verstak-browser-extension/`, then load `dist/chromium` as an
unpacked Chromium extension or `dist/firefox` temporarily in Firefox. A signed
Firefox XPI is available from the [extension's GitHub
Releases](https://github.com/mirivlad/verstak-browser-extension/releases).

To connect it to the desktop application:

1. Ensure the `browser-inbox` plugin is installed and open its settings in
   Verstak.
2. Copy the displayed Receiver URL and Pairing Token.
3. Paste both values into the extension's settings and save.
4. Use a manual Send Page, selection, link or file action to create a Browser
   Inbox capture.

Passive domain activity is disabled by default. When the user explicitly turns
it on, the extension sends only bounded time totals by normalized domain. It
does not send URLs, page titles, page contents, keystrokes, navigation history
or inactive-tab time. The extension settings provide a domain exclusion list.

## Optional sync server

The sync server is self-hosted and is not required for local use. Build and
start a development instance from a sibling checkout:

```bash
git clone https://github.com/mirivlad/verstak-sync-server.git verstak-sync-server
cd verstak-sync-server
./scripts/build.sh
./build/bin/verstak-sync-server --port 47732 --data ./server-data \
  --admin-user admin --admin-pass 'choose-a-strong-password'
```

For a second device or a production host, follow the deployment, HTTPS and
backup guidance in the [sync server
README](https://github.com/mirivlad/verstak-sync-server#readme). In the desktop
application, open the Sync plugin, enter the server URL and user credentials,
test the connection, then select **Connect**. Each vault is paired separately.

## Separate plugin archives

The desktop artifacts already include matching official plugins. To distribute
plugins separately, build OS-specific archives in the sibling repository:

```bash
cd ../verstak-official-plugins
sudo apt install gcc-mingw-w64-x86-64 zip
./scripts/package-portable.sh v0.1.0-alpha.1
```

It writes a Linux `tar.gz` and a Windows ZIP. Both archives expand directly
into the `plugins/` directory beside the corresponding desktop executable.
Separate archives are required because `platform-test` includes a native
sidecar for its target OS.

## Publish a GitHub Release

The current publisher still uses the legacy Linux tarball. It will be switched
to the portable artifacts after they have been manually checked:

```bash
./scripts/publish-github-release.sh v0.1.0-alpha.1
```

The command requires an authenticated [`gh`](https://cli.github.com/) CLI, a
clean local `main` equal to `origin/main`, and the sibling official-plugins
checkout. It runs the local release script, creates an annotated Git tag when
needed, pushes that tag through `origin`, then creates or updates the GitHub
Release with the archive and `SHA256SUMS`. In this checkout `origin` also
pushes the tag to the configured mirror; other maintainers can add a second
push URL if they use a mirror too. Publish the compatible official-plugins
release before publishing the desktop archive that embeds those plugins.

## License

Copyright © 2026 Verstak contributors. Licensed under
[GNU AGPLv3 or later](LICENSE).
