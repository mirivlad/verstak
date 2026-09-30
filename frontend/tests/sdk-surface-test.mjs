// The plugin API the desktop hands a plugin and the API the SDK declares must
// be the same surface. keys, notifications, secrets and files.openURL shipped
// in the host and were used by official plugins for weeks while the SDK typed
// none of them, so a third-party author could not know they existed.
//
// The SDK is read from the sibling verstak-sdk checkout, which CI provides; the
// test refuses to pass without it rather than quietly checking nothing.
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';

const repo = path.resolve('.');
const hostPath = path.join(repo, 'frontend/src/lib/plugin-host/VerstakPluginAPI.js');
const sdkPath = path.resolve(repo, '..', 'verstak-sdk', 'src', 'plugin-api.ts');

if (!fs.existsSync(sdkPath)) {
  console.error(`SDK surface test cannot judge: ${sdkPath} is missing; check out verstak-sdk beside verstak-desktop`);
  process.exit(1);
}

// Host-only members with a reason. Anything here is deliberately undocumented.
const HOST_ONLY = {
  folders: 'legacy folder appearance shim; appearance is core-owned since v0.1.3 and no plugin uses it',
};

function hostSurface(source) {
  const start = source.indexOf('export function createPluginAPI');
  const body = source.slice(source.indexOf('  return {', start));
  const surface = {};
  let current = null;
  for (const line of body.split('\n')) {
    if (line.startsWith('  };')) break;
    let match = line.match(/^ {4}([A-Za-z0-9]+): \{$/);
    if (match) {
      current = match[1];
      surface[current] = new Set();
      continue;
    }
    match = line.match(/^ {4}([A-Za-z0-9]+)[:,(]/);
    if (match) {
      current = null;
      surface[match[1]] = null;
      continue;
    }
    if (current) {
      match = line.match(/^ {6}(?:async )?([A-Za-z0-9]+)(?::| ?\()/);
      if (match && !match[1].startsWith('_')) surface[current].add(match[1]);
    }
  }
  return surface;
}

function sdkSurface(source) {
  const start = source.indexOf('export interface VerstakPluginAPI {');
  assert.ok(start >= 0, 'SDK must declare VerstakPluginAPI');
  const surface = {};
  let depth = 0;
  let parens = 0;
  let current = null;
  let i = source.indexOf('{', start);
  const text = source.slice(i);
  const lines = text.split('\n');
  for (const raw of lines) {
    const line = raw.replace(/\/\*.*?\*\/|\/\/.*$/g, '');
    const trimmed = line.trim();
    if (parens > 0) {
      // inside a multi-line parameter list: nothing here is a member
    } else if (depth === 1 && !trimmed.startsWith('*')) {
      const match = trimmed.match(/^(?:readonly\s+)?([A-Za-z0-9]+)\??\s*[:(<]/);
      if (match) {
        if (/:\s*\{\s*$/.test(trimmed)) {
          current = match[1];
          surface[current] = new Set();
        } else {
          surface[match[1]] = null;
        }
      }
    } else if (depth === 2 && current && !trimmed.startsWith('*')) {
      const match = trimmed.match(/^(?:readonly\s+)?([A-Za-z0-9]+)\??\s*[:(<]/);
      if (match) surface[current].add(match[1]);
    }
    for (const ch of line) {
      if (ch === '{') depth += 1;
      if (ch === '}') depth -= 1;
      if (ch === '(') parens += 1;
      if (ch === ')') parens -= 1;
    }
    if (depth === 1) current = null;
    if (depth === 0) break;
  }
  return surface;
}

const host = hostSurface(fs.readFileSync(hostPath, 'utf8'));
const sdk = sdkSurface(fs.readFileSync(sdkPath, 'utf8'));
const problems = [];

for (const name of Object.keys(host)) {
  if (HOST_ONLY[name]) continue;
  if (!(name in sdk)) {
    problems.push(`api.${name} is provided by the host but not declared in the SDK`);
    continue;
  }
  if (host[name] instanceof Set && sdk[name] instanceof Set) {
    for (const member of host[name]) {
      if (!sdk[name].has(member)) problems.push(`api.${name}.${member} is provided by the host but not declared in the SDK`);
    }
    for (const member of sdk[name]) {
      if (!host[name].has(member)) problems.push(`api.${name}.${member} is declared in the SDK but the host does not provide it`);
    }
  }
}
for (const name of Object.keys(sdk)) {
  if (!(name in host) && name !== 'dispose') problems.push(`api.${name} is declared in the SDK but the host does not provide it`);
}

assert.ok(Object.keys(host).length > 10, 'host surface parse found too little; the parser no longer matches VerstakPluginAPI.js');
assert.ok(Object.keys(sdk).length > 10, 'SDK surface parse found too little; the parser no longer matches plugin-api.ts');
assert.deepEqual(problems, [], problems.join('\n'));
console.log(`SDK declares the host plugin API: ${Object.keys(host).length} namespaces`);
