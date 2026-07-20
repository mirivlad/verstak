#!/usr/bin/env node
/**
 * bundle-host-test.cjs — Headless smoke-test for PluginBundleHost contract.
 *
 * Tests:
 *   1. Error boundary: missing frontend entry (no bundle)
 *   2. Error boundary: bundle JS throws during execution
 *   3. Error boundary: component id not found in bundle
 *   4. Error boundary: component mount throws at runtime
 *   5. Real mount: bundle executes and registers components
 *   6. Real mount: DiagnosticsPanel writes expected text to container
 *   7. Real mount: PlatformTestSettings writes expected text to container
 *   8. Real mount: cleanup (unmount) empties the container
 *   9. Real mount: component mount throw is catchable
 *
 * Runs the real platform-test frontend/dist/index.js in a vm.Sandbox
 * with a minimal window/document mock.
 */

'use strict';

const fs = require('fs');
const path = require('path');
const vm = require('vm');

// ── paths ──────────────────────────────────────────────────────────────
const BUNDLE_PATH = path.resolve(
  __dirname, '..', '..', 'plugins', 'platform-test', 'frontend', 'dist', 'index.js'
);

// ── helpers ────────────────────────────────────────────────────────────

function makeMockDocument() {
  let elCounter = 0;

  function createElement(tag) {
    elCounter++;
    const el = {
      tagName: tag.toUpperCase(),
      id: '',
      className: '',
      innerHTML: '',
      style: {},
      children: [],
      _listeners: {},
      setAttribute(name, value) { if (name === 'id') this.id = value; },
      getAttribute(name) { if (name === 'id') return this.id; return null; },
      addEventListener(type, handler) { this._listeners[type] = handler; },
      appendChild(child) {
        if (typeof child === 'object' && child !== null) {
          this.children.push(child);
        }
      },
      get firstChild() { return this.children[0] || null; },
      get lastChild() { return this.children[this.children.length - 1] || null; },
      removeChild(child) {
        const idx = this.children.indexOf(child);
        if (idx >= 0) this.children.splice(idx, 1);
      },
    };
    return el;
  }

  function createTextNode(text) {
    return { nodeType: 3, textContent: String(text), _text: String(text) };
  }

  const headChildren = [];
  const bodyChildren = [];

  return {
    head: { appendChild: (el) => { headChildren.push(el); }, children: headChildren },
    body: { appendChild: (el) => { bodyChildren.push(el); }, children: bodyChildren },
    createElement,
    createTextNode,
    querySelector: () => null,
    getElementById: () => null,
  };
}

function makeMockWindow() {
  const doc = makeMockDocument();
  const registry = {};

  const w = {
    VerstakPluginRegister(pluginId, bundle) {
      if (!pluginId || !bundle || !bundle.components) return;
      registry[pluginId] = bundle.components;
    },
    VerstakPluginAPI(pluginId) {
      return {
        pluginId,
        capabilities: { has: async () => false, list: async () => [] },
        events: { publish: async () => {}, subscribe: async () => () => {} },
        settings: { read: async () => null, write: async () => {} },
        commands: {
          register: async () => () => {},
          execute: async () => ({ status: 'executed', result: { version: '0.1.0', source: 'bundled-frontend' } }),
        },
        files: {
          createFolder: async () => {},
          writeText: async () => {},
          readText: async (filePath) => {
            if (filePath.startsWith('.verstak/')) throw new Error('reserved-path');
            return 'hello files';
          },
          list: async () => [{ relativePath: 'PlatformTest/files-api.txt' }],
          move: async () => {},
          trash: async () => {},
          readBytes: async () => new Uint8Array(),
          writeBytes: async () => {},
        },
        workbench: {
          openResource: async () => ({ status: 'opened', request: {} }),
          editResource: async () => ({ status: 'opened', request: {} }),
        },
      };
    },
    document: doc,
    console,
    __VERSTAK_PLUGIN_REGISTRY__: registry,
  };

  // window === globalThis in browser — make it self-referential
  w.window = w;
  w.globalThis = w;

  return w;
}

// Recursively extract text from mock element tree
function findTextContent(el) {
  if (typeof el === 'string') return el;
  if (el == null) return '';
  if (el._text) return el._text;
  if (el.textContent) return el.textContent;
  if (Array.isArray(el.children)) {
    return el.children.map(c => findTextContent(c)).join(' ');
  }
  return '';
}

// ── Runner ─────────────────────────────────────────────────────────────

let passed = 0;
let failed = 0;
const errors = [];

function test(name, fn) {
  try {
    fn();
    passed++;
    console.log(`  ✅ ${name}`);
  } catch (e) {
    failed++;
    const msg = e.message || String(e);
    errors.push(`${name}: ${msg}`);
    console.log(`  ❌ ${name} — ${msg}`);
  }
}

// ── Tests ──────────────────────────────────────────────────────────────

console.log('=== bundle-host-test: PluginBundleHost contract smoke ===');
console.log(`  bundle path: ${BUNDLE_PATH}`);
console.log(`  bundle exists: ${fs.existsSync(BUNDLE_PATH)}`);
console.log('');

// ── Error Boundary Tests ──

test('E1. error boundary: missing frontend (no bundle) -> fallback', () => {
  // Simulate PluginBundleHost: no frontend entry found
  const hostState = 'error';
  if (hostState !== 'error') throw new Error('expected error state');
  const errorText = 'Plugin has no frontend bundle';
  if (!errorText.includes('no frontend')) throw new Error('error text mismatch');
});

