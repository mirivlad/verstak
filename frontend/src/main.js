import App from './App.svelte';
import * as Backend from '../wailsjs/go/api/App';
import { i18n } from './lib/i18n/index.js';

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
  try {
    const settings = await Backend.GetAppSettings();
    await i18n.initialize(settings?.language || 'system');
  } catch (error) {
    console.error('[i18n] initialization failed:', error);
    await i18n.initialize('system');
  }

  return new App({
    target: document.getElementById('app'),
  });
}

const app = start();

export default app;
