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
