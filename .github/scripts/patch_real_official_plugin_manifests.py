from pathlib import Path


def replace_once(text, old, new, label):
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label} anchor count: {count}")
    return text.replace(old, new, 1)


path = Path("frontend/src/lib/test/wails-mock.js")
text = path.read_text()

# Official plugins own their manifest contract. The Wails test bridge should
# mock host APIs and data, not maintain a second copy of permissions,
# contributions, frontend entries, or optional requirements.
anchor = "import filesManifest from '../../../../../verstak-official-plugins/plugins/files/plugin.json';\n"
imports = anchor + """import platformTestManifest from '../../../../../verstak-official-plugins/plugins/platform-test/plugin.json';
import defaultEditorManifest from '../../../../../verstak-official-plugins/plugins/default-editor/plugin.json';
import trashManifest from '../../../../../verstak-official-plugins/plugins/trash/plugin.json';
import notesManifest from '../../../../../verstak-official-plugins/plugins/notes/plugin.json';
import syncManifest from '../../../../../verstak-official-plugins/plugins/sync/plugin.json';
import activityManifest from '../../../../../verstak-official-plugins/plugins/activity/plugin.json';
import journalManifest from '../../../../../verstak-official-plugins/plugins/journal/plugin.json';
import browserInboxManifest from '../../../../../verstak-official-plugins/plugins/browser-inbox/plugin.json';
import todoManifest from '../../../../../verstak-official-plugins/plugins/todo/plugin.json';
import secretsManifest from '../../../../../verstak-official-plugins/plugins/secrets/plugin.json';
import importManifest from '../../../../../verstak-official-plugins/plugins/import/plugin.json';
import searchManifest from '../../../../../verstak-official-plugins/plugins/search/plugin.json';
"""
text = replace_once(text, anchor, imports, "official manifest imports")

state_start = text.index("  var pluginStates = {\n")
state_end_marker = "  var realOverviewPluginCatalogs = {"
state_end = text.index(state_end_marker, state_start)
factory = """  function makePluginState(manifest, slug) {
    return {
      status: 'loaded',
      enabled: true,
      manifest: JSON.parse(JSON.stringify(manifest)),
      rootPath: '/tmp/verstak-test/plugins/' + slug,
      error: ''
    };
  }

  function makeDefaultPluginStates() {
    var fixtures = [
      [platformTestManifest, 'platform-test'],
      [defaultEditorManifest, 'default-editor'],
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
    var states = {};
    fixtures.forEach(function (fixture) {
      states[fixture[0].id] = makePluginState(fixture[0], fixture[1]);
    });
    return states;
  }

  var pluginStates = makeDefaultPluginStates();

"""
# This also removes overviewProviderDefs: those declarations already live in
# the real Notes/Activity/Browser/Journal/Todo manifests.
text = text[:state_start] + factory + text[state_end:]

# Real manifests already declare localization. Do not rewrite that contract in
# the mock after loading it.
old = """  Object.keys(pluginStates).forEach(function (pluginId) {
    var manifest = pluginStates[pluginId].manifest;
    manifest.localization = {
      defaultLocale: 'en',
      locales: { en: 'locales/en.json', ru: 'locales/ru.json' }
    };
  });

"""
text = replace_once(text, old, "", "synthetic localization mutation")

# Remove helper factories whose only purpose was hand-written official
# manifests. Test-only synthetic plugins created later are intentionally kept.
helpers_start = text.index("  function makeTrashPluginState() {\n")
helpers_end = text.index("  function importEntry(", helpers_start)
text = text[:helpers_start] + text[helpers_end:]

