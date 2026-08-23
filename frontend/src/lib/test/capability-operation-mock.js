// Test-only adapter for the provider-independent capability operation bridge.
// It patches the existing Wails mock instead of teaching the large fixture a
// second implementation of plugin runtime state.
const app = window.go?.api?.App;
const helpers = window.__wailsMock;

if (app && helpers) {
  app.ResolvePluginCapabilityOperation = async function(pluginId, capabilityName, operation) {
    const consumer = helpers.getPluginState(pluginId);
    const manifest = consumer && consumer.manifest;
    const declared = ((manifest && manifest.requires) || []).concat(
      (manifest && manifest.optionalRequires) || []
    );
    if (declared.indexOf(capabilityName) === -1) {
      return [{}, 'capability dependency not declared'];
    }

    const pluginResult = await app.GetPlugins();
    const plugins = Array.isArray(pluginResult) && Array.isArray(pluginResult[0])
      ? pluginResult[0]
      : pluginResult;
    const provider = (plugins || []).find(function(candidate) {
      const state = candidate && candidate.manifest;
      return candidate && candidate.enabled &&
        (candidate.status === 'loaded' || candidate.status === 'degraded') &&
        ((state && state.provides) || []).indexOf(capabilityName) !== -1;
    });
    if (!provider) return [{}, 'capability not available'];

    const operations = (provider.manifest.capabilityOperations || {})[capabilityName] || {};
    const commandId = operations[operation];
    if (!commandId) return [{}, 'capability operation not available'];
    return [{
      capability: capabilityName,
      operation,
      pluginId: provider.manifest.id,
      commandId
    }, ''];
  };
}
