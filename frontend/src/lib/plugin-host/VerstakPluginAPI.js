// VerstakPluginAPI is the restricted API passed to plugin frontend bundles.
// Plugins do NOT get direct access to Wails bridge — only what's exposed here.
// All methods are stubs or limited implementations.

(function() {
  // Store registered components per plugin
  window.__VERSTAK_PLUGIN_REGISTRY__ = window.__VERSTAK_PLUGIN_REGISTRY__ || {};

  // Original register function
  const origRegister = window.VerstakPluginRegister;
  if (origRegister) {
    // Already defined — don't override
    return;
  }

  window.VerstakPluginRegister = function(pluginId, bundle) {
    if (!pluginId || !bundle || !bundle.components) {
      console.error('[VerstakPluginRegister] invalid registration:', pluginId);
      return;
    }
    console.log('[VerstakPluginRegister] registered:', pluginId, Object.keys(bundle.components));
    window.__VERSTAK_PLUGIN_REGISTRY__[pluginId] = bundle.components;
  };

  // Create the restricted API object for a plugin host context
  window.VerstakPluginAPI = function(pluginId) {
    return {
      pluginId: pluginId,

      capabilities: {
        has: function(capId) {
          // planned: query backend cap registry
          console.log('[plugin:' + pluginId + '] capabilities.has(' + capId + ') — stub');
          return false;
        }
      },

      events: {
        publish: function(type, payload) {
          console.log('[plugin:' + pluginId + '] event publish:', type, payload);
          // planned: actual event bus bridge
        },
        subscribe: function(type, handler) {
          console.log('[plugin:' + pluginId + '] event subscribe:', type, '(stub)');
          // planned: actual event bus bridge
        }
      },

      settings: {
        read: function(key) {
          console.log('[plugin:' + pluginId + '] settings.read(' + key + ') — stub');
          return null;
        },
        write: function(key, value) {
          console.log('[plugin:' + pluginId + '] settings.write(' + key + ',', value, ') — stub');
          // planned: backend storage namespace
        }
      },

      commands: {
        execute: function(cmdId, args) {
          console.log('[plugin:' + pluginId + '] commands.execute(' + cmdId + ') — stub');
          // planned: command execution
        }
      }
    };
  };
})();
