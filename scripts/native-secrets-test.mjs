// Drives the real WebKitGTK page with physical mouse/keyboard events. The
// inspector reads DOM geometry/state; Secrets operations cross the Wails bridge.
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
const deadline = setTimeout(() => { console.error('Native creation smoke timed out'); process.exit(1); }, 180000);
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
async function master(password, creating = false) {
  await click('[data-secret-master-password]');
  input('type','--clearmodifiers','--delay','8',password);
  await waitFor(`document.querySelector('[data-secret-master-password]').value === ${JSON.stringify(password)}`, 'password typed accurately');
  if (creating) await setValue('[data-secret-master-password-confirm]',password);
  input('key','Return');
}
const oldPassword='Native-old-master-42';
const newPassword='Native-new-master-84';
const deal='Secrets Native';
const secretId='native.admin%25';
const notePath=deal+'/Notes/Admin.md';
const originalNote='Сайт. Админка: [логин/пароль](verstak-secret://'+encodeURIComponent(secretId)+')';
try {
  await waitFor('document.querySelector(".sidebar")','application');
  await sidebar('Секреты');
  await waitFor('document.querySelector("[data-secret-master-password-confirm]")','initial setup');
  await master(oldPassword,true);
  await waitFor('document.querySelector("[data-secret-search]")','old store unlocked');
  data(await call(`window.go.api.App.PluginSecretsWrite('verstak.secrets',{id:'old.test',title:'Archived fixture',value:'old-test-value',scope:{kind:'global'}})`));
  const oldRoot=path.join(vault,'.verstak/secrets');
  const oldFiles=new Map(['metadata.json',...fs.readdirSync(path.join(oldRoot,'records')).map(name=>'records/'+name)].map(name=>[name,fs.readFileSync(path.join(oldRoot,name))]));
  await call(`window.go.api.App.OpenVault(${JSON.stringify(vault)})`);
  await sidebar('Журнал');
  await sidebar('Секреты');
  await click('[data-secret-forgot-password]');
  assert.equal(await evaluate('document.querySelector("[data-secret-archive-confirm]").disabled'),true);
  await screenshot('archive-confirmation');
  await click('[data-secret-archive-acknowledge]');
  await click('[data-secret-archive-confirm]');
  await waitFor('document.querySelector("[data-secret-master-password-confirm]")','new store setup');
  const archives=path.join(vault,'.verstak/secrets-archives');
  const archive=path.join(archives,fs.readdirSync(archives)[0],'secrets');
  for (const [name,bytes] of oldFiles) assert.ok(bytes.equals(fs.readFileSync(path.join(archive,name))),'archive bytes '+name);
  await master(newPassword,true);
  await waitFor('document.querySelector("[data-secret-search]")','new store unlocked');
  assert.equal(data(await call(`window.go.api.App.PluginSecretsList('verstak.secrets')`)).length,0);
  data(await call(`window.go.api.App.PluginCreateWorkspace('verstak.templates','',${JSON.stringify(deal)},{workspaceTools:['verstak.notes','verstak.secrets'],initialFolders:['Notes'],initialFiles:[],provenance:{templateId:'native-secrets'}})`));
  data(await call(`window.go.api.App.PluginSecretsWrite('verstak.secrets',{id:${JSON.stringify(secretId)},title:'Админка сайта',username:'deploy-user',value:'native-test-value',scope:{kind:'workspace',workspaceRootPath:${JSON.stringify(deal)}}})`));
  await call(`window.go.api.App.WriteVaultTextFile('verstak.default-editor',${JSON.stringify(notePath)},${JSON.stringify(originalNote)},{createIfMissing:true,overwrite:true})`);
  await call(`window.go.api.App.OpenVault(${JSON.stringify(vault)})`);
  await evaluate(`window.__secretsUnlockCount=0; const unlock=window.go.api.App.PluginSecretsUnlock; window.go.api.App.PluginSecretsUnlock=(...args)=>{window.__secretsUnlockCount++;return unlock(...args);}`);
  const opened=data(await call(`window.go.api.App.OpenWorkbenchResource('verstak.default-editor',{kind:'vault-file',path:${JSON.stringify(notePath)},extension:'.md',context:{notesMode:true,notesScopePath:${JSON.stringify(deal)}}})`));
  await evaluate(`window.dispatchEvent(new CustomEvent('verstak:workbench-opened',{detail:${JSON.stringify(opened)}}))`);
  await click('[data-preview] .secret-link');
  await waitFor('document.querySelector("[data-resource-dialog] [data-secret-master-password]")','master prompt above note');
  assert.equal(await evaluate('document.querySelector("[data-editor-mode]").getAttribute("data-resource-path")'),notePath);
  await master(newPassword);
  await waitFor('document.querySelector("[data-secret-dialog-value]")?.value==="native-test-value"','decrypted dialog');
  assert.equal(await evaluate('document.querySelector("[data-secret-dialog-value]").readOnly'),true);
  await click('[data-secret-copy-username]');
  assert.equal(data(await call('window.runtime.ClipboardGetText()')),'deploy-user','native clipboard username');
  await click('[data-secret-copy-value]');
  assert.equal(data(await call('window.runtime.ClipboardGetText()')),'native-test-value','native clipboard value');
  await screenshot('secret-light');
  await click('[data-secret-edit]');
  await setValue('[data-secret-username]','updated-user');
  await setValue('[data-secret-value]','updated-native-value');
  await click('[data-secret-save]');
  await waitFor('document.querySelector("[data-secret-dialog-value]")?.value==="updated-native-value"','saved in same dialog');
  const changed=data(await call(`window.go.api.App.PluginSecretsRead('verstak.secrets',${JSON.stringify(secretId)})`));
  assert.equal(changed.scope.workspaceRootPath,deal,'edit retains Deal scope');
  assert.equal(changed.username,'updated-user');
  await click('[data-resource-dialog-close]');
  await waitFor('!document.querySelector("[data-resource-dialog]")','dialog close');
  assert.equal(await evaluate('document.activeElement?.matches(".secret-link")'),true,'focus returns to link');
  await click('[data-theme-toggle]');
  await waitFor('document.documentElement.dataset.theme==="dark"','dark theme');
  await click('[data-preview] .secret-link');
  await waitFor('document.querySelector("[data-secret-dialog-value]")?.value==="updated-native-value"','second opening without master prompt');
  assert.equal(await evaluate('document.querySelector("[data-secret-master-password]")===null'),true);
  assert.equal(await evaluate('window.__secretsUnlockCount'),1,'master password once per session');
  await screenshot('secret-dark');
  await click('[data-resource-dialog-close]');
  await waitFor('!document.querySelector("[data-resource-dialog]")','close before editing note');
  assert.equal(data(await call(`window.go.api.App.ReadVaultTextFile('verstak.default-editor',${JSON.stringify(notePath)})`)),originalNote);
  await click('[data-editor-action="mode-edit"]');
  await click('.cm-content');
  input('key','ctrl+a');
  input('type','--clearmodifiers','--delay','8','Site. Admin: login/password');
  input('key','ctrl+End');
  for (let i=0;i<'login/password'.length;i++) input('key','shift+Left');
  await click('[data-editor-action="insert-secret-link"]');
  await click('[data-secret-picker-id="native.admin%25"]');
  await click('[data-editor-action="save"]');
  const expected='Site. Admin: [login/password](verstak-secret://'+encodeURIComponent(secretId)+')';
  for(let i=0;i<100;i++) {
    const text=data(await call(`window.go.api.App.ReadVaultTextFile('verstak.default-editor',${JSON.stringify(notePath)})`));
    if(text===expected)break;
    if(i===99)assert.equal(text,expected);
    await delay(100);
  }
  assert.ok(!fs.readFileSync(path.join(vault,notePath),'utf8').includes('updated-native-value'),'note contains no password');
  await call(`window.go.api.App.OpenVault(${JSON.stringify(vault)})`);
  assert.equal(data(await call(`window.go.api.App.PluginSecretsStatus('verstak.secrets')`)).unlocked,false,'vault reopen clears session');
  const shipping=data(await call('window.go.api.App.GetPlugins()')).filter(plugin=>plugin.manifest.id!=='verstak.platform-test');
  assert.equal(shipping.length,17,'shipping plugins present');
  assert.ok(shipping.every(plugin=>plugin.status==='loaded'),'shipping plugins loaded');
  const logs=fs.readFileSync(path.join(output,'app.log'),'utf8');
  const config=fs.readFileSync(path.join(path.dirname(vault),'.config/verstak/config.json'),'utf8');
  for (const sensitive of [oldPassword,newPassword,'native-test-value','updated-native-value']) {
    assert.ok(!logs.includes(sensitive),'no credentials in application log');
    assert.ok(!config.includes(sensitive),'no credentials in app settings');
  }
  const result={archivePreservedByteForByte:true,newStoreEmpty:true,noteStayedOpen:true,masterPromptCount:1,nativeClipboard:true,inPlaceEdit:true,dealScopePreserved:true,encodedStableId:true,selectedLabelPreserved:true,noPasswordInMarkdown:true,lightAndDark:true,vaultReopenLocks:true,noCredentialsInLogsOrSettings:true,shippingPlugins:shipping.length};
  fs.writeFileSync(path.join(output,'result.json'),JSON.stringify(result,null,2)+'\n');
  console.log(JSON.stringify(result));
} catch (error) {
  console.error('Native UI state:', await evaluate(`({status:[...document.querySelectorAll('.secrets-status')].map(n=>n.textContent),passwordLength:document.querySelector('[data-secret-master-password]')?.value.length,confirmationLength:document.querySelector('[data-secret-master-password-confirm]')?.value.length})`));
  await screenshot('failure');
  throw error;
} finally {
  clearTimeout(deadline);
  socket.close();
}
