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
      },
    },
  },
};
globalThis.__mockApp = window.go.api.App;

const sourcePath = path.resolve('frontend/src/lib/plugin-host/VerstakPluginAPI.js');
const source = fs.readFileSync(sourcePath, 'utf8')
  .replace("import * as App from '../../../wailsjs/go/api/App';", 'const App = globalThis.__mockApp;');
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

const bytes = await api.files.readBytes('Docs/image.png');
if (bytes.dataBase64 !== 'iVBORw==' || bytes.mimeHint !== 'image/png') {
  throw new Error(`unexpected readBytes result: ${JSON.stringify(bytes)}`);
}

const restored = await api.files.restoreTrash('trash-1', { overwrite: true });
if (restored !== 'Docs/restored.txt') {
  throw new Error(`unexpected restore result: ${JSON.stringify(restored)}`);
}
if (calls.length !== 2 || calls[0].method !== 'ReadVaultFileBytes' || calls[0].pluginId !== 'verstak.files' || calls[0].relativePath !== 'Docs/image.png') {
  throw new Error(`unexpected ReadVaultFileBytes call: ${JSON.stringify(calls)}`);
}
if (calls[1].method !== 'RestoreVaultTrash' || calls[1].pluginId !== 'verstak.files' || calls[1].trashId !== 'trash-1' || calls[1].options.overwrite !== true) {
  throw new Error(`unexpected RestoreVaultTrash call: ${JSON.stringify(calls)}`);
}

console.log('plugin api files smoke passed');
