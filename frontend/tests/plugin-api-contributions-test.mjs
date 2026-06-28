import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';

const pluginData = {};

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
      },
    },
  },
};
globalThis.__mockApp = window.go.api.App;

const sourcePath = path.resolve('frontend/src/lib/plugin-host/VerstakPluginAPI.js');
const source = fs.readFileSync(sourcePath, 'utf8')
  .replace("import * as App from '../../../wailsjs/go/api/App';", 'const App = globalThis.__mockApp;');
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

console.log('plugin api contributions smoke passed');
