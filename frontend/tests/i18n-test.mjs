import assert from 'node:assert/strict';
import {
  createI18n,
  resolveLocale,
} from '../src/lib/i18n/index.js';
import englishShellCatalog from '../src/lib/i18n/catalogs/en.js';
import russianShellCatalog from '../src/lib/i18n/catalogs/ru.js';

assert.deepEqual(
  Object.keys(russianShellCatalog).sort(),
  Object.keys(englishShellCatalog).sort(),
  'English and Russian shell catalogs must have identical keys',
);

assert.equal(resolveLocale('system', ['ru-RU']), 'ru');
assert.equal(resolveLocale('system', ['uk-UA', 'en-US']), 'en');
assert.equal(resolveLocale('ru', ['en-US']), 'ru');
assert.equal(resolveLocale('en', ['ru-RU']), 'en');
assert.throws(() => resolveLocale('de', ['de-DE']), /unsupported language/);

const catalogs = {
  'localized.plugin': {
    en: { 'manifest.name': 'Localized Plugin', greeting: 'Hello, {name}!' },
    ru: {
      'manifest.name': 'Локализованный плагин',
      'contributions.views.localized.view.title': 'Локализованный экран',
      greeting: 'Привет, {name}!',
    },
  },
};
const loads = [];
const service = createI18n({
  shellCatalogs: {
    en: { loading: 'Loading {name}...', fallbackOnly: 'English fallback' },
    ru: { loading: 'Загрузка {name}...' },
  },
  systemLanguages: () => ['ru-RU'],
  loadPluginCatalog: async (pluginId, locale) => {
    loads.push(`${pluginId}:${locale}`);
    return catalogs[pluginId]?.[locale] || {};
  },
});

await service.initialize('system');
assert.equal(service.getLanguagePreference(), 'system');
assert.equal(service.getLocale(), 'ru');
assert.equal(service.t('loading', { name: 'Верстак' }), 'Загрузка Верстак...');
assert.equal(service.t('fallbackOnly'), 'English fallback');
assert.equal(service.t('missing', undefined, 'Explicit fallback'), 'Explicit fallback');
assert.equal(service.t('unknown'), 'unknown');

await service.loadPlugin('localized.plugin', {
  defaultLocale: 'en',
  locales: { en: 'locales/en.json', ru: 'locales/ru.json' },
});
assert.deepEqual(loads.sort(), ['localized.plugin:en', 'localized.plugin:ru']);
assert.equal(service.translatePlugin('localized.plugin', 'greeting', { name: 'Мир' }), 'Привет, Мир!');

const plugin = {
  manifest: {
    id: 'localized.plugin',
    name: 'Localized Plugin',
    description: 'Literal description',
    contributes: {
      views: [{ id: 'localized.view', title: 'Literal View', component: 'View' }],
    },
  },
};
const localized = service.localizePlugin(plugin);
assert.notEqual(localized, plugin);
assert.equal(localized.manifest.name, 'Локализованный плагин');
assert.equal(localized.manifest.description, 'Literal description');
assert.equal(localized.manifest.contributes.views[0].title, 'Локализованный экран');
assert.equal(plugin.manifest.name, 'Localized Plugin');
const summary = service.localizeContributionSummary({
  views: [{ pluginId: 'localized.plugin', id: 'localized.view', title: 'Literal View', component: 'View' }],
});
assert.equal(summary.views[0].title, 'Локализованный экран');

let notifications = 0;
const unsubscribe = service.subscribe(() => { notifications += 1; });
assert.equal(notifications, 1);
await service.setLanguagePreference('en');
assert.equal(service.getLocale(), 'en');
assert.equal(service.translatePlugin('localized.plugin', 'greeting', { name: 'World' }), 'Hello, World!');
assert.equal(notifications, 2);
unsubscribe();
await service.setLanguagePreference('ru');
assert.equal(notifications, 2);

const resilientService = createI18n({
  shellCatalogs: { en: { title: 'Title' }, ru: { title: 'Заголовок' } },
  systemLanguages: () => ['en-US'],
  loadPluginCatalog: async () => { throw new Error('broken catalog'); },
});
await resilientService.initialize('en');
await assert.rejects(
  resilientService.loadPlugin('broken.plugin', {
    defaultLocale: 'en',
    locales: { en: 'locales/en.json', ru: 'locales/ru.json' },
  }),
  /broken catalog/,
);
let resilientNotification = '';
resilientService.subscribe((nextLocale) => { resilientNotification = nextLocale; });
await resilientService.setLanguagePreference('ru');
assert.equal(resilientService.getLanguagePreference(), 'ru');
assert.equal(resilientService.getLocale(), 'ru');
assert.equal(resilientNotification, 'ru');
assert.equal(resilientService.t('title'), 'Заголовок');

console.log('i18n service tests passed');
