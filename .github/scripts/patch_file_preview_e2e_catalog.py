from pathlib import Path


def replace_once(text, old, new, label):
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label} anchor count: {count}")
    return text.replace(old, new, 1)


path = Path("frontend/src/lib/test/wails-mock.js")
text = path.read_text()

text = replace_once(
    text,
    "import filesSource from '../../../../../verstak-official-plugins/plugins/files/frontend/src/index.js?raw';\n",
    "import filesSource from '../../../../../verstak-official-plugins/plugins/files/frontend/src/index.js?raw';\n"
    "import filePreviewSource from '../../../../../verstak-official-plugins/plugins/file-preview/frontend/src/index.js?raw';\n",
    "file preview source import",
)
text = replace_once(
    text,
    "import defaultEditorManifest from '../../../../../verstak-official-plugins/plugins/default-editor/plugin.json';\n",
    "import defaultEditorManifest from '../../../../../verstak-official-plugins/plugins/default-editor/plugin.json';\n"
    "import filePreviewManifest from '../../../../../verstak-official-plugins/plugins/file-preview/plugin.json';\n",
    "file preview manifest import",
)
text = replace_once(
    text,
    "import browserRuCatalog from '../../../../../verstak-official-plugins/plugins/browser-inbox/locales/ru.json';\n",
    "import browserRuCatalog from '../../../../../verstak-official-plugins/plugins/browser-inbox/locales/ru.json';\n"
    "import filePreviewEnCatalog from '../../../../../verstak-official-plugins/plugins/file-preview/locales/en.json';\n"
    "import filePreviewRuCatalog from '../../../../../verstak-official-plugins/plugins/file-preview/locales/ru.json';\n",
    "file preview locale imports",
)

# One canonical catalog must describe the official plugins present in the E2E
# vault. Plugin runtime state and vault enabled/desired state are projections of
# the same list rather than separately maintained inventories.
start = text.index("  function makeDefaultPluginStates() {\n")
end = text.index("\n\n  var pluginStates = makeDefaultPluginStates();", start)
replacement = """  var officialPluginFixtures = [
    [platformTestManifest, 'platform-test'],
    [defaultEditorManifest, 'default-editor'],
    [filePreviewManifest, 'file-preview'],
    [filesManifest, 'files'],
    [trashManifest, 'trash'],
    [notesManifest, 'notes'],
    [syncManifest, 'sync'],
    [activityManifest, 'activity'],
    [journalManifest, 'journal'],
    [browserInboxManifest, 'browser-inbox'],
    [todoManifest, 'todo'],
    [secretsManifest, 'secrets'],
    [importManifest, 'import'],
    [searchManifest, 'search']
  ];

  function makeDefaultPluginStates() {
    var states = {};
    officialPluginFixtures.forEach(function (fixture) {
      states[fixture[0].id] = makePluginState(fixture[0], fixture[1]);
    });
    return states;
  }

  function makeDefaultVaultPluginState() {
    return {
      enabledPlugins: officialPluginFixtures.map(function (fixture) { return fixture[0].id; }),
      disabledPlugins: [],
      desiredPlugins: officialPluginFixtures.map(function (fixture) {
        return { id: fixture[0].id, version: fixture[0].version, source: 'official' };
      })
    };
  }"""
text = text[:start] + replacement + text[end:]

# File Preview has UI-localized strings, so exercise its real catalogs too.
text = text.replace("realOverviewPluginCatalogs", "realPluginCatalogs")
text = replace_once(
    text,
    "    'verstak.browser-inbox': { en: browserEnCatalog, ru: browserRuCatalog },\n",
    "    'verstak.browser-inbox': { en: browserEnCatalog, ru: browserRuCatalog },\n"
    "    'verstak.file-preview': { en: filePreviewEnCatalog, ru: filePreviewRuCatalog },\n",
    "file preview real locale catalog",
)

# Replace both hand-maintained vault inventories: initial state and reset().
initial_start = text.index("  var vaultPluginState = {")
initial_end = text.index("  var appSettings = {", initial_start)
text = text[:initial_start] + "  var vaultPluginState = makeDefaultVaultPluginState();\n" + text[initial_end:]

reset_start = text.index("      vaultPluginState = {")
reset_end = text.index("      appSettings = {", reset_start)
text = text[:reset_start] + "      vaultPluginState = makeDefaultVaultPluginState();\n" + text[reset_end:]

# Load File Preview from the source declared by its real manifest.
asset_anchor = """      if (pluginId === defaultEditorManifest.id && assetPath === defaultEditorManifest.frontend.entry) {
        return Promise.resolve(defaultEditorSource);
      }
"""
asset_addition = asset_anchor + """      if (pluginId === filePreviewManifest.id && assetPath === filePreviewManifest.frontend.entry) {
        return Promise.resolve(filePreviewSource);
      }
"""
text = replace_once(text, asset_anchor, asset_addition, "file preview asset loader")

if "vaultPluginState.enabledPlugins.push(" in text:
    raise SystemExit("manual vault plugin inventory still present")
path.write_text(text)

# Lock the single-catalog model and File Preview's real bundle into the source
# contract so future shipped plugins cannot silently disappear from E2E again.
path = Path("frontend/tests/shell-source-contract-test.mjs")
text = path.read_text()
text = replace_once(
    text,
    "  ['default-editor', 'defaultEditorManifest'],\n",
    "  ['default-editor', 'defaultEditorManifest'],\n  ['file-preview', 'filePreviewManifest'],\n",
    "file preview official fixture guard",
)
anchor = """assertExcludes(
  wailsMock,
  'function filesPluginBundle()',
  'E2E should not maintain a divergent synthetic Files implementation',
);

"""
addition = anchor + """assertIncludes(
  wailsMock,
  "plugins/file-preview/frontend/src/index.js?raw",
  'E2E should exercise the real shipped File Preview frontend bundle',
);
assertIncludes(
  wailsMock,
  'Promise.resolve(filePreviewSource)',
  'E2E File Preview asset loading should return the real official source',
);
assertIncludes(
  wailsMock,
  'var officialPluginFixtures = [',
  'E2E should define one canonical official plugin fixture catalog',
);
assertIncludes(
  wailsMock,
  'enabledPlugins: officialPluginFixtures.map(',
  'E2E vault enabled plugins should be projected from the official fixture catalog',
);
assertIncludes(
  wailsMock,
  'desiredPlugins: officialPluginFixtures.map(',
  'E2E vault desired plugins should be projected from the official fixture catalog',
);
assertExcludes(
  wailsMock,
  'vaultPluginState.enabledPlugins.push(',
  'E2E should not maintain a second incremental official plugin inventory',
);

"""
text = replace_once(text, anchor, addition, "single official fixture catalog guard")
path.write_text(text)
