// Drives the real WebKitGTK page with physical mouse/keyboard events. The
// inspector reads DOM geometry/state; creation still crosses the Wails bridge.
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { execFileSync } from 'node:child_process';

const [vault, output, windowId, inspectorPort] = process.argv.slice(2);
const socket = new WebSocket(`ws://127.0.0.1:${inspectorPort}/socket/1/1/WebPage`);
const pending = new Map();
let sequence = 10;
let targetId;
let ready;
const connected = new Promise((resolve) => { ready = resolve; });
socket.onmessage = ({ data }) => {
  const message = JSON.parse(data);
  if (message.method === 'Target.targetCreated') {
    targetId = message.params.targetInfo.targetId;
    ready();
  }
  if (message.method === 'Target.dispatchMessageFromTarget') {
    const response = JSON.parse(message.params.message);
    const request = pending.get(response.id);
    if (!request) return;
    pending.delete(response.id);
    if (response.error || response.result?.wasThrown) request.reject(new Error(JSON.stringify(response)));
    else request.resolve(response.result?.result?.value);
  }
};
const deadline = setTimeout(() => { console.error('Native creation smoke timed out'); process.exit(1); }, 90000);
const delay = (ms) => new Promise(resolve => setTimeout(resolve, ms));
async function evaluate(expression) {
  await connected;
  const id = sequence++;
  return new Promise((resolve, reject) => {
    pending.set(id, { resolve, reject });
    socket.send(JSON.stringify({ id: sequence++, method: 'Target.sendMessageToTarget', params: {
      targetId, message: JSON.stringify({ id, method: 'Runtime.evaluate', params: { expression, returnByValue: true } }),
    } }));
  });
}
async function waitFor(expression, label) {
  for (let attempt = 0; attempt < 100; attempt++) {
    const value = await evaluate(expression);
    if (value) return value;
    await delay(100);
  }
  throw new Error(`Timed out: ${label || expression}`);
}
async function call(expression) {
  await evaluate(`window.__nativeResult = null; Promise.resolve().then(() => (${expression})).then(value => window.__nativeResult = { done: true, value }, error => window.__nativeResult = { done: true, error: String(error?.message || error) })`);
  return waitFor('window.__nativeResult?.done && window.__nativeResult', expression);
}
function input(...args) { execFileSync('xdotool', args, { stdio: 'pipe' }); }
async function click(selector, button = '1') {
  const rect = await waitFor(`(() => { const node = document.querySelector(${JSON.stringify(selector)}); if (!node || !node.getClientRects().length || node.disabled) return null; const box = node.getBoundingClientRect(); return {x:box.x + box.width/2, y:box.y + box.height/2}; })()`, selector);
  input('mousemove', '--window', windowId, String(Math.round(rect.x)), String(Math.round(rect.y)), 'click', button);
}
function screenshot(name) { execFileSync('import', ['-window', windowId, path.join(output, name + '.png')]); }
async function openRoot() {
  await click('[data-workspace-tree] button[aria-label="Новое Дело"]');
  await waitFor('document.querySelector("[data-new-deal-template=seed-general]")', 'template catalog');
  assert.equal(await evaluate('document.activeElement === document.querySelector("[data-new-deal-name]")'), true, 'name receives initial focus');
}
async function create(name, relativePath = name) {
  await click('[data-new-deal-name]');
  input('type', '--clearmodifiers', '--delay', '25', name);
  await waitFor(`document.querySelector('[data-new-deal-name]')?.value === ${JSON.stringify(name)}`, 'name typed');
  await click('[data-new-deal-create]');
  await waitFor(`!document.querySelector('[data-new-deal-dialog]') && document.querySelector('[data-main-content-header]')?.textContent.includes(${JSON.stringify(name)})`, 'created Deal opens');
  const marker = JSON.parse(fs.readFileSync(path.join(vault, relativePath, '.verstak/workspace.json')));
  assert.match(marker.workspaceId, /^[0-9a-f-]{36}$/);
  const metadata = JSON.parse(fs.readFileSync(path.join(vault, '.verstak/workspaces', 'uuid-' + marker.workspaceId + '.json')));
  assert.equal(metadata.workspaceName, name);
  assert.ok(!metadata.workspaceTools.includes('verstak.activity'));
  return metadata;
}

