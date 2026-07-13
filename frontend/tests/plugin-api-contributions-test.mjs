import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';

const pluginData = {};
const scheduledNotifications = [];

globalThis.window = {
  __VERSTAK_PLUGIN_REGISTRY__: {},
  __VERSTAK_EVENT_HANDLERS__: {},
  __VERSTAK_COMMAND_HANDLERS__: {},
  go: {
    api: {
      App: {
        GetContributions: () => Promise.resolve({
          fileActions: [
            {
              pluginId: 'provider.plugin',
              id: 'provider.file.action',
              label: 'Provider File Action',
              handler: 'provider.command',
            },
          ],
          noteActions: [],
          contextMenuEntries: [],
        }),
        ExecutePluginCommand: (pluginId, commandId, args) => Promise.resolve([{
          status: 'declared',
          pluginId,
          commandId,
          args,
        }, '']),
        ReadPluginDataJSON: (pluginId, name) => Promise.resolve([
          Object.assign({}, (pluginData[pluginId] && pluginData[pluginId][name]) || {}),
          '',
        ]),
        WritePluginDataJSON: (pluginId, name, data) => {
          pluginData[pluginId] = pluginData[pluginId] || {};
          pluginData[pluginId][name] = Object.assign({}, data || {});
          return Promise.resolve('');
        },
        ReplacePluginNotifications: (pluginId, items) => {
          scheduledNotifications.push({ pluginId, items });
          return Promise.resolve(items[0]?.id === 'rejected' ? 'notification permission denied' : '');
        },
        ClearPluginNotifications: (pluginId) => {
          scheduledNotifications.push({ pluginId, clear: true });
          return Promise.resolve('');
        },
      },
    },
  },
};
globalThis.__mockApp = window.go.api.App;
const localeListeners = new Set();
globalThis.__mockI18n = {
  getLocale: () => 'ru',
  translatePlugin: (_pluginId, key, params, fallback) => {
    const messages = { greeting: 'Привет, {name}!' };
    const message = messages[key] || fallback || key;
    return message.replace(/\{([^}]+)\}/g, (placeholder, name) => (
      Object.prototype.hasOwnProperty.call(params || {}, name) ? String(params[name]) : placeholder
    ));
  },
  subscribe: (listener) => {
    localeListeners.add(listener);
    listener('ru');
    return () => localeListeners.delete(listener);
  },
};

const sourcePath = path.resolve('frontend/src/lib/plugin-host/VerstakPluginAPI.js');
const source = fs.readFileSync(sourcePath, 'utf8')
  .replace("import * as App from '../../../wailsjs/go/api/App';", 'const App = globalThis.__mockApp;')
  .replace("import { i18n } from '../i18n/index.js';", 'const i18n = globalThis.__mockI18n;');
const tempPath = path.resolve('/tmp/verstak-plugin-api-contributions-test.mjs');
fs.writeFileSync(tempPath, source);

const apiModule = await import(pathToFileURL(tempPath).href + '?t=' + Date.now());
const api = apiModule.createPluginAPI('verstak.files');

if (!api.contributions || typeof api.contributions.list !== 'function') {
  throw new Error('api.contributions.list is missing');
}
if (!api.commands || typeof api.commands.executeFor !== 'function') {
  throw new Error('api.commands.executeFor is missing');
}
if (!api.i18n || typeof api.i18n.getLocale !== 'function' || typeof api.i18n.t !== 'function' || typeof api.i18n.onDidChangeLocale !== 'function') {
  throw new Error('api.i18n contract is missing');
}
if (api.i18n.getLocale() !== 'ru' || api.i18n.t('greeting', { name: 'Мир' }) !== 'Привет, Мир!') {
  throw new Error('api.i18n locale or translation is incorrect');
}
let localeNotifications = 0;
api.i18n.onDidChangeLocale(() => { localeNotifications += 1; });
if (localeNotifications !== 1 || localeListeners.size !== 1) {
  throw new Error('api.i18n locale subscription was not registered');
}

const fileActions = await api.contributions.list('fileActions');
if (fileActions.length !== 1 || fileActions[0].id !== 'provider.file.action') {
  throw new Error(`unexpected file actions: ${JSON.stringify(fileActions)}`);
}

window.__VERSTAK_COMMAND_HANDLERS__['provider.plugin:provider.command'] = (args, declared) => ({
  handledPath: args.path,
  declaredPlugin: declared.pluginId,
});

const result = await api.commands.executeFor('provider.plugin', 'provider.command', {
  source: 'files',
  path: 'Project/Docs/readme.md',
});
if (result.status !== 'handled' || result.result.handledPath !== 'Project/Docs/readme.md') {
  throw new Error(`unexpected executeFor result: ${JSON.stringify(result)}`);
}

if (!api.storage || !api.storage.data || typeof api.storage.data.read !== 'function' || typeof api.storage.data.write !== 'function') {
  throw new Error('api.storage.data read/write is missing');
}

await api.storage.data.write('search-index', { version: 1, workspaceRootPath: 'Project' });
const stored = await api.storage.data.read('search-index');
if (stored.version !== 1 || stored.workspaceRootPath !== 'Project') {
  throw new Error(`unexpected storage data: ${JSON.stringify(stored)}`);
}

if (!api.notifications || typeof api.notifications.replace !== 'function' || typeof api.notifications.clear !== 'function') {
  throw new Error('api.notifications replace/clear contract is missing');
}
await api.notifications.replace([{ id: 'reminder-1', dueAt: '2026-07-14T10:00:00Z', title: 'Reminder' }]);
await api.notifications.clear();
if (scheduledNotifications.length !== 2
  || scheduledNotifications[0].pluginId !== 'verstak.files'
  || scheduledNotifications[0].items[0].id !== 'reminder-1'
  || scheduledNotifications[1].pluginId !== 'verstak.files'
  || !scheduledNotifications[1].clear) {
  throw new Error(`unexpected notification calls: ${JSON.stringify(scheduledNotifications)}`);
}
let rejected = false;
try {
  await api.notifications.replace([{ id: 'rejected', dueAt: '2026-07-14T10:00:00Z', title: 'Reminder' }]);
} catch (err) {
  rejected = String(err.message || err).includes('[plugin:verstak.files] notifications.replace failed: notification permission denied');
}
if (!rejected) {
  throw new Error('notification backend errors must be plugin-scoped rejections');
}

api.dispose();
if (localeListeners.size !== 0) {
  throw new Error('api.i18n locale subscription was not disposed');
}

console.log('plugin api contributions smoke passed');
