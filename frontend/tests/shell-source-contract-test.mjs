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
const statusBar = read('frontend/src/lib/shell/StatusBar.svelte');
const pluginManager = read('frontend/src/lib/plugin-manager/PluginManager.svelte');
const syncManifest = JSON.parse(read('../verstak-official-plugins/plugins/sync/plugin.json'));

assertIncludes(
  workspaceHost,
  'data-workspace-search',
  'WorkspaceHost should expose a stable workspace header search slot'
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
  "import PluginBundleHost from '../plugin-host/PluginBundleHost.svelte';",
  'StatusBar should mount plugin-provided status bar components'
);
assertIncludes(
  statusBar,
  'componentId={item.handler}',
  'StatusBar should use statusBarItems.handler as the component id'
);

const syncStatus = syncManifest.contributes.statusBarItems.find((item) => item.id === 'verstak.sync.status');
if (!syncStatus || syncStatus.handler !== 'SyncStatusBar') {
  throw new Error('Sync statusBarItem should declare handler "SyncStatusBar"');
}

if (/lastOpenedKey\s*=\s*key;\s*openSettingsFromProps\(activeSettingsPluginId,\s*activeSettingsPanelId\)/s.test(pluginManager)) {
  throw new Error('PluginManager should not mark settings panel as opened before resolving contributions');
}

console.log('shell source contract smoke passed');
