// Drives the real WebKitGTK page with physical mouse/keyboard events. The
// inspector reads DOM geometry/state; Browser Inbox operations cross the Wails bridge.
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
const deadline = setTimeout(() => { console.error('Native Browser Inbox smoke timed out'); process.exit(1); }, 180000);
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
  if (Array.isArray(result.value) && result.value.length === 2 && (typeof result.value[1] === 'string' || result.value[1] == null)) { assert.ok(!result.value[1]); return result.value[0]; }
  return result.value;
}
async function setValue(selector, value) {
  await evaluate(`document.querySelector(${JSON.stringify(selector)}).value=${JSON.stringify(value)}`);
}
async function sidebar(label) {
  const point = await waitFor(`(() => { const n=[...document.querySelectorAll('.sidebar .plugin-item')].find(n=>n.textContent.trim()===${JSON.stringify(label)});if(!n)return null;const r=n.getBoundingClientRect();return {x:r.x+r.width/2,y:r.y+r.height/2};})()`,label);
  input('mousemove','--window',windowId,String(Math.round(point.x)),String(Math.round(point.y)),'click','1');
}

const inboxId = 'verstak.browser-inbox';
const dialog = '[data-browser-inbox-add-dialog]';
const field = name => dialog + ' [data-browser-inbox-add="' + name + '"]';
const add = '[data-browser-inbox-action="add"]';
async function clickText(selector, label) {
  const point = await waitFor(`(() => { const n=[...document.querySelectorAll(${JSON.stringify(selector)})].find(n=>n.textContent.trim()===${JSON.stringify(label)});if(!n)return null;const r=n.getBoundingClientRect();return {x:r.x+r.width/2,y:r.y+r.height/2};})()`, label);
  input('mousemove','--window',windowId,String(Math.round(point.x)),String(Math.round(point.y)),'click','1');
}
async function type(selector, value) {
  await click(selector);
  input('type','--clearmodifiers','--delay','8',value);
  await waitFor(`document.querySelector(${JSON.stringify(selector)}).value===${JSON.stringify(value)}`, 'typed field');
}
async function records() {
  return data(await call(`window.go.api.App.ReadPluginSettings('${inboxId}')`))['captures:global'] || [];
}
async function save() {
  await click('[data-browser-inbox-add-submit]');
  await waitFor(`!document.querySelector(${JSON.stringify(dialog)})`, 'material saved');
}
async function assertDialog() {
  const outside = await evaluate(`(() => { const panel=document.querySelector('${dialog}'),r=panel.getBoundingClientRect();return [...panel.querySelectorAll('input,textarea,select,button')].filter(n=>{const b=n.getBoundingClientRect();return b.top<r.top-1||b.bottom>r.bottom+1||b.left<r.left-1||b.right>r.right+1;}).length;})()`);
  assert.equal(outside,0,'controls inside native dialog');
  const color = await evaluate(`getComputedStyle(document.querySelector('[data-browser-inbox-add-submit]')).color`);
  assert.equal(color,'rgb(255, 255, 255)','white action text');
}
try {
  await waitFor('document.querySelector(".sidebar")','application');
  const parent = data(await call("window.go.api.App.CreateFolderV2('', 'Clients')"));
  data(await call(`window.go.api.App.PluginCreateWorkspace('verstak.templates',${JSON.stringify(parent.id)},'Site',{workspaceTools:['verstak.browser-inbox','verstak.notes'],initialFolders:['Notes'],initialFiles:[],provenance:{templateId:'native-browser-inbox'}})`));
  const workspaces = data(await call(`window.go.api.App.PluginListWorkspaces('${inboxId}')`));
  const deal = workspaces.find(w=>w.rootPath==='Clients/Site');
  assert.ok(deal?.id,'nested Deal UUID');
  await call(`window.go.api.App.OpenVault(${JSON.stringify(vault)})`);
  await evaluate("window.dispatchEvent(new CustomEvent('verstak:workspace-tree-changed'))");
  await sidebar('Браузер');
  await click(add);
  await waitFor(`document.querySelector(${JSON.stringify(field('workspace'))})?.options.length===2`,'only actual Deals offered');
  assert.equal(await evaluate(`document.activeElement.matches('[data-browser-inbox-add="url"]')`),true,'initial URL focus');
  await type(field('url'),'file:///tmp/example');
  await click('[data-browser-inbox-add-submit]');
  await waitFor(`document.querySelector('${dialog} [role="alert"]')`,'URL validation');
  await setValue(field('url'),'https://example.test/guide');
  await type(field('title'),'Saved reference');
  await type(field('text'),'A useful quotation.');
  await setValue(field('workspace'),deal.id);
  await assertDialog();
  await screenshot('manual-form-light');
  await save();
  let captures=await records();
  assert.equal(captures.length,1);
  assert.equal(captures[0].source,'manual');
  assert.equal(captures[0].workspaceId,deal.id);
  assert.equal(captures[0].workspaceRootPath,'Clients/Site');
  assert.equal(captures[0].text,'A useful quotation.');
  await screenshot('manual-detail-light');
  await click('[data-browser-inbox-action="create-note"]');
  await waitFor(`document.querySelector('.browser-inbox-status')?.textContent.includes('архив')`,'capture converted and archived');
  const notePath=path.join(vault,'Clients/Site/Notes/Saved_reference.md');
  assert.ok(fs.readFileSync(notePath,'utf8').includes('A useful quotation.'));
  await click('[data-theme-toggle]');
  await waitFor('document.documentElement.dataset.theme==="dark"','dark theme');
  await click(add);
  await type(field('text'),'Text without a URL.');
  await assertDialog();
  await screenshot('manual-form-dark');
  await save();
  captures=await records();
  assert.equal(captures.length,2);
  const unassigned=captures.find(c=>c.title==='Text without a URL.');
  assert.equal(unassigned.url,'');
  assert.equal(unassigned.workspaceId,'');
  assert.equal(unassigned.workspaceState,'unassigned');
  await sidebar('Журнал');
  await sidebar('Браузер');
  assert.equal((await records()).length,2,'persisted across navigation');
  await screenshot('manual-detail-dark');
  if (!await evaluate("[...document.querySelectorAll('.wt-label')].some(n=>n.textContent.trim()==='Site')")) {
    await clickText('.wt-label','Clients');
  }
  await clickText('.wt-label','Site');
  await clickText('[role="tab"]','Браузер');
  await click(add);
  assert.equal(await evaluate(`document.querySelector('${dialog} select')===null`),true,'current Deal is fixed');
  await type(field('title'),'From the Deal');
  await type(field('text'),'Scoped material.');
  await save();
  captures=await records();
  const scoped=captures.find(c=>c.title==='From the Deal');
  assert.equal(scoped.workspaceId,deal.id);
  assert.equal(scoped.workspaceRootPath,'Clients/Site');
  const settingsPath=path.join(vault,'.verstak/plugin-settings',inboxId,'settings.json');
  const onDisk=JSON.parse(fs.readFileSync(settingsPath,'utf8'))['captures:global'];
  assert.equal(onDisk.length,3);
  const result={materialsOnDisk:3,nestedDealUUID:true,textWithoutURL:true,explicitUnassigned:true,scopedCapture:true,noteConversion:true,persistsAcrossNavigation:true,httpUrlValidation:true,dialogBounds:true,whiteActionText:true,lightAndDark:true};
  fs.writeFileSync(path.join(output,'result.json'),JSON.stringify(result,null,2)+'\n');
  console.log(JSON.stringify(result));
} catch (error) {
  console.error('Fixture tree on failure:',JSON.stringify(data(await call('window.go.api.App.GetWorkspaceTreeV2()'))));
  console.error('Native UI state:',await evaluate(`({status:[...document.querySelectorAll('.browser-inbox-status,.browser-inbox-manual-status')].map(n=>n.textContent),tabs:[...document.querySelectorAll('[role="tab"]')].map(n=>n.textContent)})`));
  await screenshot('failure');
  throw error;
} finally {
  clearTimeout(deadline);
  socket.close();
}
