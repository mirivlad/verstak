import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';

const calls = [];
let emitImportProgress = () => {};
const styleElements = [];

globalThis.document = {
  head: {
    appendChild: (element) => styleElements.push(element),
  },
  createElement: () => ({
    attributes: {},
    textContent: '',
    setAttribute(name, value) { this.attributes[name] = value; },
    remove() {
      const index = styleElements.indexOf(this);
      if (index !== -1) styleElements.splice(index, 1);
    },
  }),
};

globalThis.window = {
  __VERSTAK_PLUGIN_REGISTRY__: {},
  __VERSTAK_EVENT_HANDLERS__: {},
  __VERSTAK_COMMAND_HANDLERS__: {},
  runtime: {
    EventsOnMultiple: (eventName, listener) => {
      if (eventName === 'verstak:import-progress') emitImportProgress = listener;
      return () => {};
    },
  },
  go: {
    api: {
      App: {
        PluginSelectImportArchive: (pluginId) => {
          calls.push({ method: 'selectArchive', pluginId });
          return Promise.resolve([{ sourceHandle: 'archive-1', fingerprint: 'fp-1' }, '']);
        },
        PluginSelectImportDirectory: (pluginId) => {
          calls.push({ method: 'selectDirectory', pluginId });
          return Promise.resolve([{ sourceHandle: 'directory-1', fingerprint: 'fp-2' }, '']);
        },
        PluginCancelImport: (pluginId, sourceHandle) => {
          calls.push({ method: 'cancel', pluginId, sourceHandle });
          return Promise.resolve('');
        },
        PluginCloseImportSource: (pluginId, sourceHandle) => {
          calls.push({ method: 'close', pluginId, sourceHandle });
          return Promise.resolve('');
        },
        GetPluginAssetContent: (pluginId, stylePath) => {
          calls.push({ method: 'style', pluginId, stylePath });
          return Promise.resolve(['.importer { color: green; }', '']);
        },
      },
    },
  },
};
globalThis.__mockApp = window.go.api.App;
globalThis.__mockI18n = {
  getLocale: () => 'ru',
  translatePlugin: (_pluginId, key, _params, fallback) => fallback || key,
  subscribe: () => () => {},
};

const sourcePath = path.resolve('frontend/src/lib/plugin-host/VerstakPluginAPI.js');
const source = fs.readFileSync(sourcePath, 'utf8')
  .replace("import * as App from '../../../wailsjs/go/api/App';", 'const App = globalThis.__mockApp;')
  .replace("import { i18n } from '../i18n/index.js';", 'const i18n = globalThis.__mockI18n;');
const tempPath = path.resolve('/tmp/verstak-plugin-api-imports-test.mjs');
fs.writeFileSync(tempPath, source);

const apiModule = await import(pathToFileURL(tempPath).href + '?t=' + Date.now());
const releaseStyleOne = await apiModule.acquirePluginStyle('verstak.import', 'frontend/dist/style.css');
const releaseStyleTwo = await apiModule.acquirePluginStyle('verstak.import', 'frontend/dist/style.css');
if (styleElements.length !== 1 || calls.filter((call) => call.method === 'style').length !== 1) {
  throw new Error(`stylesheet was not shared: elements=${styleElements.length} calls=${JSON.stringify(calls)}`);
}
releaseStyleOne();
if (styleElements.length !== 1) throw new Error('stylesheet released while still referenced');
releaseStyleTwo();
if (styleElements.length !== 0) throw new Error('stylesheet was not removed');

const api = apiModule.createPluginAPI('verstak.import');
if (!api.imports) throw new Error('api.imports is missing');

const sourceSession = await api.imports.selectArchive();
const progress = [];
const unsubscribe = api.imports.onProgress(sourceSession.sourceHandle, (item) => progress.push(item.phase));
emitImportProgress({ pluginId: 'other.plugin', sourceHandle: sourceSession.sourceHandle, phase: 'staging' });
emitImportProgress({ pluginId: 'verstak.import', sourceHandle: 'other-handle', phase: 'staging' });
emitImportProgress({ pluginId: 'verstak.import', sourceHandle: sourceSession.sourceHandle, phase: 'staging' });
if (progress.join(',') !== 'staging') throw new Error(`unexpected progress: ${progress}`);
unsubscribe();
emitImportProgress({ pluginId: 'verstak.import', sourceHandle: sourceSession.sourceHandle, phase: 'publishing' });
if (progress.join(',') !== 'staging') throw new Error(`unsubscribe failed: ${progress}`);

await api.imports.cancel(sourceSession.sourceHandle);
await api.imports.closeSource(sourceSession.sourceHandle);
await api.imports.selectDirectory();
api.dispose();
await new Promise((resolve) => setTimeout(resolve, 0));

const closes = calls.filter((call) => call.method === 'close');
if (closes.length !== 2 || closes[0].sourceHandle !== 'archive-1' || closes[1].sourceHandle !== 'directory-1') {
  throw new Error(`unexpected closes: ${JSON.stringify(closes)}`);
}
const cancel = calls.find((call) => call.method === 'cancel');
if (!cancel || cancel.pluginId !== 'verstak.import' || cancel.sourceHandle !== 'archive-1') {
  throw new Error(`unexpected cancel: ${JSON.stringify(cancel)}`);
}

console.log('plugin api imports smoke passed');
