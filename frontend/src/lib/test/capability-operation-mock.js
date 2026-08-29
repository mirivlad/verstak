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

  app.ResolveDealCapabilityOperation = async function(pluginId, capabilityName, operation, request) {
    const allowed = {
      'verstak/notes/v2': ['list', 'create', 'open'],
      'verstak/files/v2': ['list', 'create', 'open'],
      'verstak/todo/v2': ['list', 'create', 'setStatus'],
      'verstak/activity/v2': ['list', 'search'],
    };
    if (!allowed[capabilityName] || allowed[capabilityName].indexOf(operation) === -1) {
      return [{}, 'Deal capability operation is not available'];
    }
    const scope = request && request.scope;
    if (!scope || scope.kind !== 'deal') return [{}, 'DealScope.kind must be deal'];
    if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(String(scope.workspaceId || ''))) {
      return [{}, 'DealScope.workspaceId must be a UUID'];
    }
    const consumer = helpers.getPluginState(pluginId);
    if (!consumer || !consumer.manifest || consumer.manifest.apiVersion !== '0.1.0') {
      return [{}, 'Deal capability host API is incompatible'];
    }
    const resolved = await app.ResolvePluginCapabilityOperation(pluginId, capabilityName, operation);
    const value = Array.isArray(resolved) ? resolved[0] : resolved;
    const err = Array.isArray(resolved) ? resolved[1] : '';
    if (err) return [value, err];
    const provider = helpers.getPluginState(value.pluginId);
    if (!provider || !provider.manifest || provider.manifest.apiVersion !== '0.1.0') {
      return [{}, 'Deal capability host API is incompatible'];
    }
    return [value, ''];
  };
}
