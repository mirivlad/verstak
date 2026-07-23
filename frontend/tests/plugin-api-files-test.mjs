import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';

const calls = [];

globalThis.window = {
  __VERSTAK_PLUGIN_REGISTRY__: {},
  __VERSTAK_EVENT_HANDLERS__: {},
  __VERSTAK_COMMAND_HANDLERS__: {},
  go: {
    api: {
      App: {
        ReadVaultFileBytes: (pluginId, relativePath) => {
          calls.push({ method: 'ReadVaultFileBytes', pluginId, relativePath });
          return Promise.resolve([{ relativePath, size: 4, mimeHint: 'image/png', dataBase64: 'iVBORw==' }, '']);
        },
        RestoreVaultTrash: (pluginId, trashId, options) => {
          calls.push({ method: 'RestoreVaultTrash', pluginId, trashId, options });
          return Promise.resolve(['Docs/restored.txt', '']);
        },
        WriteVaultFileBytes: (pluginId, relativePath, dataBase64, options) => {
          calls.push({ method: 'WriteVaultFileBytes', pluginId, relativePath, dataBase64, options });
          return Promise.resolve('');
        },
        CopyVaultPath: (pluginId, fromRelativePath, toRelativePath, options) => {
          calls.push({ method: 'CopyVaultPath', pluginId, fromRelativePath, toRelativePath, options });
          return Promise.resolve('');
        },
        PluginListWorkspaces: (pluginId) => {
          calls.push({ method: 'PluginListWorkspaces', pluginId });
          return Promise.resolve([[{ id: 'ws-nested', name: 'Acme', rootPath: 'Clients/Acme' }], '']);
        },
      },
    },
  },
};
globalThis.__mockApp = window.go.api.App;
globalThis.__mockI18n = {
  getLocale: () => 'en',
  translatePlugin: (_pluginId, key, _params, fallback) => fallback || key,
  subscribe: () => () => {},
};

const sourcePath = path.resolve('frontend/src/lib/plugin-host/VerstakPluginAPI.js');
const source = fs.readFileSync(sourcePath, 'utf8')
  .replace("import * as App from '../../../wailsjs/go/api/App';", 'const App = globalThis.__mockApp;')
  .replace("import { i18n } from '../i18n/index.js';", 'const i18n = globalThis.__mockI18n;');
const tempPath = path.resolve('/tmp/verstak-plugin-api-files-test.mjs');
fs.writeFileSync(tempPath, source);

const apiModule = await import(pathToFileURL(tempPath).href + '?t=' + Date.now());
const api = apiModule.createPluginAPI('verstak.files');

if (!api.files || typeof api.files.restoreTrash !== 'function') {
  throw new Error('api.files.restoreTrash is missing');
}
if (typeof api.files.readBytes !== 'function') {
  throw new Error('api.files.readBytes is missing');
}
if (typeof api.files.writeBytes !== 'function') {
  throw new Error('api.files.writeBytes is missing');
}
if (typeof api.files.copy !== 'function') {
  throw new Error('api.files.copy is missing');
}
if (!api.workspaces || typeof api.workspaces.list !== 'function') {
  throw new Error('api.workspaces.list is missing');
}

const bytes = await api.files.readBytes('Docs/image.png');
if (bytes.dataBase64 !== 'iVBORw==' || bytes.mimeHint !== 'image/png') {
  throw new Error(`unexpected readBytes result: ${JSON.stringify(bytes)}`);
}

await api.files.writeBytes('Docs/copy.png', 'iVBORw==', { createIfMissing: true });
await api.files.copy('Project/Files/report.pdf', 'Client/Files/report.pdf', { overwrite: false });

const restored = await api.files.restoreTrash('trash-1', { overwrite: true });
const workspaces = await api.workspaces.list();
if (restored !== 'Docs/restored.txt') {
  throw new Error(`unexpected restore result: ${JSON.stringify(restored)}`);
}
if (workspaces.length !== 1 || workspaces[0].rootPath !== 'Clients/Acme') {
  throw new Error(`unexpected workspace result: ${JSON.stringify(workspaces)}`);
}
if (calls.length !== 5 || calls[0].method !== 'ReadVaultFileBytes' || calls[0].pluginId !== 'verstak.files' || calls[0].relativePath !== 'Docs/image.png') {
  throw new Error(`unexpected ReadVaultFileBytes call: ${JSON.stringify(calls)}`);
}
if (calls[1].method !== 'WriteVaultFileBytes' || calls[1].pluginId !== 'verstak.files' || calls[1].relativePath !== 'Docs/copy.png' || calls[1].dataBase64 !== 'iVBORw==' || calls[1].options.createIfMissing !== true) {
  throw new Error(`unexpected WriteVaultFileBytes call: ${JSON.stringify(calls)}`);
}
if (calls[2].method !== 'CopyVaultPath' || calls[2].fromRelativePath !== 'Project/Files/report.pdf' || calls[2].toRelativePath !== 'Client/Files/report.pdf' || calls[2].options.overwrite !== false) {
  throw new Error(`unexpected CopyVaultPath call: ${JSON.stringify(calls)}`);
}
if (calls[3].method !== 'RestoreVaultTrash' || calls[3].pluginId !== 'verstak.files' || calls[3].trashId !== 'trash-1' || calls[3].options.overwrite !== true) {
  throw new Error(`unexpected RestoreVaultTrash call: ${JSON.stringify(calls)}`);
}
if (calls[4].method !== 'PluginListWorkspaces' || calls[4].pluginId !== 'verstak.files') {
  throw new Error(`unexpected PluginListWorkspaces call: ${JSON.stringify(calls)}`);
}

console.log('plugin api files smoke passed');
