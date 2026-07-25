// Guards what must never reach a shipped bundle.
//
// The test mock answers every backend call with plausible fixtures. If it is
// present in production it is one failed runtime injection away from showing a
// user a vault that does not exist, so its absence is a contract, not a
// size optimisation.
import { execFileSync } from 'node:child_process';
import { readdirSync, readFileSync, existsSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert';

const frontend = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const dist = path.join(frontend, 'dist');

if (!existsSync(path.join(dist, 'index.html')) || process.env.VERSTAK_REBUILD_DIST === '1') {
  execFileSync('npx', ['vite', 'build'], { cwd: frontend, stdio: 'inherit' });
}

const assets = path.join(dist, 'assets');
const files = readdirSync(assets).filter((name) => name.endsWith('.js'));
assert.ok(files.length > 0, 'production build produced no JavaScript');

const offenders = files.filter((name) => {
  if (name.includes('wails-mock')) return true;
  const source = readFileSync(path.join(assets, name), 'utf8');
  return source.includes('__wailsMock') || source.includes('setWorkspaceTreeV2');
});
assert.deepStrictEqual(offenders, [], 'test mock leaked into the production bundle');

// The mock used to be reached from index.html, which meant a missing Wails
// runtime silently booted fixtures instead of reporting a failure.
const html = readFileSync(path.join(frontend, 'index.html'), 'utf8');
assert.doesNotMatch(html, /wails-mock/, 'index.html must not reference the test mock');

const main = readFileSync(path.join(frontend, 'src', 'main.js'), 'utf8');
assert.match(main, /__VERSTAK_TEST_MOCK__/, 'mock import must stay behind the build-time flag');
assert.match(main, /renderRuntimeMissing/, 'a missing backend must be reported, not worked around');

console.log(`production bundle contract: ok (${files.length} chunks checked)`);
