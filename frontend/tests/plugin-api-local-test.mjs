import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import {pathToFileURL} from 'node:url';
globalThis.window={};const calls=[];
globalThis.__localApp={
 ReadPluginLocalDataJSON:async(...args)=>{calls.push(['read',...args]);return {one:'hash'};},
 WritePluginLocalDataJSON:async(...args)=>{calls.push(['write',...args]);return '';},
 PluginProcessConfirm:async(...args)=>{calls.push(['confirm',...args]);return false;}
};
globalThis.__localI18n={getLocale:()=> 'en',subscribe:()=>()=>{}};
const sourcePath=path.resolve('frontend/src/lib/plugin-host/VerstakPluginAPI.js');
const source=fs.readFileSync(sourcePath,'utf8').replace("import * as App from '../../../wailsjs/go/api/App';",'const App=globalThis.__localApp;').replace("import { i18n } from '../i18n/index.js';",'const i18n=globalThis.__localI18n;');
const temp=path.join(path.dirname(sourcePath),'.local-api-test.mjs');fs.writeFileSync(temp,source);
try{
 const {createPluginAPI}=await import(pathToFileURL(temp).href);const api=createPluginAPI('local.plugin');
 assert.deepEqual(await api.storage.local.read('trust'),{one:'hash'});
 await api.storage.local.write('trust',{two:'hash2'});
 const request={kind:'url',url:'https://example.org'};assert.equal(await api.processes.confirm(request),false);
 assert.deepEqual(calls,[['read','local.plugin','trust'],['write','local.plugin','trust',{two:'hash2'}],['confirm','local.plugin',request]]);
 api.dispose();assert.throws(()=>api.storage.local.write('trust',{}),/disposed/);assert.throws(()=>api.processes.confirm(request),/disposed/);
 console.log('device-local data / optional confirmation bridge: namespace, payload and disposal passed');
}finally{fs.unlinkSync(temp);}
