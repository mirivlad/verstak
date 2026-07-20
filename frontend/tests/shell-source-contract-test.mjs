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
  'toolOrder',
  'WorkspaceHost should define usage-based workspace tool ordering'
);
assertIncludes(
  workspaceHost,
  'sortWorkspaceTools',
  'WorkspaceHost should sort workspace tools by expected usage'
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
