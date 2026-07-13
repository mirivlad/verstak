import * as App from '../../../wailsjs/go/api/App';
import { i18n } from '../i18n/index.js';

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

function dispatchBackendEvent(event) {
  if (!event || !event.name) return;
  const handlers = getEventHandlers(event.name).slice();
  handlers.forEach(function(handler) {
    try {
      handler(event);
    } catch (e) {
      console.error('[VerstakPluginAPI] backend event handler error:', e);
    }
  });
}

window.__VERSTAK_DISPATCH_BACKEND_EVENT__ = dispatchBackendEvent;

if (!window.__VERSTAK_BACKEND_EVENT_BRIDGE__ && window.runtime && typeof window.runtime.EventsOnMultiple === 'function') {
  window.__VERSTAK_BACKEND_EVENT_BRIDGE__ = window.runtime.EventsOnMultiple('verstak:plugin-event', dispatchBackendEvent, -1);
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

    i18n: {
      getLocale: function() {
        assertActive('i18n.getLocale');
        return i18n.getLocale();
      },
      t: function(key, params, fallback) {
        assertActive('i18n.t(' + key + ')');
        return i18n.translatePlugin(pluginId, key, params, fallback);
      },
      onDidChangeLocale: function(listener) {
        assertActive('i18n.onDidChangeLocale');
        if (typeof listener !== 'function') {
          throw new Error('i18n.onDidChangeLocale requires a listener function');
        }
        return trackCleanup(i18n.subscribe(listener));
      }
    },

    ui: {
      openSettings: function(panelId) {
        assertActive('ui.openSettings');
        window.dispatchEvent(new CustomEvent('verstak:open-settings', {
          detail: { pluginId: pluginId, panelId: panelId || '' }
        }));
      }
    },

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
        if (!window.__VERSTAK_BACKEND_EVENT_BRIDGE__) {
          dispatchLocalEvent(pluginId, type, payload || {});
        }
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

    notifications: {
      replace: function(items) {
        assertActive('notifications.replace');
        if (!Array.isArray(items)) {
          throw new Error('notifications.replace requires an array');
        }
        return callBackendErrorString(pluginId, 'notifications.replace', function() {
          return App.ReplacePluginNotifications(pluginId, items);
        });
      },
      clear: function() {
        assertActive('notifications.clear');
        return callBackendErrorString(pluginId, 'notifications.clear', function() {
          return App.ClearPluginNotifications(pluginId);
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
        await callBackendErrorString(pluginId, 'settings.write(' + key + ')', function() {
          return App.WritePluginSetting(pluginId, key, value);
        });
        return this.read();
      },
      writeAll: function(settings) {
        assertActive('settings.writeAll');
        return callBackendErrorString(pluginId, 'settings.writeAll', function() {
          return App.WritePluginSettings(pluginId, settings || {});
        });
      }
    },

    storage: {
      data: {
        read: function(name) {
          assertActive('storage.data.read(' + name + ')');
          if (!name) {
            throw new Error('storage.data.read requires a name');
          }
          return callBackend(pluginId, 'storage.data.read(' + name + ')', function() {
            return App.ReadPluginDataJSON(pluginId, name);
          }).then(function(data) {
            return data || {};
          });
        },
        readNDJSON: function(name) {
          assertActive('storage.data.readNDJSON(' + name + ')');
          if (!name) {
            throw new Error('storage.data.readNDJSON requires a name');
          }
          return callBackend(pluginId, 'storage.data.readNDJSON(' + name + ')', function() {
            return App.ReadPluginDataNDJSON(pluginId, name);
          }).then(function(records) {
            return Array.isArray(records) ? records : [];
          });
        },
        writeNDJSON: function(name, records) {
          assertActive('storage.data.writeNDJSON(' + name + ')');
          if (!name) {
            throw new Error('storage.data.writeNDJSON requires a name');
          }
          return callBackendErrorString(pluginId, 'storage.data.writeNDJSON(' + name + ')', function() {
            return App.WritePluginDataNDJSON(pluginId, name, Array.isArray(records) ? records : []);
          });
        },
        write: function(name, data) {
          assertActive('storage.data.write(' + name + ')');
          if (!name) {
            throw new Error('storage.data.write requires a name');
          }
          return callBackendErrorString(pluginId, 'storage.data.write(' + name + ')', function() {
            return App.WritePluginDataJSON(pluginId, name, data || {});
          });
        }
      }
    },

    secrets: {
      status: function() {
        assertActive('secrets.status');
        return callBackend(pluginId, 'secrets.status', function() {
          return App.PluginSecretsStatus(pluginId);
        });
      },
      unlock: function(masterPassword) {
        assertActive('secrets.unlock');
        return callBackendErrorString(pluginId, 'secrets.unlock', function() {
          return App.PluginSecretsUnlock(pluginId, String(masterPassword == null ? '' : masterPassword));
        });
      },
      list: function() {
        assertActive('secrets.list');
        return callBackend(pluginId, 'secrets.list', function() {
          return App.PluginSecretsList(pluginId);
        }).then(function(records) {
          return Array.isArray(records) ? records : [];
        });
      },
      read: function(secretId) {
        assertActive('secrets.read(' + secretId + ')');
        if (!secretId) {
          throw new Error('secrets.read requires a secret id');
        }
        return callBackend(pluginId, 'secrets.read(' + secretId + ')', function() {
          return App.PluginSecretsRead(pluginId, secretId);
        });
      },
      write: function(record) {
        assertActive('secrets.write');
        return callBackend(pluginId, 'secrets.write', function() {
          return App.PluginSecretsWrite(pluginId, record || {});
        });
      },
      delete: function(secretId) {
        assertActive('secrets.delete(' + secretId + ')');
        if (!secretId) {
          throw new Error('secrets.delete requires a secret id');
        }
        return callBackendErrorString(pluginId, 'secrets.delete(' + secretId + ')', function() {
          return App.PluginSecretsDelete(pluginId, secretId);
        });
      },
      copyLink: function(secretId) {
        assertActive('secrets.copyLink(' + secretId + ')');
        if (!secretId) {
          throw new Error('secrets.copyLink requires a secret id');
        }
        return callBackend(pluginId, 'secrets.copyLink(' + secretId + ')', function() {
          return App.PluginSecretsCopyLink(pluginId, secretId);
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
      readBytes: function(relativePath) {
        assertActive('files.readBytes(' + relativePath + ')');
        return callBackend(pluginId, 'files.readBytes(' + relativePath + ')', function() {
          return App.ReadVaultFileBytes(pluginId, relativePath);
        });
      },
      writeText: function(relativePath, content, options) {
        assertActive('files.writeText(' + relativePath + ')');
        return callBackendErrorString(pluginId, 'files.writeText(' + relativePath + ')', function() {
          return App.WriteVaultTextFile(pluginId, relativePath, String(content == null ? '' : content), options || {});
        });
      },
      writeBytes: function(relativePath, dataBase64, options) {
        assertActive('files.writeBytes(' + relativePath + ')');
        return callBackendErrorString(pluginId, 'files.writeBytes(' + relativePath + ')', function() {
          return App.WriteVaultFileBytes(pluginId, relativePath, String(dataBase64 == null ? '' : dataBase64), options || {});
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
      },
      listTrash: function() {
        assertActive('files.listTrash');
        return callBackend(pluginId, 'files.listTrash', function() {
          return App.ListVaultTrash(pluginId);
        });
      },
      restoreTrash: function(trashId, options) {
        assertActive('files.restoreTrash(' + trashId + ')');
        return callBackend(pluginId, 'files.restoreTrash(' + trashId + ')', function() {
          return App.RestoreVaultTrash(pluginId, trashId, options || {});
        });
      },
      deleteTrash: function(trashId) {
        assertActive('files.deleteTrash(' + trashId + ')');
        return callBackendErrorString(pluginId, 'files.deleteTrash(' + trashId + ')', function() {
          return App.DeleteVaultTrash(pluginId, trashId);
        });
      },
      openExternal: function(relativePath) {
        assertActive('files.openExternal(' + relativePath + ')');
        return callBackendErrorString(pluginId, 'files.openExternal(' + relativePath + ')', function() {
          return App.OpenVaultPathExternal(pluginId, relativePath);
        });
      },
      openURL: function(url) {
        assertActive('files.openURL');
        return callBackendErrorString(pluginId, 'files.openURL', function() {
          return App.OpenExternalURL(pluginId, String(url == null ? '' : url));
        });
      },
      showInFolder: function(relativePath) {
        assertActive('files.showInFolder(' + relativePath + ')');
        return callBackendErrorString(pluginId, 'files.showInFolder(' + relativePath + ')', function() {
          return App.ShowVaultPathInFolder(pluginId, relativePath);
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
      resetKey: function() {
        assertActive('sync.resetKey');
        return callBackendErrorString(pluginId, 'sync.resetKey', function() {
          return App.PluginSyncResetKey(pluginId);
        });
      },
      now: function() {
        assertActive('sync.now');
        return callBackend(pluginId, 'sync.now', function() {
          return App.PluginSyncNow(pluginId);
        });
      }
    },

    browserReceiver: {
      pairing: function() {
        assertActive('browserReceiver.pairing');
        return callBackend(pluginId, 'browserReceiver.pairing', function() {
          return App.PluginBrowserReceiverPairing(pluginId);
        });
      },
      rotateToken: function() {
        assertActive('browserReceiver.rotateToken');
        return callBackend(pluginId, 'browserReceiver.rotateToken', function() {
          return App.PluginRotateBrowserReceiverToken(pluginId);
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

    contributions: {
      list: async function(point) {
        assertActive('contributions.list');
        const summary = await callBackend(pluginId, 'contributions.list', function() {
          return App.GetContributions();
        });
        if (!point) {
          return summary || {};
        }
        return Array.isArray((summary || {})[point]) ? summary[point] : [];
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
      },
      executeFor: async function(targetPluginId, cmdId, args) {
        assertActive('commands.executeFor(' + targetPluginId + ':' + cmdId + ')');
        return executePluginCommand(targetPluginId, cmdId, args || {});
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
