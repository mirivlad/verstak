import { writable } from 'svelte/store';
import * as App from '../../../wailsjs/go/api/App';
import { debug } from '../log/debug.js';

const media = window.matchMedia('(prefers-color-scheme: dark)');
let current = { preference: 'system', resolved: media.matches ? 'dark' : 'light', saving: false, error: false };
export const theme = writable(current);

function publish(patch) {
  current = { ...current, ...patch };
  current.resolved = current.preference === 'system' ? (media.matches ? 'dark' : 'light') : current.preference;
  document.documentElement.dataset.theme = current.resolved;
  document.documentElement.style.colorScheme = current.resolved;
  theme.set(current);
}

export function initializeTheme(preference) {
  publish({ preference: ['light', 'dark', 'system'].includes(preference) ? preference : 'system', error: false });
}

export async function setTheme(preference) {
  if (current.saving || !['light', 'dark', 'system'].includes(preference)) return;
  publish({ saving: true, error: false });
  try {
    const error = await App.UpdateAppSettings({ theme: preference });
    if (error) throw new Error(error);
    publish({ preference, saving: false });
  } catch (error) {
    debug.log('Theme', 'Could not save appearance', String(error));
    publish({ saving: false, error: true });
  }
}

media.addEventListener('change', () => {
  if (current.preference === 'system') publish({});
});
