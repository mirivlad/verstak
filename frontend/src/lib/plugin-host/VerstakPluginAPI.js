import * as App from '../../../wailsjs/go/api/App';

window.__VERSTAK_PLUGIN_REGISTRY__ = window.__VERSTAK_PLUGIN_REGISTRY__ || {};
window.__VERSTAK_EVENT_HANDLERS__ = window.__VERSTAK_EVENT_HANDLERS__ || {};
window.__VERSTAK_COMMAND_HANDLERS__ = window.__VERSTAK_COMMAND_HANDLERS__ || {};

if (!window.VerstakPluginRegister) {
  window.VerstakPluginRegister = function(pluginId, bundle) {
    if (!pluginId || !bundle || !bundle.components) {
      console.error('[VerstakPluginRegister] invalid registration:', pluginId);
      return;
    }
    console.log('[VerstakPluginRegister] registered:', pluginId, Object.keys(bundle.components));
    window.__VERSTAK_PLUGIN_REGISTRY__[pluginId] = bundle.components;
  };
}

function unpack(result) {
  if (Array.isArray(result) && result.length === 2 && (typeof result[1] === 'string' || result[1] == null)) {
    return [result[0], result[1] || ''];
  }
  return [result, ''];
}

async function callBackend(pluginId, label, fn) {
  try {
    const [value, err] = unpack(await fn());
    if (err) {
      throw new Error(err);
    }
    return value;
  } catch (e) {
    const message = e && e.message ? e.message : String(e);
    throw new Error('[plugin:' + pluginId + '] ' + label + ' failed: ' + message);
  }
}

async function callBackendErrorString(pluginId, label, fn) {
  try {
    const err = await fn();
    if (err) {
      throw new Error(err);
    }
  } catch (e) {
    const message = e && e.message ? e.message : String(e);
    throw new Error('[plugin:' + pluginId + '] ' + label + ' failed: ' + message);
  }
}

function getEventHandlers(eventName) {
  if (!window.__VERSTAK_EVENT_HANDLERS__[eventName]) {
    window.__VERSTAK_EVENT_HANDLERS__[eventName] = [];
  }
  return window.__VERSTAK_EVENT_HANDLERS__[eventName];
}

function dispatchLocalEvent(pluginId, eventName, payload) {
  const event = {
    name: eventName,
    pluginId: pluginId,
    payload: payload || {},
    timestamp: new Date().toISOString()
  };
  const handlers = getEventHandlers(eventName).slice();
  handlers.forEach(function(handler) {
    try {
      handler(event);
    } catch (e) {
      console.error('[VerstakPluginAPI] event handler error:', e);
    }
  });
}

function commandKey(pluginId, commandId) {
  return pluginId + ':' + commandId;
}

export async function executePluginCommand(pluginId, cmdId, args) {
  if (!pluginId) {
    throw new Error('executePluginCommand requires pluginId');
  }
  if (!cmdId) {
    throw new Error('executePluginCommand requires command id');
  }
  const declared = await callBackend(pluginId, 'commands.execute(' + cmdId + ')', function() {
    return App.ExecutePluginCommand(pluginId, cmdId, args || {});
  });
  const handler = window.__VERSTAK_COMMAND_HANDLERS__[commandKey(pluginId, cmdId)];
  if (!handler) {
    throw new Error('[plugin:' + pluginId + '] commands.execute(' + cmdId + ') failed: declared-but-unhandled');
  }
  const result = await handler(args || {}, declared);
  return {
    status: 'handled',
    pluginId: pluginId,
    commandId: cmdId,
    result: result
  };
}

