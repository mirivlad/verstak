import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';

const calls = [];
const workspaceId = '11111111-1111-4111-8111-111111111111';

globalThis.document = {
  head: { appendChild() {} },
  createElement: () => ({ setAttribute() {}, remove() {}, textContent: '' }),
};
globalThis.window = {
  __VERSTAK_PLUGIN_REGISTRY__: {},
  __VERSTAK_EVENT_HANDLERS__: {},
  __VERSTAK_COMMAND_HANDLERS__: {},
  runtime: { EventsOnMultiple: () => () => {} },
  go: { api: { App: {
    PluginGitClone(pluginId, request) {
      calls.push({ method: 'clone', pluginId, request });
      return Promise.resolve([{ checkoutPath: 'Project/Repositories/repo-one' }, '']);
    },
    PluginGitRegisterExisting(pluginId, request) {
      calls.push({ method: 'registerExisting', pluginId, request });
      return Promise.resolve([{ checkoutPath: 'Project/Repositories/repo-one' }, '']);
    },
    PluginGitStatus(pluginId, request) {
      calls.push({ method: 'status', pluginId, request });
      return Promise.resolve([{ state: 'cloned', branch: 'main', clean: true, changedCount: 0, untrackedCount: 0, changedFiles: [], ahead: 0, behind: 0, recentCommits: [] }, '']);
    },
    PluginGitFetch(pluginId, request) { calls.push({ method: 'fetch', pluginId, request }); return Promise.resolve(''); },
    PluginGitPull(pluginId, request) { calls.push({ method: 'pull', pluginId, request }); return Promise.resolve(''); },
    PluginGitPush(pluginId, request) { calls.push({ method: 'push', pluginId, request }); return Promise.resolve(''); },
    PluginGitOpenDirectory(pluginId, request) { calls.push({ method: 'openDirectory', pluginId, request }); return Promise.resolve(''); },
  } } },
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
const tempPath = path.join(path.dirname(sourcePath), '.verstak-plugin-api-git-test.mjs');
fs.writeFileSync(tempPath, source);
process.on('exit', () => { try { fs.unlinkSync(tempPath); } catch {} });

const apiModule = await import(pathToFileURL(tempPath).href + '?t=' + Date.now());
const api = apiModule.createPluginAPI('verstak.git');
if (!api.git) throw new Error('api.git is missing');

const request = {
  scope: { kind: 'deal', workspaceId },
  repositoryId: 'repo-1', checkoutName: 'repo-one', remoteUrl: 'https://example.com/repo.git',
  branch: 'main', credentialRef: 'verstak-secret://git-token',
};
await api.git.clone(request);
await api.git.status(request);
await api.git.registerExisting({ ...request, sourcePath: '' });
await api.git.fetch(request);
await api.git.pull(request);
await api.git.push(request);
await api.git.openDirectory(request);

if (calls.length !== 7) throw new Error(`unexpected Git call count: ${calls.length}`);
for (const call of calls) {
  if (call.pluginId !== 'verstak.git') throw new Error(`wrong plugin id: ${JSON.stringify(call)}`);
  if (call.request.workspaceId !== workspaceId || 'scope' in call.request) {
    throw new Error(`DealScope was not flattened safely: ${JSON.stringify(call)}`);
  }
  if (call.request.repositoryId !== 'repo-1' || call.request.checkoutName !== 'repo-one') {
    throw new Error(`repository identity was not preserved: ${JSON.stringify(call)}`);
  }
}
if (calls.find((call) => call.method === 'registerExisting').request.sourcePath !== '') {
  throw new Error('registerExisting should allow Core to choose the external source');
}

let rejected = false;
try {
  await api.git.status({ scope: { kind: 'deal', workspaceId: 'Project' }, repositoryId: 'repo-1', checkoutName: 'repo-one' });
} catch (error) {
  rejected = String(error && error.message || error).includes('DealScope.workspaceId');
}
if (!rejected) throw new Error('invalid Deal scope reached the backend');

api.dispose();
console.log('plugin api Git bridge smoke passed');
