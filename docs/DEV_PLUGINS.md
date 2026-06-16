# Verstak Desktop — Development Plugin Flow

## Overview

Official plugins live in the **verstak-official-plugins** monorepo and are developed
separately from the desktop core. During development, plugins are installed into
the desktop's local `plugins/` directory as **packaged bundles** (not source code).

```
git/
  verstak-desktop/                ← desktop core
  verstak-official-plugins/       ← official plugin source + dist output
```

## Plugin Source → Package Flow

1. **Source code** lives in `verstak-official-plugins/plugins/<plugin-id>/`
2. **Running `build.sh`** in official-plugins:
   - Builds frontend (if frontend/package.json exists)
   - Builds backend Go binary (if backend/main.go exists)
   - Packages the result into `dist/<plugin-id>/`
3. **Package structure** (`dist/platform-test/`):
   ```
   plugin.json
   frontend/dist/index.js
   backend/platform-test          (compiled binary)
   ```

## Installing Dev Plugins in Desktop

From the `verstak-desktop/` directory:

```bash
./scripts/install-dev-plugins.sh
```

This script:
- Locates `../verstak-official-plugins/`
- Builds packages there if `dist/` is stale
- Creates `./plugins/<plugin-id>/` from the dist package
- Does **not** affect other plugins in `./plugins/`

The `plugins/` directory is in `.gitignore` — it is never committed.

## Smoke Test

After installing, verify that the desktop runtime discovers the plugin:

```bash
./scripts/smoke-platform.sh
```

This validates:
- Plugin directory exists
- `plugin.json` manifest is valid
- All required manifest fields are present
- `DiscoverPlugins()` finds `verstak.platform-test`
- Capabilities are registerable

## Desktop Runtime Scanning Paths

The desktop scans two directories for plugins:

| Path | Purpose |
|------|---------|
| `~/.config/verstak/plugins/` | User-installed plugins |
| `./plugins/` | Bundled/dev plugins (project-local) |

## Important Rules

- **Never commit** `plugins/` to `verstak-desktop` — it's in `.gitignore`
- **Never copy source code** from `verstak-official-plugins/plugins/` directly —
  always use the dist package from `verstak-official-plugins/dist/`
- **Run `install-dev-plugins.sh`** after any change in the plugin source
- **Run `smoke-platform.sh`** after installing to verify discovery
