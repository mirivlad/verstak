import * as App from '../../../wailsjs/go/api/App';
import { i18n } from '../i18n/index.js';
import { matchesShortcut } from '../ui/shortcuts.js';
import { registerNavigationHandler } from '../shell/navigation-handlers.js';

window.__VERSTAK_PLUGIN_REGISTRY__ = window.__VERSTAK_PLUGIN_REGISTRY__ || {};
window.__VERSTAK_PLUGIN_BUNDLES__ = window.__VERSTAK_PLUGIN_BUNDLES__ || {};
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
    // The whole bundle is kept, not just its components: activate() lives
    // beside them and has to survive the moment of registration.
    window.__VERSTAK_PLUGIN_BUNDLES__[pluginId] = bundle;
  };
}

function unpack(result) {
  if (Array.isArray(result) && result.length === 2 && (typeof result[1] === 'string' || result[1] == null)) {
    return [result[0], result[1] || ''];
  }
  return [result, ''];
}

const dealWorkspaceIdPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
const dealScopedProviderCapabilities = new Set([
  'verstak/notes/v2',
  'verstak/files/v2',
  'verstak/todo/v2',
  'verstak/activity/v2',
]);

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

window.__VERSTAK_IMPORT_PROGRESS_HANDLERS__ = window.__VERSTAK_IMPORT_PROGRESS_HANDLERS__ || {};

function importProgressKey(pluginId, sourceHandle) {
  return pluginId + ':' + sourceHandle;
}

function dispatchImportProgress(progress) {
  if (!progress || !progress.pluginId || !progress.sourceHandle) return;
  const key = importProgressKey(progress.pluginId, progress.sourceHandle);
  const handlers = (window.__VERSTAK_IMPORT_PROGRESS_HANDLERS__[key] || []).slice();
  handlers.forEach(function(handler) {
    try {
      handler(progress);
    } catch (e) {
      console.error('[VerstakPluginAPI] import progress handler error:', e);
    }
  });
}

function trackImportProgress(pluginId, sourceHandle, listener) {
  const key = importProgressKey(pluginId, sourceHandle);
  const handlers = window.__VERSTAK_IMPORT_PROGRESS_HANDLERS__[key] || [];
  handlers.push(listener);
  window.__VERSTAK_IMPORT_PROGRESS_HANDLERS__[key] = handlers;
  let active = true;
  return function unsubscribeImportProgress() {
    if (!active) return;
    active = false;
    const current = window.__VERSTAK_IMPORT_PROGRESS_HANDLERS__[key] || [];
    const remaining = current.filter(function(item) { return item !== listener; });
    if (remaining.length > 0) {
      window.__VERSTAK_IMPORT_PROGRESS_HANDLERS__[key] = remaining;
    } else {
      delete window.__VERSTAK_IMPORT_PROGRESS_HANDLERS__[key];
    }
  };
}

function clearImportProgress(pluginId, sourceHandle) {
  delete window.__VERSTAK_IMPORT_PROGRESS_HANDLERS__[importProgressKey(pluginId, sourceHandle)];
}

window.__VERSTAK_DISPATCH_IMPORT_PROGRESS__ = dispatchImportProgress;

if (!window.__VERSTAK_IMPORT_PROGRESS_BRIDGE__ && window.runtime && typeof window.runtime.EventsOnMultiple === 'function') {
  window.__VERSTAK_IMPORT_PROGRESS_BRIDGE__ = window.runtime.EventsOnMultiple('verstak:import-progress', dispatchImportProgress, -1);
}

// Bulk file transfers report progress per item. A plugin only ever hears about
// its own transfers.
window.__VERSTAK_TRANSFER_PROGRESS_HANDLERS__ = window.__VERSTAK_TRANSFER_PROGRESS_HANDLERS__ || {};

function dispatchTransferProgress(progress) {
  if (!progress || !progress.pluginId) return;
  const handlers = (window.__VERSTAK_TRANSFER_PROGRESS_HANDLERS__[progress.pluginId] || []).slice();
  handlers.forEach(function(handler) {
    try {
      handler(progress);
    } catch (e) {
      console.error('[VerstakPluginAPI] transfer progress handler error:', e);
    }
  });
}

