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
const app = read('frontend/src/App.svelte');
const statusBar = read('frontend/src/lib/shell/StatusBar.svelte');
const compactPluginHost = read('frontend/src/lib/plugin-host/CompactPluginHost.svelte');
const pluginManager = read('frontend/src/lib/plugin-manager/PluginManager.svelte');
const syncManifest = JSON.parse(read('../verstak-official-plugins/plugins/sync/plugin.json'));

assertIncludes(
  app,
  '<GlobalSearch />',
  'App should expose global search in the main content header'
);
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
  'frontend/src/lib/shell/TodaySurface.svelte',
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

console.log('shell source contract smoke passed');
