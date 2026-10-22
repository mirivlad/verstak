import assert from 'node:assert/strict';
import { unmountPluginComponent, waitForPluginCleanup } from '../src/lib/plugin-host/componentLifecycle.js';

let disposed = false;
unmountPluginComponent('sync', { unmount() {} }, {}, { dispose() { disposed = true; } });
assert.equal(disposed, true, 'synchronous components still release the API immediately');

let finish;
disposed = false;
unmountPluginComponent('editor', {
  unmount() {
    return new Promise(resolve => { finish = resolve; }).then(() => {
      assert.equal(disposed, false, 'the old API is available for the final write');
    });
  },
}, {}, { dispose() { disposed = true; } });
let ready = false;
const opening = waitForPluginCleanup('editor').then(() => { ready = true; });
await waitForPluginCleanup('other-plugin');
assert.equal(ready, false, 'a new host waits for the same plugin, without blocking other plugins');
finish();
await opening;
assert.equal(disposed, true);

// Two live hosts may unmount the same plugin. Finishing one must not let a
// replacement mount while the other is still preserving state.
const releases = [];
for (let index = 0; index < 2; index++) {
  unmountPluginComponent('multiple', {
    unmount: () => new Promise(resolve => releases.push(resolve)),
  }, {}, { dispose() {} });
}
let replacementMounted = false;
const replacement = waitForPluginCleanup('multiple').then(() => { replacementMounted = true; });
releases[1]();
await Promise.resolve();
assert.equal(replacementMounted, false);
releases[0]();
await replacement;

const errors = [];
const originalError = console.error;
console.error = (...args) => errors.push(args);
try {
  for (const asynchronous of [false, true]) {
    disposed = false;
    unmountPluginComponent('failed', {
      unmount() {
        if (asynchronous) return Promise.reject(new Error('flush failed'));
        throw new Error('unmount failed');
      },
    }, {}, { dispose() { disposed = true; } });
    await waitForPluginCleanup('failed');
    assert.equal(disposed, true, 'cleanup failure still releases the API and allows reopening');
  }
  assert.equal(errors.length, 2);
} finally { console.error = originalError; }
console.log('component lifecycle: sync, async, multiple hosts and failure cleanup passed');
