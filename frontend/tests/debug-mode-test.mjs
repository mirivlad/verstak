import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import vm from 'node:vm';

const source = fs.readFileSync(path.resolve('frontend/src/lib/log/debug.js'), 'utf8');
const values = new Map();
const localStorage = {
  getItem(key) { return values.has(key) ? values.get(key) : null; },
  setItem(key, value) { values.set(key, String(value)); },
  removeItem(key) { values.delete(key); },
};
const context = vm.createContext({
  console: { log() {} },
  localStorage,
  window: { location: { search: '' } },
});
const module = new vm.SourceTextModule(source, { context, identifier: 'debug.js' });
await module.link(() => { throw new Error('debug.js must not import modules'); });
await module.evaluate();

const { debug } = module.namespace;
debug.enable({ persist: false });

assert.equal(debug.isEnabled(), true, 'session debug should enable diagnostics');
assert.equal(
  localStorage.getItem('verstak-debug'),
  null,
  'a --debug session must not make diagnostics persist into ordinary launches',
);

console.log('debug mode smoke passed');