function trackTransferProgress(pluginId, listener) {
  const handlers = window.__VERSTAK_TRANSFER_PROGRESS_HANDLERS__[pluginId] || [];
  handlers.push(listener);
  window.__VERSTAK_TRANSFER_PROGRESS_HANDLERS__[pluginId] = handlers;
  let active = true;
  return function unsubscribeTransferProgress() {
    if (!active) return;
    active = false;
    const current = window.__VERSTAK_TRANSFER_PROGRESS_HANDLERS__[pluginId] || [];
    const remaining = current.filter(function(item) { return item !== listener; });
    if (remaining.length > 0) {
      window.__VERSTAK_TRANSFER_PROGRESS_HANDLERS__[pluginId] = remaining;
    } else {
      delete window.__VERSTAK_TRANSFER_PROGRESS_HANDLERS__[pluginId];
    }
  };
}

window.__VERSTAK_DISPATCH_TRANSFER_PROGRESS__ = dispatchTransferProgress;

if (!window.__VERSTAK_TRANSFER_PROGRESS_BRIDGE__ && window.runtime && typeof window.runtime.EventsOnMultiple === 'function') {
  window.__VERSTAK_TRANSFER_PROGRESS_BRIDGE__ = window.runtime.EventsOnMultiple('verstak:files-transfer-progress', dispatchTransferProgress, -1);
}

function normalizeTransfers(transfers) {
  if (!Array.isArray(transfers)) {
    throw new Error('a bulk transfer requires an array of { from, to } pairs');
  }
  return transfers.map(function(transfer, index) {
    const from = transfer && transfer.from;
    const to = transfer && transfer.to;
    if (!from || !to) {
      throw new Error('transfer ' + index + ' needs both a from and a to path');
    }
    return { from: String(from), to: String(to) };
  });
}

window.__VERSTAK_PLUGIN_STYLE_RECORDS__ = window.__VERSTAK_PLUGIN_STYLE_RECORDS__ || {};

export async function acquirePluginStyle(pluginId, stylePath) {
  if (!stylePath) return function() {};
  const records = window.__VERSTAK_PLUGIN_STYLE_RECORDS__;
  let record = records[pluginId];
  if (!record) {
    record = { refs: 0, element: null, promise: null };
    records[pluginId] = record;
    record.promise = callBackend(pluginId, 'frontend.style', function() {
      return App.GetPluginAssetContent(pluginId, stylePath);
    }).then(function(content) {
      if (!content) throw new Error('declared plugin stylesheet is empty');
      const element = document.createElement('style');
      element.setAttribute('data-verstak-plugin-style', pluginId);
      element.textContent = content;
      document.head.appendChild(element);
      record.element = element;
    });
  }
  record.refs += 1;
  try {
    await record.promise;
  } catch (error) {
    record.refs -= 1;
    if (record.refs === 0 && records[pluginId] === record) delete records[pluginId];
    throw error;
  }
  let released = false;
  return function releasePluginStyle() {
    if (released) return;
    released = true;
    record.refs -= 1;
    if (record.refs === 0) {
      record.element?.remove();
      if (records[pluginId] === record) delete records[pluginId];
    }
  };
}

window.__VERSTAK_PLUGIN_BUNDLE_LOADS__ = window.__VERSTAK_PLUGIN_BUNDLE_LOADS__ || {};
window.__VERSTAK_PLUGIN_ACTIVATIONS__ = window.__VERSTAK_PLUGIN_ACTIVATIONS__ || {};

function bundleFailure(code, message, info) {
  const error = new Error(message);
  error.code = code;
  if (info) error.info = info;
  return error;
}

async function executePluginBundle(pluginId, entry) {
  const [content, assetError] = unpack(await App.GetPluginAssetContent(pluginId, entry));
  if (assetError || !content) {
    throw bundleFailure('asset', assetError || 'plugin bundle is empty');
  }
  try {
    // Executed through the Function constructor rather than eval: the bundle
    // gets the global scope and nothing of ours.
    const fn = new Function(content);
    fn();
  } catch (e) {
    throw bundleFailure('execution', e && e.message ? e.message : String(e));
  }
  if (!window.__VERSTAK_PLUGIN_REGISTRY__[pluginId]) {
    throw bundleFailure('registration', 'plugin bundle registered no components');
  }
}

// Loads and runs a plugin's frontend bundle once, whatever asked for it first.
// Two views of the same plugin can open at the same moment, and a command can
// arrive while a view is still loading -- all of them wait on one execution.
export async function loadPluginBundle(pluginId) {
  if (!pluginId) {
    throw bundleFailure('invalid', 'loadPluginBundle requires pluginId');
  }
  const info = await App.GetPluginFrontendInfo(pluginId);
  if (!info || info.status === 'no-frontend' || info.status === 'not-found' || !info.entry) {
    const code = info && info.status === 'not-found' ? 'not-found' : 'no-frontend';
    throw bundleFailure(code, 'plugin has no frontend to load');
  }
  if (!window.__VERSTAK_PLUGIN_REGISTRY__[pluginId]) {
    const loads = window.__VERSTAK_PLUGIN_BUNDLE_LOADS__;
    if (!loads[pluginId]) {
      loads[pluginId] = executePluginBundle(pluginId, info.entry);
      loads[pluginId].catch(function() {
        // A failed load must not be remembered as done: reopening the view is
        // how a user retries.
        delete loads[pluginId];
      });
    }
    try {
      await loads[pluginId];
    } catch (error) {
      if (error && !error.info) error.info = info;
      throw error;
    }
  }
  return {
    info: info,
    components: window.__VERSTAK_PLUGIN_REGISTRY__[pluginId],
    bundle: window.__VERSTAK_PLUGIN_BUNDLES__[pluginId] || null
  };
}

