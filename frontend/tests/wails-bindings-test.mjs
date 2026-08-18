// Every method the shell can call must have a generated binding.
//
// This used to be a hand-written list of ten method names. A list maintained by
// hand only catches what someone remembered to add to it, which is the same
// effort as remembering to regenerate the bindings — so it caught nothing. The
// set is derived from the Go source instead: add a method to App and forget
// `wails build`, and this fails by construction.
//
// The frontend imports App.js directly, so a missing export is not a stale
// artifact — it is a method the shell cannot call at all.
import fs from 'node:fs';
import path from 'node:path';

const bindings = fs.readFileSync(path.resolve('frontend/wailsjs/go/api/App.js'), 'utf8');

const apiDir = path.resolve('internal/api');
const goSource = fs
  .readdirSync(apiDir)
  .filter((name) => name.endsWith('.go') && !name.endsWith('_test.go'))
  .map((name) => fs.readFileSync(path.join(apiDir, name), 'utf8'))
  .join('\n');

// Wails binds exported methods on the bound struct, with the exception of the
// lifecycle hooks it calls itself.
const LIFECYCLE = new Set(['Startup', 'BeforeClose', 'DomReady', 'Shutdown', 'OnBeforeClose']);

const goMethods = [...goSource.matchAll(/^func \(a \*App\) ([A-Z][A-Za-z0-9]*)\(/gm)]
  .map((match) => match[1])
  .filter((name) => !LIFECYCLE.has(name));

const exported = new Set(
  [...bindings.matchAll(/^export function ([A-Za-z0-9]+)\(/gm)].map((match) => match[1]),
);

const missing = [...new Set(goMethods)].filter((name) => !exported.has(name)).sort();
if (missing.length) {
  throw new Error(
    `Wails bindings do not export ${missing.length} App method(s): ${missing.join(', ')}\n`
    + '  regenerate with: bash scripts/build.sh',
  );
}

console.log(`Wails bindings expose all ${new Set(goMethods).size} App methods`);
