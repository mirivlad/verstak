import App from './App.svelte';
import * as Backend from '../wailsjs/go/api/App';
import { i18n } from './lib/i18n/index.js';
import './lib/ui/design-system.css';

// Replaced with a literal by Vite; false outside `--mode test`, so the mock
// imports below are removed from the bundle rather than shipped and skipped.
/* global __VERSTAK_TEST_MOCK__ */
const TEST_MOCK_ENABLED = __VERSTAK_TEST_MOCK__;

function backendAvailable() {
  return Boolean(window.go && window.go.api);
}

// Without the Wails runtime there is no vault, no plugins and no settings.
// Booting anyway used to fall through to the test mock, which answers every
// call with plausible fixtures — the user would be shown a vault that does not
// exist. Say so instead.
function renderRuntimeMissing() {
  const target = document.getElementById('app');
  if (!target) return;
  target.innerHTML = '';
  const box = document.createElement('div');
  box.className = 'startup-failure';
  box.setAttribute('role', 'alert');
  box.setAttribute('data-startup-failure', 'runtime-missing');

  const title = document.createElement('p');
  title.className = 'startup-failure-title';
  title.textContent = 'Verstak could not start';

  const detail = document.createElement('p');
  detail.textContent =
    'The application backend did not load, so no vault can be opened. '
    + 'Please restart Verstak. If it keeps happening, run it with --debug and '
    + 'share the log from ~/.local/share/verstak/debug/.';

  box.append(title, detail);
  target.appendChild(box);
}

function unpack(result) {
  if (Array.isArray(result) && result.length === 2 && (typeof result[1] === 'string' || result[1] == null)) {
    if (result[1]) throw new Error(result[1]);
    return result[0];
  }
  return result;
}

i18n.configure({
  loadPluginCatalog: async (pluginId, locale) => unpack(await Backend.GetPluginLocalization(pluginId, locale)),
});

async function start() {
  if (TEST_MOCK_ENABLED && !backendAvailable()) {
    await import('./lib/test/wails-mock.js');
    // Search-provider support was added after the original monolithic Wails
    // fixture. Keep that compatibility seam isolated and backed by the real
    // official plugin manifests/bundle instead of duplicating provider logic.
    await import('./lib/test/search-provider-mock.js');
    await import('./lib/test/capability-operation-mock.js');
  }

  if (!backendAvailable()) {
    renderRuntimeMissing();
    return null;
  }

  try {
    const settings = await Backend.GetAppSettings();
    await i18n.initialize(settings?.language || 'system');
  } catch (error) {
    console.error('[i18n] initialization failed:', error);
    await i18n.initialize('system');
  }

  // Keep the document language in step with the selected application
  // language: screen readers, spell checking and hyphenation all read it, and
  // a hardcoded value is wrong for every user who switches.
  syncDocumentLanguage();
  i18n.subscribe(syncDocumentLanguage);

  return new App({
    target: document.getElementById('app'),
  });
}

function syncDocumentLanguage() {
  document.documentElement.lang = i18n.getLocale() || 'en';
}

const app = start();

export default app;