# Asset selection must follow each real manifest's declared frontend entry.
replacements = {
    "if (pluginId === 'verstak.platform-test' && assetPath === 'frontend/dist/index.js') {": "if (pluginId === platformTestManifest.id && assetPath === platformTestManifest.frontend.entry) {",
    "if (pluginId === 'verstak.default-editor') {": "if (pluginId === defaultEditorManifest.id && assetPath === defaultEditorManifest.frontend.entry) {",
    "if (pluginId === 'verstak.files' && assetPath === filesManifest.frontend.entry) {": "if (pluginId === filesManifest.id && assetPath === filesManifest.frontend.entry) {",
    "if (pluginId === 'verstak.trash' && assetPath === 'frontend/dist/index.js') {": "if (pluginId === trashManifest.id && assetPath === trashManifest.frontend.entry) {",
    "if (pluginId === 'verstak.notes' && assetPath === 'frontend/dist/index.js') {": "if (pluginId === notesManifest.id && assetPath === notesManifest.frontend.entry) {",
    "if (pluginId === 'verstak.sync' && assetPath === 'frontend/dist/index.js') {": "if (pluginId === syncManifest.id && assetPath === syncManifest.frontend.entry) {",
    "if (pluginId === 'verstak.activity') {": "if (pluginId === activityManifest.id && assetPath === activityManifest.frontend.entry) {",
    "if (pluginId === 'verstak.journal' && assetPath === 'frontend/dist/index.js') {": "if (pluginId === journalManifest.id && assetPath === journalManifest.frontend.entry) {",
    "if (pluginId === 'verstak.browser-inbox' && assetPath === 'frontend/dist/index.js') {": "if (pluginId === browserInboxManifest.id && assetPath === browserInboxManifest.frontend.entry) {",
    "if (pluginId === 'verstak.todo' && assetPath === 'frontend/dist/index.js') {": "if (pluginId === todoManifest.id && assetPath === todoManifest.frontend.entry) {",
    "if (pluginId === 'verstak.secrets') {": "if (pluginId === secretsManifest.id && assetPath === secretsManifest.frontend.entry) {",
    "if (pluginId === 'verstak.search' && assetPath === 'frontend/dist/index.js') {": "if (pluginId === searchManifest.id && assetPath === searchManifest.frontend.entry) {",
    "if (pluginId === 'verstak.import' && assetPath === 'frontend/dist/index.js') {": "if (pluginId === importManifest.id && assetPath === importManifest.frontend.entry) {",
    "if (pluginId === 'verstak.import' && assetPath === 'frontend/dist/style.css') {": "if (pluginId === importManifest.id && assetPath === importManifest.frontend.style) {",
}
for old, new in replacements.items():
    text = replace_once(text, old, new, f"asset contract: {old}")

# reset() must create fresh clones from the same official manifests rather than
# restoring a second stale manifest table.
reset_start = text.index("      pluginStates = {\n")
reset_end = text.index("      vaultStatus =", reset_start)
text = text[:reset_start] + "      pluginStates = makeDefaultPluginStates();\n" + text[reset_end:]

path.write_text(text)

# Lock the architecture boundary so a hand-written official manifest cannot
# quietly return later.
path = Path("frontend/tests/shell-source-contract-test.mjs")
text = path.read_text()
old_files_state_guard = """assertIncludes(
  wailsMock,
  'manifest: filesManifest',
  'E2E Files plugin state should use the real official manifest',
);
"""
text = replace_once(
    text,
    old_files_state_guard,
    "",
    "legacy Files manifest state guard",
)
anchor = """assertExcludes(
  wailsMock,
  'function filesPluginBundle()',
  'E2E should not maintain a divergent synthetic Files implementation',
);

"""
addition = anchor + """const officialManifestFixtures = [
  ['platform-test', 'platformTestManifest'],
  ['default-editor', 'defaultEditorManifest'],
  ['files', 'filesManifest'],
  ['trash', 'trashManifest'],
  ['notes', 'notesManifest'],
  ['sync', 'syncManifest'],
  ['activity', 'activityManifest'],
  ['journal', 'journalManifest'],
  ['browser-inbox', 'browserInboxManifest'],
  ['todo', 'todoManifest'],
  ['secrets', 'secretsManifest'],
  ['import', 'importManifest'],
  ['search', 'searchManifest'],
];
for (const [slug, binding] of officialManifestFixtures) {
  assertIncludes(
    wailsMock,
    `plugins/${slug}/plugin.json`,
    `E2E should import the real official manifest for ${slug}`,
  );
  assertIncludes(
    wailsMock,
    `[${binding}, '${slug}']`,
    `E2E plugin state should be built from the real ${slug} manifest`,
  );
  assertIncludes(
    wailsMock,
    `pluginId === ${binding}.id && assetPath === ${binding}.frontend.entry`,
    `E2E asset loading should follow the real ${slug} frontend entry`,
  );
}
for (const staleFactory of [
  'makeTrashPluginState',
  'makeTodoPluginState',
  'makeSecretsPluginState',
  'makeImportPluginState',
  'overviewProviderDefs',
]) {
  assertExcludes(
    wailsMock,
    staleFactory,
    `E2E should not reconstruct official manifest metadata through ${staleFactory}`,
  );
}
assertIncludes(
  wailsMock,
  'pluginStates = makeDefaultPluginStates();',
  'E2E reset should restore fresh clones from the same official manifests',
);

"""
text = replace_once(text, anchor, addition, "manifest source-contract guard")
path.write_text(text)
