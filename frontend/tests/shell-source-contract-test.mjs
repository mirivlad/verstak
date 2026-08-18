import fs from 'node:fs';
import path from 'node:path';

const root = path.resolve('.');

function read(relativePath) {
  return fs.readFileSync(path.join(root, relativePath), 'utf8');
}

function assertIncludes(source, needle, message) {
  if (!source.includes(needle)) {
    throw new Error(message);
  }
}

function assertExcludes(source, needle, message) {
  if (source.includes(needle)) {
    throw new Error(message);
  }
}

const workspaceHost = read('frontend/src/lib/shell/WorkspaceHost.svelte');
const overviewSurface = read('frontend/src/lib/shell/OverviewSurface.svelte');
const globalSearch = read('frontend/src/lib/shell/GlobalSearch.svelte');
const workspaceTree = read('frontend/src/lib/shell/WorkspaceTree.svelte');
const folderAppearanceCore = read('internal/core/workspacetree/appearance.go');
const app = read('frontend/src/App.svelte');
const statusBar = read('frontend/src/lib/shell/StatusBar.svelte');
const compactPluginHost = read('frontend/src/lib/plugin-host/CompactPluginHost.svelte');
const pluginManager = read('frontend/src/lib/plugin-manager/PluginManager.svelte');
const wailsMock = read('frontend/src/lib/test/wails-mock.js');
const syncManifest = JSON.parse(read('../verstak-official-plugins/plugins/sync/plugin.json'));

assertIncludes(
  wailsMock,
  "plugins/files/frontend/src/index.js?raw",
  'E2E should exercise the real official Files frontend bundle',
);
assertIncludes(
  wailsMock,
  "plugins/files/plugin.json",
  'E2E should exercise the real official Files manifest and permissions',
);
assertIncludes(
  wailsMock,
  "assetPath === filesManifest.frontend.entry",
  'E2E Files bundle should be loaded through the entry declared by the real manifest',
);
assertIncludes(
  wailsMock,
  'Promise.resolve(filesSource)',
  'E2E Files asset loading should return the real official Files source',
);
assertExcludes(
  wailsMock,
  'function filesPluginBundle()',
  'E2E should not maintain a divergent synthetic Files implementation',
);

assertIncludes(
  wailsMock,
  "plugins/trash/frontend/src/index.js?raw",
  'E2E should exercise the real official Trash frontend bundle',
);
assertIncludes(
  wailsMock,
  'Promise.resolve(trashSource)',
  'E2E Trash asset loading should return the real official source',
);
assertExcludes(
  wailsMock,
  'function trashPluginBundle()',
  'E2E should not maintain a divergent synthetic Trash implementation',
);
assertExcludes(
  wailsMock,
  'Promise.resolve(trashPluginBundle())',
  'E2E Trash asset loader should never return a synthetic Trash bundle',
);

assertIncludes(
  wailsMock,
  "plugins/search/frontend/src/index.js?raw",
  'E2E should exercise the real official Search frontend bundle',
);
assertIncludes(
  wailsMock,
  'Promise.resolve(searchSource)',
  'E2E Search asset loading should return the real official source',
);
assertExcludes(
  wailsMock,
  'function searchPluginBundle()',
  'E2E should not maintain a divergent synthetic Search implementation',
);
assertExcludes(
  wailsMock,
  'Promise.resolve(searchPluginBundle())',
  'E2E Search asset loader should never return a synthetic Search bundle',
);

assertIncludes(
  wailsMock,
  "plugins/platform-test/frontend/src/index.js?raw",
  'E2E should exercise the real official Platform Test frontend bundle',
);
assertIncludes(
  wailsMock,
  'Promise.resolve(platformTestSource)',
  'E2E Platform Test asset loading should return the real official source',
);
assertExcludes(
  wailsMock,
  'function platformTestBundle()',
  'E2E should not maintain a divergent synthetic Platform Test implementation',
);
assertExcludes(
  wailsMock,
  'Promise.resolve(platformTestBundle())',
  'E2E Platform Test asset loader should never return a synthetic Platform Test bundle',
);