// A plugin can be asked for something with none of its views on screen: the
// Journal asks Activity for possible entries while the user is looking at the
// Journal. Handlers registered from mount() exist only while that view is
// mounted, so a bundle may declare activate(api), which runs once and stays for
// as long as the app does.
export function activatePlugin(pluginId) {
  const activations = window.__VERSTAK_PLUGIN_ACTIVATIONS__;
  if (activations[pluginId]) return activations[pluginId];
  const activation = loadPluginBundle(pluginId).then(function(loaded) {
    const bundle = loaded && loaded.bundle;
    if (!bundle || typeof bundle.activate !== 'function') return null;
    const api = createPluginAPI(pluginId);
    return Promise.resolve(bundle.activate(api)).then(function() {
      return api;
    });
  });
  activations[pluginId] = activation;
  activation.catch(function() {
    if (activations[pluginId] === activation) delete activations[pluginId];
  });
  return activation;
}

// Drops every background activation. Used when the plugin set itself changes.
export function deactivatePlugins() {
  const activations = window.__VERSTAK_PLUGIN_ACTIVATIONS__;
  Object.keys(activations).forEach(function(pluginId) {
    const activation = activations[pluginId];
    delete activations[pluginId];
    Promise.resolve(activation).then(function(api) {
      if (api && typeof api.dispose === 'function') api.dispose();
    }).catch(function() {});
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
  const key = commandKey(pluginId, cmdId);
  let handler = window.__VERSTAK_COMMAND_HANDLERS__[key];
  if (!handler) {
    // The command is declared, so the plugin providing it may simply not be on
    // screen. Give it its chance to register before calling it unhandled.
    try {
      await activatePlugin(pluginId);
    } catch (e) {
      console.warn('[VerstakPluginAPI] activation for ' + pluginId + ' failed:', e);
    }
    handler = window.__VERSTAK_COMMAND_HANDLERS__[key];
  }
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
  const importSourceHandles = new Set();
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

  async function selectImportSource(kind) {
    assertActive('imports.select' + (kind === 'archive' ? 'Archive' : 'Directory'));
    const source = await callBackend(pluginId, 'imports.select' + (kind === 'archive' ? 'Archive' : 'Directory'), function() {
      return kind === 'archive'
        ? App.PluginSelectImportArchive(pluginId)
        : App.PluginSelectImportDirectory(pluginId);
    });
    if (!source || !source.sourceHandle) return null;
    if (disposed) {
      Promise.resolve(App.PluginCloseImportSource(pluginId, source.sourceHandle)).catch(function() {});
      throw new Error('[plugin:' + pluginId + '] imports.select failed: API disposed');
    }
    importSourceHandles.add(source.sourceHandle);
    return source;
  }

  function closeImportSource(sourceHandle) {
    assertActive('imports.closeSource');
    return callBackendErrorString(pluginId, 'imports.closeSource', function() {
      return App.PluginCloseImportSource(pluginId, sourceHandle);
    }).finally(function() {
      importSourceHandles.delete(sourceHandle);
      clearImportProgress(pluginId, sourceHandle);
    });
  }

  return {
    pluginId: pluginId,

    // Shortcut matching is a host service on purpose. Left to plugins it gets
    // reimplemented with event.key, which silently stops working on any
    // non-Latin keyboard layout.
    keys: {
      matches: function(event, spec) {
        assertActive('keys.matches');
        return matchesShortcut(event, spec);
      },
    },

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

    navigation: {
      // Back and forward reach whatever is on screen first. A plugin that keeps
      // its own history — a file browser walking folders — says so here instead
      // of the shell hunting for its buttons in the DOM.
      registerHandler: function(handler) {
        assertActive('navigation.registerHandler');
        return trackCleanup(registerNavigationHandler(pluginId, handler));
      },
      // Workspace navigation is a host concern. Plugins supply stable target
      // identity and optional opaque state instead of dispatching shell-private
      // DOM events themselves. WorkspaceHost already transports toolRequest to
      // the mounted contribution, so this remains provider/component agnostic.
      openWorkspace: async function(request) {
        assertActive('navigation.openWorkspace');
        request = request || {};
        const workspaceId = String(request.workspaceId || '').trim();
        const workspaceItemId = String(request.workspaceItemId || '').trim();
        if (!dealWorkspaceIdPattern.test(workspaceId)) {
          throw new Error('navigation.openWorkspace requires workspaceId UUID');
        }
        await callBackendErrorString(pluginId, 'navigation.openWorkspace', function() {
          return App.SetCurrentWorkspaceV2(workspaceId);
        });
        const workspace = await callBackend(pluginId, 'navigation.openWorkspace', function() {
          return App.GetWorkspaceByID(workspaceId);
        });
        const workspaceRootPath = String(workspace?.rootPath || '').trim();
        if (!workspace || workspace.id !== workspaceId || !workspaceRootPath) {
          throw new Error('[plugin:' + pluginId + '] navigation.openWorkspace failed: workspaceId was not resolved by host');
        }
        window.dispatchEvent(new CustomEvent('verstak:workspace-selected', {
          detail: {
            workspaceId: workspaceId,
            workspaceName: workspaceRootPath,
            workspaceRootPath: workspaceRootPath,
            workspaceItemId: workspaceItemId,
            toolRequest: request.toolRequest || null
          }
        }));
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
      },
      invoke: async function(capId, operation, args) {
        assertActive('capabilities.invoke(' + capId + ':' + operation + ')');
        const request = args || {};
        const resolved = await callBackend(pluginId, 'capabilities.invoke(' + capId + ':' + operation + ')', function() {
          if (dealScopedProviderCapabilities.has(capId)) {
            return App.ResolveDealCapabilityOperation(pluginId, capId, operation, request);
          }
          return App.ResolvePluginCapabilityOperation(pluginId, capId, operation);
        });
        if (!resolved || !resolved.pluginId || !resolved.commandId) {
          throw new Error('[plugin:' + pluginId + '] capabilities.invoke(' + capId + ':' + operation + ') failed: invalid capability resolution');
        }
        return executePluginCommand(resolved.pluginId, resolved.commandId, request);
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

    folders: {
      _readAll: async function() {
        const data = await callBackend(pluginId, 'storage.data.read(appearance)', function() {
          return App.ReadPluginDataJSON(pluginId, 'appearance');
        });
        return (data && data.folders) || {};
      },
      getAppearance: async function(folderId) {
        try {
          const all = await this._readAll();
          return all[folderId] || { iconId: '', colorId: '' };
        } catch {
          return { iconId: '', colorId: '' };
        }
      },
      setAppearance: function(folderId, appearance) {
        assertActive('folders.setAppearance(' + folderId + ')');
        return this._readAll().then(function(all) {
          if (appearance && (appearance.iconId || appearance.colorId)) {
            all[folderId] = { iconId: appearance.iconId || '', colorId: appearance.colorId || '' };
          } else {
            delete all[folderId];
          }
          return callBackendErrorString(pluginId, 'storage.data.write(appearance)', function() {
            return App.WritePluginDataJSON(pluginId, 'appearance', { folders: all });
          });
        });
      }
    },

    workspaces: {
      list: function() {
        assertActive('workspaces.list');
        return callBackend(pluginId, 'workspaces.list', function() {
          return App.PluginListWorkspaces(pluginId);
        });
      },
      // Read-only user-visible Deal/folder hierarchy. Unlike list(), this is
      // intentionally not filtered by whether this plugin contributes a tool
      // to a Deal; stable Deal UUIDs are platform identity, not plugin state.
      tree: function() {
        assertActive('workspaces.tree');
        return callBackend(pluginId, 'workspaces.tree', function() {
          return App.GetWorkspaceTreeV2();
        }).then(function(snapshot) {
          if (!snapshot || !Array.isArray(snapshot.roots)) {
            return { roots: [], currentWorkspaceId: '', revision: 0, warnings: [] };
          }
          return {
            roots: snapshot.roots,
            currentWorkspaceId: snapshot.currentWorkspaceId || '',
            revision: Number(snapshot.revision || 0),
            warnings: Array.isArray(snapshot.warnings) ? snapshot.warnings : []
          };
        });
      },
      resolvePath: function(relativePath) {
        assertActive('workspaces.resolvePath');
        return callBackend(pluginId, 'workspaces.resolvePath', function() {
          return App.PluginResolveWorkspacePath(pluginId, String(relativePath || ''));
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
      copy: function(fromRelativePath, toRelativePath, options) {
        assertActive('files.copy(' + fromRelativePath + ')');
        return callBackendErrorString(pluginId, 'files.copy(' + fromRelativePath + ')', function() {
          return App.CopyVaultPath(pluginId, fromRelativePath, toRelativePath, options || {});
        });
      },
      // One call for many paths. A loop over files.move costs the host one sync
      // recording per file, which is what made large pastes appear to hang.
      moveMany: function(transfers, options) {
        assertActive('files.moveMany');
        const settings = options || {};
        return callBackend(pluginId, 'files.moveMany', function() {
          return App.MoveVaultPaths(pluginId, settings.transferId || '', normalizeTransfers(transfers), settings);
        });
      },
      copyMany: function(transfers, options) {
        assertActive('files.copyMany');
        const settings = options || {};
        return callBackend(pluginId, 'files.copyMany', function() {
          return App.CopyVaultPaths(pluginId, settings.transferId || '', normalizeTransfers(transfers), settings);
        });
      },
      cancelTransfer: function(transferId) {
        assertActive('files.cancelTransfer');
        return callBackendErrorString(pluginId, 'files.cancelTransfer', function() {
          return App.CancelVaultTransfer(pluginId, String(transferId == null ? '' : transferId));
        });
      },
      onTransferProgress: function(listener) {
        assertActive('files.onTransferProgress');
        if (typeof listener !== 'function') {
          throw new Error('files.onTransferProgress requires a listener function');
        }
        return trackCleanup(trackTransferProgress(pluginId, listener));
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

    imports: {
      selectDirectory: function() {
        return selectImportSource('directory');
      },
      selectArchive: function() {
        return selectImportSource('archive');
      },
      listEntries: function(sourceHandle, cursor) {
        assertActive('imports.listEntries');
        return callBackend(pluginId, 'imports.listEntries', function() {
          return App.PluginListImportEntries(pluginId, sourceHandle, cursor || '');
        });
      },
      readText: function(sourceHandle, entryId) {
        assertActive('imports.readText');
        return callBackend(pluginId, 'imports.readText', function() {
          return App.PluginReadImportText(pluginId, sourceHandle, entryId);
        });
      },
      onProgress: function(sourceHandle, listener) {
        assertActive('imports.onProgress');
        if (!importSourceHandles.has(sourceHandle)) {
          throw new Error('[plugin:' + pluginId + '] imports.onProgress failed: unknown source handle');
        }
        if (typeof listener !== 'function') {
          throw new Error('imports.onProgress requires a listener function');
        }
        return trackCleanup(trackImportProgress(pluginId, sourceHandle, listener));
      },
      applyPlan: function(sourceHandle, plan) {
        assertActive('imports.applyPlan');
        return callBackend(pluginId, 'imports.applyPlan', function() {
          return App.PluginApplyImportPlan(pluginId, sourceHandle, plan);
        });
      },
      cancel: function(sourceHandle) {
        assertActive('imports.cancel');
        return callBackendErrorString(pluginId, 'imports.cancel', function() {
          return App.PluginCancelImport(pluginId, sourceHandle);
        });
      },
      closeSource: closeImportSource
    },

    sync: {
      status: function() {
        assertActive('sync.status');
        return callBackend(pluginId, 'sync.status', function() {
          return App.PluginSyncStatus(pluginId);
        });
      },
      configure: function(serverURL, username, password, vaultId) {
        assertActive('sync.configure');
        return callBackendErrorString(pluginId, 'sync.configure', function() {
          return App.PluginSyncConfigure(pluginId, serverURL || '', username || '', password || '', vaultId || '');
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
        const raw = await callBackend(pluginId, 'contributions.list', function() {
          return App.GetContributions();
        });
        // A plugin showing another plugin's contribution shows its label to the
        // user, so it has to arrive in the user's language -- the shell does
        // this for its own menus and the same has to hold here.
        const summary = typeof i18n.localizeContributionSummary === 'function'
          ? i18n.localizeContributionSummary(raw)
          : raw;
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
      importSourceHandles.forEach(function(sourceHandle) {
        clearImportProgress(pluginId, sourceHandle);
        Promise.resolve(App.PluginCloseImportSource(pluginId, sourceHandle)).then(function(error) {
          if (error) console.error('[VerstakPluginAPI] source cleanup error:', error);
        }).catch(function(error) {
          console.error('[VerstakPluginAPI] source cleanup error:', error);
        });
      });
      importSourceHandles.clear();
    }
  };
}

window.createPluginAPI = createPluginAPI;
window.VerstakPluginAPI = createPluginAPI;
