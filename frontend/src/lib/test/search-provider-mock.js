// Test-only compatibility layer for Search provider contributions.
//
// The long-lived Wails mock predates background Search providers and still
// contains a synthetic SearchView bundle. Provider behavior is important enough
// that tests should exercise the real official frontend bundle and manifests
// instead of growing another copy of Search semantics here.
import searchSource from '../../../../../verstak-official-plugins/plugins/search/frontend/src/index.js?raw';
import searchManifest from '../../../../../verstak-official-plugins/plugins/search/plugin.json';
import activityManifest from '../../../../../verstak-official-plugins/plugins/activity/plugin.json';
import journalManifest from '../../../../../verstak-official-plugins/plugins/journal/plugin.json';
import browserManifest from '../../../../../verstak-official-plugins/plugins/browser-inbox/plugin.json';

const app = window.go?.api?.App;
if (!app) {
  throw new Error('search-provider-mock requires the Wails mock bridge');
}

const manifests = [searchManifest, activityManifest, journalManifest, browserManifest];
const originalGetContributions = app.GetContributions.bind(app);
const originalGetPluginAssetContent = app.GetPluginAssetContent.bind(app);
const originalExecutePluginCommand = app.ExecutePluginCommand.bind(app);

function cleanVaultRelativePath(value) {
  const raw = String(value || '');
  if (!raw || raw.includes('\\') || raw.startsWith('/') || raw.split('/').some((part) => part === '..')) return null;
  const parts = raw.split('/').filter(Boolean);
  if (parts[0] === '.verstak') return null;
  return parts.join('/');
}

app.PluginResolveWorkspacePath = async function pluginResolveWorkspacePath(pluginId, relativePath) {
  const path = cleanVaultRelativePath(relativePath);
  if (!path) return [null, 'invalid relative path'];
  const snapshot = await app.GetWorkspaceTreeV2();
  let best = null;
  function walk(nodes) {
    (nodes || []).forEach((node) => {
      const root = String(node?.path || '').replace(/^\/+|\/+$/g, '');
      if (node?.kind === 'workspace' && root && (path === root || path.startsWith(`${root}/`))) {
        if (!best || root.length > best.root.length) best = { node, root };
      }
      walk(node?.children || []);
    });
  }
  walk(snapshot?.roots || []);
  if (!best) return [{ found: false }, ''];
  return [{
    found: true,
    workspaceId: best.node.id || '',
    workspaceName: best.node.name || '',
    workspaceRootPath: best.root,
    relativePath: path === best.root ? '' : path.slice(best.root.length + 1),
  }, ''];
};

function searchProviderRows(manifest) {
  const contributes = manifest?.contributes || {};
  return (contributes.searchProviders || []).map((provider) => ({
    ...provider,
    pluginId: manifest.id,
  }));
}

function commandFor(pluginId, commandId) {
  const manifest = manifests.find((candidate) => candidate.id === pluginId);
  return (manifest?.contributes?.commands || []).find((command) => command.id === commandId) || null;
}

app.GetContributions = async function getContributionsWithRealSearchProviders() {
  const summary = await originalGetContributions();
  const providers = Array.isArray(summary?.searchProviders) ? summary.searchProviders.slice() : [];
  const seen = new Set(providers.map((provider) => `${provider.pluginId}:${provider.id}`));
  manifests.flatMap(searchProviderRows).forEach((provider) => {
    const key = `${provider.pluginId}:${provider.id}`;
    if (seen.has(key)) return;
    seen.add(key);
    providers.push(provider);
  });
  return { ...(summary || {}), searchProviders: providers };
};

app.GetPluginAssetContent = function getPluginAssetContentWithRealSearchBundle(pluginId, assetPath) {
  if (pluginId === searchManifest.id) {
    return Promise.resolve(searchSource);
  }
  return originalGetPluginAssetContent(pluginId, assetPath);
};

app.ExecutePluginCommand = function executePluginCommandWithRealSearchDeclarations(pluginId, commandId, args) {
  const command = commandFor(pluginId, commandId);
  if (command) {
    return Promise.resolve([{
      status: 'declared',
      pluginId,
      commandId,
      handler: command.handler,
      args: args || {},
    }, '']);
  }
  return originalExecutePluginCommand(pluginId, commandId, args);
};
