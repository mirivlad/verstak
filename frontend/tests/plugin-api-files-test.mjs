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
        RestoreVaultTrash: (pluginId, trashId, options) => {
          calls.push({ pluginId, trashId, options });
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

const restored = await api.files.restoreTrash('trash-1', { overwrite: true });
if (restored !== 'Docs/restored.txt') {
  throw new Error(`unexpected restore result: ${JSON.stringify(restored)}`);
}
if (calls.length !== 1 || calls[0].pluginId !== 'verstak.files' || calls[0].trashId !== 'trash-1' || calls[0].options.overwrite !== true) {
  throw new Error(`unexpected RestoreVaultTrash call: ${JSON.stringify(calls)}`);
}

console.log('plugin api files smoke passed');
