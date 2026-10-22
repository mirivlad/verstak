import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';

const calls = [];
let error = '';
globalThis.window = {
  __VERSTAK_PLUGIN_REGISTRY__: {}, __VERSTAK_EVENT_HANDLERS__: {}, __VERSTAK_COMMAND_HANDLERS__: {},
};
globalThis.__mockApp = {
  UpsertPluginSettingRecord: async (...args) => { calls.push(args); return error; },
};
globalThis.__mockI18n = { getLocale: () => 'en', translatePlugin: (_id, key, _params, fallback) => fallback || key, subscribe: () => () => {} };
const sourcePath = path.resolve('frontend/src/lib/plugin-host/VerstakPluginAPI.js');
const source = fs.readFileSync(sourcePath, 'utf8')
  .replace("import * as App from '../../../wailsjs/go/api/App';", 'const App = globalThis.__mockApp;')
  .replace("import { i18n } from '../i18n/index.js';", 'const i18n = globalThis.__mockI18n;');
const tempPath = path.join(path.dirname(sourcePath), '.verstak-plugin-api-settings-records-test.mjs');
fs.writeFileSync(tempPath, source);
try {
  const { createPluginAPI } = await import(pathToFileURL(tempPath).href);
  const api = createPluginAPI('records.plugin');
  const record = { id: 'one', text: 'Saved input' };
  await api.settings.upsertRecord('entries', 'id', record, 100);
  assert.deepEqual(calls, [['records.plugin', 'entries', 'id', record, 100]]);
  error = 'permission denied';
  await assert.rejects(api.settings.upsertRecord('entries', 'id', record, 100), /permission denied/);
  api.dispose();
  assert.throws(() => api.settings.upsertRecord('entries', 'id', record, 100), /API disposed/);
  assert.equal(calls.length, 2, 'disposed API never reaches the backend');
  console.log('settings record bridge: namespace, arguments, failure and disposal passed');
} finally { fs.unlinkSync(tempPath); }