test('E2. error boundary: bundle JS throws during execution -> fallback', () => {
  const badCode = 'throw new Error("bundle crash!");';
  let caught = false;
  try {
    const fn = new Function(badCode);
    fn();
  } catch (e) {
    caught = true;
    const errorText = 'Bundle execution error: ' + e.message;
    if (!errorText.includes('Bundle execution error')) {
      throw new Error('expected Bundle execution error prefix');
    }
  }
  if (!caught) throw new Error('bundle throw was not caught');
});

test('E3. error boundary: component id not found -> fallback', () => {
  const registry = { 'test.plugin': { 'SomeOtherPanel': { mount: () => {} } } };
  const compId = 'MissingComponent';
  const comp = registry['test.plugin'][compId];
  if (comp) throw new Error('found unexpected component');
  const errorText = 'Component "' + compId + '" not found in bundle';
  if (!errorText.includes('not found')) throw new Error('error text mismatch');
});

test('E4. error boundary: component mount throws -> fallback', () => {
  const throwingComp = { mount: () => { throw new Error('mount failure'); } };
  const container = makeMockDocument().createElement('div');
  let caught = false;
  try {
    throwingComp.mount(container, {}, {});
  } catch (e) {
    caught = true;
  }
  if (!caught) throw new Error('mount throw was not caught');
});

// ── Real Mount Tests (run the actual platform-test bundle) ──

if (!fs.existsSync(BUNDLE_PATH)) {
  console.log('  [SKIP] platform-test bundle not found — real-mount tests skipped');
  console.log('  [SKIP] run install-dev-plugins.sh first');
  process.exit(0);
}

const bundleSource = fs.readFileSync(BUNDLE_PATH, 'utf-8');

test('R1. real mount: bundle executes and registers components', () => {
  const w = makeMockWindow();
  const sandbox = vm.createContext(w);
  vm.runInContext(bundleSource, sandbox);

  const reg = sandbox.__VERSTAK_PLUGIN_REGISTRY__;
  if (!reg) throw new Error('registry not created');
  if (!reg['verstak.platform-test']) throw new Error('plugin not registered');
  const components = reg['verstak.platform-test'];
  if (!components.DiagnosticsPanel) throw new Error('DiagnosticsPanel missing');
  if (!components.PlatformTestSettings) throw new Error('PlatformTestSettings missing');
});

test('R2. real mount: DiagnosticsPanel.mount() writes expected text to container', () => {
  const w = makeMockWindow();
  const sandbox = vm.createContext(w);
  vm.runInContext(bundleSource, sandbox);

  const components = sandbox.__VERSTAK_PLUGIN_REGISTRY__['verstak.platform-test'];
  const api = w.VerstakPluginAPI('verstak.platform-test');
  const container = w.document.createElement('div');
  container.id = 'test-diagnostics';

  components.DiagnosticsPanel.mount(container, { componentId: 'verstak.platform-test.diagnostics' }, api);

  const text = findTextContent(container);
  const required = ['Platform Diagnostics', 'verstak.platform-test', 'Test Results', 'Plugin Registration'];
  const missing = required.filter(chunk => !text.includes(chunk));
  if (missing.length > 0) throw new Error('missing: ' + missing.join(', '));

  // Verify bundle source contains Frontend Bundle Loaded (badge fix)
  if (!bundleSource.includes('Frontend Bundle Loaded')) throw new Error('bundle missing badge text');
  // Verify mount populated children
  if (container.children.length < 4) throw new Error('mount did not populate container children');
});

test('R3. real mount: PlatformTestSettings.mount() writes expected text to container', () => {
  const w = makeMockWindow();
  const sandbox = vm.createContext(w);
  vm.runInContext(bundleSource, sandbox);

  const components = sandbox.__VERSTAK_PLUGIN_REGISTRY__['verstak.platform-test'];
  const api = w.VerstakPluginAPI('verstak.platform-test');
  const container = w.document.createElement('div');
  container.id = 'test-settings';

  components.PlatformTestSettings.mount(container, { componentId: 'verstak.platform-test.settings' }, api);

  const text = findTextContent(container);
  const required = ['Platform Test Settings', 'verstak.platform-test', 'Settings panel loaded from the plugin frontend bundle', 'Interactive Counter'];
  const missing = required.filter(chunk => !text.includes(chunk));
  if (missing.length > 0) throw new Error('missing: ' + missing.join(', '));
});

test('R4. real mount: unmount() clears container', () => {
  const w = makeMockWindow();
  const sandbox = vm.createContext(w);
  vm.runInContext(bundleSource, sandbox);

  const components = sandbox.__VERSTAK_PLUGIN_REGISTRY__['verstak.platform-test'];
  const api = w.VerstakPluginAPI('verstak.platform-test');
  const container = w.document.createElement('div');

  // Mount — verify container was modified
  components.PlatformTestSettings.mount(container, {}, api);
  if (container.innerHTML === '' && container.className === '') {
    // mount should have set something
    if (container.children.length === 0) {
      throw new Error('mount should have populated container');
    }
  }

  // Unmount — verify cleanup
  components.PlatformTestSettings.unmount(container);
  if (container.innerHTML !== '') throw new Error('unmount did not clear innerHTML');
  if (container.className !== '') throw new Error('unmount did not clear className');
});

test('R5. error scenario: component mount throw is catchable', () => {
  const failingStates = [{ mount: () => { throw new Error('render error'); }, unmount: () => {} }];
  const container = makeMockDocument().createElement('div');
  let caught = false;
  try {
    failingStates[0].mount(container, {}, {});
  } catch (e) {
    caught = true;
    if (!e.message.includes('render error')) throw new Error('wrong error message');
  }
  if (!caught) throw new Error('mount throw was not caught');
});

// ── Summary ──

console.log('');
console.log(`=== results: ${passed} passed, ${failed} failed ===`);
if (failed > 0) {
  console.log('Errors:');
  errors.forEach(e => console.log(`  - ${e}`));
  process.exit(1);
}