assertIncludes(
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
  "plugins/sync/frontend/dist/index.js?raw",
  'E2E should exercise the real official Sync frontend bundle',
);
assertIncludes(
  wailsMock,
  'Promise.resolve(syncSource)',
  'E2E Sync asset loading should return the real official build output',
);
assertIncludes(
  wailsMock,
  'Promise.resolve(syncStyle)',
  'E2E Sync should serve the stylesheet its manifest declares',
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
if ((wailsMock.match(/vaultPluginState = makeDefaultVaultPluginState\(\);/g) || []).length !== 2) {
  throw new Error('E2E initial and reset vault plugin inventories should come from the canonical official fixture catalog');
}

const officialManifestFixtures = [
  ['platform-test', 'platformTestManifest'],
  ['default-editor', 'defaultEditorManifest'],
  ['file-preview', 'filePreviewManifest'],
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

assertIncludes(
  app,
  '<GlobalSearch />',
  'App should expose global search in the main content header'
);
assertIncludes(
  workspaceTree,
  'App.GetFolderAppearance(',
  'WorkspaceTree should read core-owned folder appearance through Wails',
);
assertIncludes(
  workspaceTree,
  'App.SetFolderAppearance(',
  'WorkspaceTree should save core-owned folder appearance through Wails',
);
assertExcludes(
  workspaceTree,
  "createPluginAPI('verstak.folder-appearance')",
  'WorkspaceTree must not impersonate the retired folder-appearance plugin',
);
assertIncludes(
  folderAppearanceCore,
  'legacyFolderAppearancePluginID = "verstak.folder-appearance"',
  'core folder appearance should preserve legacy plugin data migration',
);
for (const staleContributionHelper of ['GetFolderTreeNodeActions', 'GetWorkspaceTreeNodeActions']) {
  assertExcludes(
    folderAppearanceCore,
    staleContributionHelper,
    `core appearance should not retain abandoned contribution helper ${staleContributionHelper}`,
  );
}
assertIncludes(
  workspaceHost,
  'sortWorkspaceTools',
  'WorkspaceHost should sort the workspace tool tabs'
);
// Where a tool sits belongs to the tool. A table of plugin names in the shell
// makes third-party tools second-class and forces a core edit to reorder.
assertIncludes(
  workspaceHost,
  'tool?.order',
  'WorkspaceHost should take workspace tool order from each plugin manifest'
);
for (const pluginId of ['verstak.notes', 'verstak.todo', 'verstak.activity', 'verstak.secrets']) {
  assertExcludes(
    workspaceHost.slice(0, workspaceHost.indexOf('function filterWorkspaceTools')),
    pluginId,
    `WorkspaceHost should not rank workspace tools by hardcoded plugin id (${pluginId})`
  );
}
// Back and forward are offered to the plugin on screen through the navigation
// registry, not by finding its toolbar buttons in the DOM.
// The window cannot be narrower than 800px, so a viewport media query below
// that never fires in the application. Panels sit inside the content area,
// which is narrower still once the sidebar takes its share, and must respond
// to that rather than to the window.
assertIncludes(
  app,
  'container-name: vt-content',
  'App should make the content area a size container for the panels inside it'
);
for (const panel of [
  'frontend/src/lib/plugin-manager/PluginManager.svelte',
  'frontend/src/lib/plugin-manager/PluginCard.svelte',
  'frontend/src/lib/settings/SettingsWindow.svelte',
  'frontend/src/lib/shell/OverviewSurface.svelte',
]) {
  assertExcludes(
    read(panel),
    '@media (max-width',
    `${panel} should lay itself out from the space it has, not the window width`
  );
}

assertIncludes(
  app,
  'offerNavigation(',
  'App should route navigation requests through the navigation handler registry'
);
assertExcludes(
  app,
  'data-files-action',
  'App should not reach into a specific plugin\'s DOM to navigate'
);

assertIncludes(
  statusBar,
  'data-status-item-id={item.id}',
  'StatusBar should expose stable selectors for plugin-provided status items'
);
assertIncludes(
  statusBar,
  '<CompactPluginHost pluginId={item.pluginId} handler={item.handler}',
  'StatusBar should mount declared compact plugin status handlers'
);
assertIncludes(compactPluginHost, 'data-plugin-status-handler', 'Compact plugin status host should expose a stable mount selector');
if (statusBar.includes('compact status only')) throw new Error('StatusBar should not replace handler contributions with a warning label');

const syncStatus = syncManifest.contributes.statusBarItems.find((item) => item.id === 'verstak.sync.status');
if (!syncStatus || syncStatus.handler !== 'SyncStatusBar') {
  throw new Error('Sync statusBarItem should declare handler "SyncStatusBar"');
}

if (/lastOpenedKey\s*=\s*key;\s*openSettingsFromProps\(activeSettingsPluginId,\s*activeSettingsPanelId\)/s.test(pluginManager)) {
  throw new Error('PluginManager should not mark settings panel as opened before resolving contributions');
}


for (const forbidden of [
  'ReadPluginSettings',
  'ReadPluginDataNDJSON',
  'ListVaultFiles',
  'verstak.browser-inbox',
  'verstak.activity',
  'verstak.journal',
  'verstak.todo',
  'captures:workspace:',
  'captures:global',
  'todos:global',
  'worklog:workspace:',
  'work-session-candidates:workspace:',
]) {
  assertExcludes(
    overviewSurface,
    forbidden,
    `Overview shell must consume provider semantics instead of plugin internals (${forbidden})`,
  );
}
assertIncludes(
  overviewSurface,
  'executePluginCommand(provider.pluginId, provider.handler',
  'Overview shell should consume declared Overview providers through the generic command runtime',
);
if ((overviewSurface.match(/openTool\(item\.actionKind, item\.toolRequest\)/g) || []).length !== 5) {
  throw new Error('Overview shell should preserve provider toolRequest for summary, resume, attention, recent, and resources');
}
assertIncludes(
  overviewSurface,
  'toolRequest: item.action.toolRequest || null',
  'Overview summary normalization should preserve provider toolRequest',
);
assertExcludes(overviewSurface, 'today-', 'Overview shell should not retain Today-era internal CSS names');
assertExcludes(workspaceHost, 'TodaySurface', 'WorkspaceHost should use the final OverviewSurface component name');
assertIncludes(
  workspaceHost,
  'findWorkspaceItem(workspaceItemId)',
  'WorkspaceHost should resolve Overview navigation by exact workspace item id',
);
assertExcludes(
  workspaceHost,
  "kind === 'browser-inbox'",
  'WorkspaceHost should not special-case Browser navigation',
);
assertExcludes(
  workspaceHost,
  'text.includes(kind)',
  'WorkspaceHost should not guess a workspace tool from arbitrary title/id substrings',
);

for (const forbidden of [
  'ReadPluginSettings',
  'ReadVaultTextFile',
  'ListVaultFiles',
  'indexPluginSettings',
  'verstak.browser-inbox',
  'verstak.activity',
  'verstak.journal',
  "category === 'files'",
  "category === 'folders'",
  '__filesHistoryByWorkspace',
  'App.GetWorkspaceTree()',
]) {
  assertExcludes(
    globalSearch,
    forbidden,
    `GlobalSearch shell must consume generic Search providers instead of domain storage (${forbidden})`,
  );
}
assertIncludes(
  globalSearch,
  'App.GetWorkspaceTreeV2()',
  'GlobalSearch should index Deals from the semantic workspace tree',
);
assertIncludes(
  globalSearch,
  'collectWorkspaceNodes',
  'GlobalSearch should recursively index Deals nested under semantic folders',
);

assertIncludes(
  globalSearch,
  'item?.categoryLabel || provider?.label',
  'GlobalSearch should render provider-owned category labels without interpreting provider category ids',
);
assertIncludes(
  globalSearch,
  'enabledPluginIds.has(provider?.pluginId)',
  'GlobalSearch should query only providers from enabled loaded/degraded plugins',
);

assertIncludes(
  globalSearch,
  'contributions.searchProviders || []',
  'GlobalSearch should discover Search providers from contribution metadata',
);
assertIncludes(
  globalSearch,
  'executePluginCommand(provider.pluginId, provider.handler',
  'GlobalSearch should execute declared Search providers through the generic command runtime',
);
assertIncludes(
  globalSearch,
  'workspaceItemId:',
  'GlobalSearch should navigate workspace tools by exact workspace item id',
);
assertExcludes(
  globalSearch,
  'toolKind:',
  'GlobalSearch should not encode semantic workspace tool kinds',
);
assertExcludes(
  globalSearch,
  'detail: { kind:',
  'GlobalSearch should not dispatch legacy kind-based workspace navigation',
);

console.log('shell source contract smoke passed');
