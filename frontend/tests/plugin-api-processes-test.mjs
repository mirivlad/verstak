import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import {pathToFileURL} from 'node:url';
globalThis.window={};
const calls=[];
const request={kind:'process',executable:'sshkeeper',args:['connect','two words; $(literal)'],mode:'detached'};
globalThis.__mockApp={};
for(const name of ['Approved','Review','Revoke','Run','Cancel']) globalThis.__mockApp['PluginProcess'+name]=async(...args)=>{calls.push([name,...args]);return name==='Approved'||name==='Review'?false:name==='Run'?{started:true}:undefined;};
globalThis.__mockI18n={getLocale:()=> 'en',subscribe:()=>()=>{}};
const sourcePath=path.resolve('frontend/src/lib/plugin-host/VerstakPluginAPI.js');
const source=fs.readFileSync(sourcePath,'utf8').replace("import * as App from '../../../wailsjs/go/api/App';",'const App=globalThis.__mockApp;').replace("import { i18n } from '../i18n/index.js';",'const i18n=globalThis.__mockI18n;');
const tempPath=path.join(path.dirname(sourcePath),'.actions-api-test.mjs');fs.writeFileSync(tempPath,source);
try{
 const {createPluginAPI}=await import(pathToFileURL(tempPath).href);const api=createPluginAPI('test.actions');
 assert.equal(await api.processes.approved(request),false);assert.equal(await api.processes.review(request),false);
 await api.processes.run('one',request);await api.processes.cancel('one');await api.processes.revoke(request);
 assert.deepEqual(calls,[['Approved','test.actions',request],['Review','test.actions',request],['Run','test.actions','one',request],['Cancel','test.actions','one'],['Revoke','test.actions',request]]);
 globalThis.__mockApp.PluginProcessRun=async()=>{throw new Error('approval_required');};
 await assert.rejects(api.processes.run('two',request),/approval_required/);
 api.dispose();assert.throws(()=>api.processes.run('three',request),/disposed/);
 console.log('process bridge: plugin identity, literal argv, errors and disposal passed');
}finally{fs.unlinkSync(tempPath);}
