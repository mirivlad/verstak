import englishShellCatalog from './catalogs/en.js';
import russianShellCatalog from './catalogs/ru.js';

const LANGUAGE_PREFERENCES = new Set(['system', 'en', 'ru']);
const CONTRIBUTION_TEXT_FIELDS = {
  views: 'title',
  commands: 'title',
  settingsPanels: 'title',
  sidebarItems: 'title',
  fileActions: 'label',
  noteActions: 'label',
  contextMenuEntries: 'label',
  searchProviders: 'label',
  statusBarItems: 'label',
  openProviders: 'title',
  workspaceItems: 'title',
};

function normalizeSystemLanguages(languages) {
  const values = Array.isArray(languages) ? languages : [languages];
  return values.map((value) => String(value || '').trim().toLowerCase()).filter(Boolean);
}

export function resolveLocale(preference, systemLanguages = []) {
  if (!LANGUAGE_PREFERENCES.has(preference)) {
    throw new Error(`unsupported language preference: ${preference}`);
  }
  if (preference === 'en' || preference === 'ru') return preference;
  return normalizeSystemLanguages(systemLanguages).some((locale) => locale === 'ru' || locale.startsWith('ru-'))
    ? 'ru'
    : 'en';
}

function interpolate(message, params) {
  if (!params) return message;
  return message.replace(/\{([A-Za-z0-9_.-]+)\}/g, (placeholder, name) => (
    Object.prototype.hasOwnProperty.call(params, name) ? String(params[name]) : placeholder
  ));
}

function browserLanguages() {
  if (typeof navigator === 'undefined') return ['en'];
  if (Array.isArray(navigator.languages) && navigator.languages.length > 0) return navigator.languages;
  return [navigator.language || 'en'];
}

export function createI18n(options = {}) {
  const shellCatalogs = options.shellCatalogs || { en: {}, ru: {} };
  let systemLanguages = options.systemLanguages || browserLanguages;
  let catalogLoader = options.loadPluginCatalog || (async () => ({}));
  let preference = 'system';
  let locale = 'en';
  const listeners = new Set();
  const pluginConfigs = new Map();
  const pluginCatalogs = new Map();

  function configure(next = {}) {
    if (next.systemLanguages) systemLanguages = next.systemLanguages;
    if (next.loadPluginCatalog) catalogLoader = next.loadPluginCatalog;
  }

  function notify() {
    listeners.forEach((listener) => listener(locale));
  }

  async function initialize(initialPreference = 'system') {
    preference = LANGUAGE_PREFERENCES.has(initialPreference) ? initialPreference : 'system';
    locale = resolveLocale(preference, systemLanguages());
  }

  async function loadCatalog(pluginId, catalogLocale) {
    if (!catalogLocale) return;
    let catalogs = pluginCatalogs.get(pluginId);
    if (!catalogs) {
      catalogs = new Map();
      pluginCatalogs.set(pluginId, catalogs);
    }
    if (catalogs.has(catalogLocale)) return;
    const messages = await catalogLoader(pluginId, catalogLocale);
    catalogs.set(catalogLocale, { ...(messages || {}) });
  }

  async function loadPlugin(pluginId, localization) {
    if (!pluginId || !localization || !localization.defaultLocale || !localization.locales) return;
    pluginConfigs.set(pluginId, localization);
    const requested = localization.locales[locale] ? locale : '';
    const defaultLocale = localization.defaultLocale;
    await Promise.all(Array.from(new Set([requested, defaultLocale].filter(Boolean))).map((item) => loadCatalog(pluginId, item)));
  }

  async function setLanguagePreference(nextPreference) {
    const nextLocale = resolveLocale(nextPreference, systemLanguages());
    preference = nextPreference;
    locale = nextLocale;
    await Promise.all(Array.from(pluginConfigs.entries()).map(([pluginId, config]) => (
      loadPlugin(pluginId, config).catch((error) => {
        console.warn(`[i18n] failed to load ${pluginId} catalog for ${nextLocale}:`, error);
      })
    )));
    notify();
  }

  function t(key, params, fallback) {
    const message = shellCatalogs[locale]?.[key]
      ?? shellCatalogs.en?.[key]
      ?? fallback
      ?? key;
    return interpolate(message, params);
  }

  function translatePlugin(pluginId, key, params, fallback) {
    const config = pluginConfigs.get(pluginId);
    const catalogs = pluginCatalogs.get(pluginId);
    const message = catalogs?.get(locale)?.[key]
      ?? catalogs?.get(config?.defaultLocale)?.[key]
      ?? fallback
      ?? key;
    return interpolate(message, params);
  }

  function localizeContributions(pluginId, contributions) {
    if (!contributions) return contributions;
    const localized = { ...contributions };
    Object.entries(CONTRIBUTION_TEXT_FIELDS).forEach(([point, field]) => {
      if (!Array.isArray(contributions[point])) return;
      localized[point] = contributions[point].map((item) => ({
        ...item,
        [field]: translatePlugin(
          pluginId,
          `contributions.${point}.${item.id}.${field}`,
          undefined,
          item[field],
        ),
      }));
    });
    return localized;
  }

  function localizePlugin(plugin) {
    if (!plugin) return plugin;
    const hasStateWrapper = !!plugin.manifest;
    const manifest = hasStateWrapper ? plugin.manifest : plugin;
    const pluginId = manifest.id;
    const localizedManifest = {
      ...manifest,
      name: translatePlugin(pluginId, 'manifest.name', undefined, manifest.name),
      description: translatePlugin(pluginId, 'manifest.description', undefined, manifest.description),
      contributes: localizeContributions(pluginId, manifest.contributes),
    };
    return hasStateWrapper ? { ...plugin, manifest: localizedManifest } : localizedManifest;
  }

  function localizeContributionSummary(summary) {
    if (!summary) return summary;
    const localized = { ...summary };
    Object.entries(CONTRIBUTION_TEXT_FIELDS).forEach(([point, field]) => {
      if (!Array.isArray(summary[point])) return;
      localized[point] = summary[point].map((item) => ({
        ...item,
        [field]: translatePlugin(
          item.pluginId,
          `contributions.${point}.${item.id}.${field}`,
          undefined,
          item[field],
        ),
      }));
    });
    return localized;
  }

  function subscribe(listener) {
    listeners.add(listener);
    listener(locale);
    return () => listeners.delete(listener);
  }

  return {
    configure,
    initialize,
    getLanguagePreference: () => preference,
    getLocale: () => locale,
    setLanguagePreference,
    t,
    subscribe,
    loadPlugin,
    translatePlugin,
    localizeContributions,
    localizePlugin,
    localizeContributionSummary,
  };
}

export const i18n = createI18n({
  shellCatalogs: {
    en: englishShellCatalog,
    ru: russianShellCatalog,
  },
});

export const t = (...args) => i18n.t(...args);
