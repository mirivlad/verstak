// Exercise draft preservation with physical input in Wails/WebKitGTK.
// Only the inspector reads geometry/state; files use the real Wails bridge.
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
const deadline = setTimeout(() => { console.error('Native draft smoke timed out'); process.exit(1); }, 180000);
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
async function screenshot(name) { input('mousemove', '--window', windowId, '1150', '700'); await delay(300); execFileSync('import', ['-window', windowId, path.join(output, name + '.png')]); }


function data(result) {
  assert.equal(result.error, undefined);
  if (Array.isArray(result.value) && result.value.length === 2) {
    assert.ok(!result.value[1], result.value[1]);
    return result.value[0];
  }
  return result.value;
}

try {
  await waitFor('!!document.querySelector(".sidebar")', 'application');
  for (const folder of ['Drafts', 'Drafts/Notes']) {
    const error = data(await call(`window.go.api.App.CreateVaultFolder('verstak.default-editor',${JSON.stringify(folder)})`));
    assert.ok(!error, error);
  }
  const results = [];
  for (const markdown of [false, true]) {
    for (const settled of [true, false]) {
      const notePath = `Drafts/Notes/${settled ? 'Control' : 'Rapid'}.${markdown ? 'md' : 'txt'}`;
      const content = 'native rapid draft';
      const writeError = data(await call(`window.go.api.App.WriteVaultTextFile('verstak.default-editor',${JSON.stringify(notePath)},'base',{createIfMissing:true,overwrite:true})`));
      assert.ok(!writeError, writeError);
      const open = async () => {
        const request = { kind: 'vault-file', path: notePath, extension: markdown ? '.md' : '.txt', mode: 'edit', context: markdown ? { notesMode: true, notesScopePath: 'Drafts' } : {} };
        const opened = data(await call(`window.go.api.App.OpenWorkbenchResource('verstak.default-editor',${JSON.stringify(request)})`));
        assert.equal(opened.status, 'opened');
        await evaluate(`window.dispatchEvent(new CustomEvent('verstak:workbench-opened',{detail:${JSON.stringify(opened)}}))`);
        await waitFor('!!document.querySelector(".cm-content")', 'editor');
      };
      await open();
      await click('.cm-content');
      input('key', 'ctrl+a');
      const close = await evaluate(`(() => { const r = document.querySelector('.main-content-header .close-btn').getBoundingClientRect(); return {x:r.x+r.width/2,y:r.y+r.height/2}; })()`);
      input('type', '--clearmodifiers', '--delay', '0', content);
      await waitFor(`document.querySelector('.cm-content').textContent === ${JSON.stringify(content)}`, 'full text entered');
      if (settled) await delay(400);
      input('mousemove', '--window', windowId, String(Math.round(close.x)), String(Math.round(close.y)), 'click', '1');
      await waitFor('!document.querySelector(".cm-content")', 'closed editor');
      await open();
      const reopened = await waitFor(`(() => { const text = [...document.querySelectorAll('.cm-content .cm-line')].map(n => n.textContent).join('\\n'); return text === ${JSON.stringify(content)} && text; })()`, 'restored latest draft');
      assert.equal(fs.readFileSync(path.join(vault, notePath), 'utf8'), 'base', 'the file is not implicitly saved');
      const draftFiles = fs.readdirSync(vault, { recursive: true }).filter(file => file.includes('draft-') && file.endsWith('.json'));
      const draft = draftFiles.map(file => JSON.parse(fs.readFileSync(path.join(vault, file), 'utf8'))).find(record => record.path === notePath);
      assert.equal(draft?.content, content, 'the draft is persisted through the real backend');
      results.push({ markdown, settled, reopened, persistedDraft: true, diskUnchanged: true });
      await click('.main-content-header .close-btn');
      await waitFor('!document.querySelector(".cm-content")', 'closed editor');
    }
  }
  fs.writeFileSync(path.join(output, 'result.json'), JSON.stringify(results, null, 2) + '\n');
  console.log(JSON.stringify(results));
} catch (error) {
  console.error(error);
  await screenshot('failure');
  process.exitCode = 1;
} finally {
  clearTimeout(deadline);
  socket.close();
}