export function createPluginAPI(pluginId) {
  if (!pluginId) {
    throw new Error('createPluginAPI requires pluginId');
  }

  const cleanups = [];
  let disposed = false;

  function assertActive(label) {
    if (disposed) {
      throw new Error('[plugin:' + pluginId + '] ' + label + ' failed: API disposed');
    }
  }

  function trackCleanup(fn) {
    cleanups.push(fn);
    return function untrackAndRun() {
      const idx = cleanups.indexOf(fn);
      if (idx !== -1) {
        cleanups.splice(idx, 1);
      }
      fn();
    };
  }

  return {
    pluginId: pluginId,

    capabilities: {
      has: async function(capId) {
        const info = await callBackend(pluginId, 'capabilities.has(' + capId + ')', function() {
          return App.GetPluginCapability(pluginId, capId);
        });
        return !!(info && info.available);
      },
      get: function(capId) {
        return callBackend(pluginId, 'capabilities.get(' + capId + ')', function() {
          return App.GetPluginCapability(pluginId, capId);
        });
      },
      list: function() {
        return callBackend(pluginId, 'capabilities.list', function() {
          return App.ListPluginCapabilities(pluginId);
        });
      }
    },

    events: {
      publish: async function(type, payload) {
        await callBackendErrorString(pluginId, 'events.publish(' + type + ')', function() {
          return App.PublishPluginEvent(pluginId, type, payload || {});
        });
        dispatchLocalEvent(pluginId, type, payload || {});
      },
      subscribe: function(type, handler) {
        assertActive('events.subscribe(' + type + ')');
        if (typeof handler !== 'function') {
          throw new Error('events.subscribe requires a handler function');
        }
        return callBackendErrorString(pluginId, 'events.subscribe(' + type + ')', function() {
          return App.SubscribePluginEvent(pluginId, type);
        }).then(function() {
          const handlers = getEventHandlers(type);
          handlers.push(handler);
          return trackCleanup(function unsubscribe() {
            const current = getEventHandlers(type);
            window.__VERSTAK_EVENT_HANDLERS__[type] = current.filter(function(item) {
              return item !== handler;
            });
          });
        });
      }
    },

    settings: {
      read: async function(key) {
        assertActive('settings.read');
        const settings = await callBackend(pluginId, 'settings.read', function() {
          return App.ReadPluginSettings(pluginId);
        });
        if (!key) {
          return settings || {};
        }
        return settings ? settings[key] : undefined;
      },
      write: async function(key, value) {
        assertActive('settings.write(' + key + ')');
        if (!key) {
          throw new Error('settings.write requires a key');
        }
        const settings = await this.read();
        settings[key] = value;
        await callBackendErrorString(pluginId, 'settings.write(' + key + ')', function() {
          return App.WritePluginSettings(pluginId, settings);
        });
        return settings;
      },
      writeAll: function(settings) {
        assertActive('settings.writeAll');
        return callBackendErrorString(pluginId, 'settings.writeAll', function() {
          return App.WritePluginSettings(pluginId, settings || {});
        });
      }
    },

    files: {
      list: function(relativeDir) {
        assertActive('files.list');
        return callBackend(pluginId, 'files.list(' + (relativeDir || '') + ')', function() {
          return App.ListVaultFiles(pluginId, relativeDir || '');
        });
      },
      metadata: function(relativePath) {
        assertActive('files.metadata(' + relativePath + ')');
        return callBackend(pluginId, 'files.metadata(' + relativePath + ')', function() {
          return App.GetVaultFileMetadata(pluginId, relativePath);
        });
      },
      readText: function(relativePath) {
        assertActive('files.readText(' + relativePath + ')');
        return callBackend(pluginId, 'files.readText(' + relativePath + ')', function() {
          return App.ReadVaultTextFile(pluginId, relativePath);
        });
      },
      writeText: function(relativePath, content, options) {
        assertActive('files.writeText(' + relativePath + ')');
        return callBackendErrorString(pluginId, 'files.writeText(' + relativePath + ')', function() {
          return App.WriteVaultTextFile(pluginId, relativePath, String(content == null ? '' : content), options || {});
        });
      },
      createFolder: function(relativePath) {
        assertActive('files.createFolder(' + relativePath + ')');
        return callBackendErrorString(pluginId, 'files.createFolder(' + relativePath + ')', function() {
          return App.CreateVaultFolder(pluginId, relativePath);
        });
      },
      move: function(fromRelativePath, toRelativePath, options) {
        assertActive('files.move(' + fromRelativePath + ')');
        return callBackendErrorString(pluginId, 'files.move(' + fromRelativePath + ')', function() {
          return App.MoveVaultPath(pluginId, fromRelativePath, toRelativePath, options || {});
        });
      },
      trash: function(relativePath) {
        assertActive('files.trash(' + relativePath + ')');
        return callBackend(pluginId, 'files.trash(' + relativePath + ')', function() {
          return App.TrashVaultPath(pluginId, relativePath);
        });
      }
    },

    sync: {
      status: function() {
        assertActive('sync.status');
        return callBackend(pluginId, 'sync.status', function() {
          return App.PluginSyncStatus(pluginId);
        });
      },
      configure: function(serverURL, username, password) {
        assertActive('sync.configure');
        return callBackendErrorString(pluginId, 'sync.configure', function() {
          return App.PluginSyncConfigure(pluginId, serverURL || '', username || '', password || '');
        });
      },
      disconnect: function() {
        assertActive('sync.disconnect');
        return callBackendErrorString(pluginId, 'sync.disconnect', function() {
          return App.PluginSyncDisconnect(pluginId);
        });
      },
      testConnection: function(serverURL, username, password) {
        assertActive('sync.testConnection');
        return callBackendErrorString(pluginId, 'sync.testConnection', function() {
          return App.PluginSyncTestConnection(pluginId, serverURL || '', username || '', password || '');
        });
      },
      setInterval: function(minutes) {
        assertActive('sync.setInterval');
        return callBackendErrorString(pluginId, 'sync.setInterval', function() {
          return App.PluginSyncSetInterval(pluginId, Number(minutes) || 0);
        });
      },
      now: function() {
        assertActive('sync.now');
        return callBackend(pluginId, 'sync.now', function() {
          return App.PluginSyncNow(pluginId);
        });
      }
    },

    workbench: {
      openResource: async function(request) {
        assertActive('workbench.openResource');
        const result = await callBackend(pluginId, 'workbench.openResource', function() {
          return App.OpenWorkbenchResource(pluginId, request || {});
        });
        window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
        return result;
      },
      editResource: async function(request) {
        assertActive('workbench.editResource');
        const result = await callBackend(pluginId, 'workbench.editResource', function() {
          return App.EditWorkbenchResource(pluginId, request || {});
        });
        window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
        return result;
      }
    },

    commands: {
      register: function(cmdId, handler) {
        assertActive('commands.register(' + cmdId + ')');
        if (!cmdId) {
          throw new Error('commands.register requires a command id');
        }
        if (typeof handler !== 'function') {
          throw new Error('commands.register requires a handler function');
        }
        return callBackend(pluginId, 'commands.register(' + cmdId + ')', function() {
          return App.ExecutePluginCommand(pluginId, cmdId, { validateOnly: true });
        }).then(function() {
          const key = commandKey(pluginId, cmdId);
          window.__VERSTAK_COMMAND_HANDLERS__[key] = handler;
          return trackCleanup(function unregisterCommand() {
            if (window.__VERSTAK_COMMAND_HANDLERS__[key] === handler) {
              delete window.__VERSTAK_COMMAND_HANDLERS__[key];
            }
          });
        });
      },
      execute: async function(cmdId, args) {
        assertActive('commands.execute(' + cmdId + ')');
        return executePluginCommand(pluginId, cmdId, args || {});
      }
    },

    dispose: function() {
      if (disposed) return;
      disposed = true;
      while (cleanups.length > 0) {
        const cleanup = cleanups.pop();
        try {
          cleanup();
        } catch (e) {
          console.error('[VerstakPluginAPI] cleanup error:', e);
        }
      }
    }
  };
}

window.createPluginAPI = createPluginAPI;
window.VerstakPluginAPI = createPluginAPI;