try {
  await waitFor('document.querySelector("[data-workspace-tree]")', 'native application startup');
  await evaluate(`localStorage.setItem('verstak-debug','true'); window.__nativeCreationCalls = []; const original = window.go.api.App.PluginCreateWorkspace; window.go.api.App.PluginCreateWorkspace = async (...args) => { const call = {parent:args[1],name:args[2]}; window.__nativeCreationCalls.push(call); try { return await original(...args); } catch(error) { call.error = String(error); throw error; } };`);
  const root = await (async () => { await openRoot(); return create('Native Root'); })();
  assert.equal(root.createdFromTemplate.templateVersion, 2, 'legacy recipe was migrated');
  screenshot('root-created');

  await openRoot();
  for (let index = 0; index < 14; index++) {
    input('key', 'Tab');
    assert.equal(await evaluate('document.querySelector("[role=dialog]").contains(document.activeElement)'), true, 'Tab stays inside dialog');
  }
  await click('[data-new-deal-name]');
  input('type', '--clearmodifiers', '--delay', '25', 'Native Root');
  await waitFor('document.querySelector("[data-new-deal-name]")?.value === "Native Root"', 'duplicate name typed');
  await click('[data-new-deal-create]');
  await waitFor('document.querySelector("[data-new-deal-error]")?.textContent.includes("уже есть")', 'duplicate name error');
  assert.equal(await evaluate('document.querySelector("[data-new-deal-name]").value'), 'Native Root');
  screenshot('duplicate-error');
  input('key', 'Escape');
  await waitFor('!document.querySelector("[data-new-deal-dialog]")', 'cancel duplicate');
  assert.equal(await evaluate('document.activeElement?.getAttribute("aria-label")'), 'Новое Дело', 'cancel restores opener focus');

  const folderId = '11111111-1111-4111-8111-111111111111';
  await click(`[data-tree-key="folder:${folderId}"]`, '3');
  await click('[role=menu] .vt-menu-item');
  await waitFor('document.querySelector("[data-new-deal-template=seed-general]")', 'folder template catalog');
  assert.equal(await evaluate('document.querySelector("[data-new-deal-dialog] select").value'), folderId);
  const nested = await create('Native Nested', 'Clients/Native Nested');
  await waitFor(`document.querySelector('[data-tree-key="workspace:${nested.workspaceId}"]')?.getClientRects().length`, 'created Deal is visible in expanded folder');

  await openRoot();
  await click('[data-new-deal-template=custom-unavailable]');
  await click('[data-new-deal-name]');
  input('type', '--clearmodifiers', '--delay', '25', 'Should Not Exist');
  await waitFor('document.querySelector("[data-new-deal-name]")?.value === "Should Not Exist"', 'missing tool name typed');
  await click('[data-new-deal-create]');
  await waitFor('document.querySelector("[data-new-deal-error]")?.textContent.includes("unknown.plugin")', 'missing tool error');
  assert.equal(fs.existsSync(path.join(vault, 'Should Not Exist')), false);
  screenshot('missing-tool-error');
  input('key', 'Escape');
  await waitFor('!document.querySelector("[data-new-deal-dialog]")', 'cancel missing tool');

  const invalidParent = await call(`window.go.api.App.PluginCreateWorkspace('verstak.templates','1','Invalid Parent',{workspaceTools:[],initialFolders:[],initialFiles:[],provenance:{templateId:'native-test'}})`);
  assert.match(invalidParent.error, /parent folder not found/);
  assert.equal(fs.existsSync(path.join(vault, 'Invalid Parent')), false);
  const stored = fs.readFileSync(path.join(vault, '.verstak/plugin-data/verstak.templates/templates.ndjson'), 'utf8').trim().split('\n').map(JSON.parse);
  assert.ok(stored.every(row => !row.workspaceTools.includes('verstak.activity')));
  assert.ok(stored.find(row => row.id === 'custom-unavailable').workspaceTools.includes('unknown.plugin'));
  assert.equal(stored.find(row => row.id === 'custom-unavailable').toolConfig.custom.preserve, true);

  const cleared = await call(`window.go.api.App.WritePluginDataNDJSON('verstak.templates','templates',[])`);
  assert.ok(!cleared.error);
  await openRoot();
  const fresh = await create('Native Fresh');
  assert.equal(fresh.createdFromTemplate.templateVersion, 1, 'fresh vault seeds need no migration');
  screenshot('fresh-created');
  const calls = await evaluate('window.__nativeCreationCalls');
  assert.equal(calls[0].parent, '');
  assert.ok(calls.some(call => call.name === 'Native Nested' && call.parent === folderId));
  fs.writeFileSync(path.join(output, 'results.json'), JSON.stringify({ root, fresh, stored, invalidParent, calls }, null, 2) + '\n');
  console.log('Real desktop GUI checked: root click, folder menu, legacy/fresh recipes, focus, duplicate/missing tool, Wails rejection');
} finally {
  clearTimeout(deadline);
  socket.close();
}
