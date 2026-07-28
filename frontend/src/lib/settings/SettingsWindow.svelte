<script>
  // Settings had no home. Language lived in a dropdown in the status bar, each
  // plugin's settings opened as a modal inside the Plugin Manager, and nothing
  // told a user where to look for a given option. This is the one place: a
  // section list on the left, the section itself on the right, and a search
  // box for people who know the name of a setting but not its home.
  import { onDestroy, onMount, tick } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import { i18n } from '../i18n/index.js';
  import Icon from '../ui/Icon.svelte';
  import PluginBundleHost from '../plugin-host/PluginBundleHost.svelte';
  import { GetPluginFrontendInfo } from '../../../wailsjs/go/api/App';

  export let requestedSection = '';

  const GENERAL = 'general';
  const PLUGINS = 'plugins';
  const DIAGNOSTICS = 'diagnostics';

  let locale = i18n.getLocale();
  const unsubscribeLocale = i18n.subscribe((next) => { locale = next; });
  $: tr = ((_) => (key, params, fallback) => i18n.t(key, params, fallback))(locale);

  let contributions = {};
  let plugins = [];
  let activeSection = GENERAL;
  let query = '';
  let selectedLanguage = i18n.getLanguagePreference();
  let pluginInfo = {};
  let listEl = null;
  let diagnostics = null;
  let diagnosticsReportPath = '';
  let diagnosticsError = '';
  let collectingDiagnostics = false;

  // Somebody reporting a problem should not have to be told to find a terminal
  // and pass a flag. The report is written where they can read it first.
  async function collectDiagnostics() {
    collectingDiagnostics = true;
    diagnosticsError = '';
    diagnosticsReportPath = '';
    try {
      const response = await App.CollectDiagnostics();
      const [path, err] = Array.isArray(response) ? response : [response, ''];
      if (err) diagnosticsError = err;
      else diagnosticsReportPath = path;
    } catch (error) {
      diagnosticsError = error?.message || String(error);
    } finally {
      collectingDiagnostics = false;
    }
  }

  $: pluginSections = (contributions.settingsPanels || [])
    .filter((panel) => enabledPluginIds.has(panel.pluginId))
    .map((panel) => ({
      id: `plugin:${panel.pluginId}:${panel.id}`,
      title: panel.title || panel.id,
      icon: panel.icon || 'settings',
      panel,
    }));

  $: enabledPluginIds = new Set(
    (plugins || [])
      .filter((plugin) => plugin.enabled && (plugin.status === 'loaded' || plugin.status === 'degraded'))
      .map((plugin) => plugin.manifest?.id)
  );

  $: sections = [
    { id: GENERAL, title: tr('settings.section.general', undefined, 'General'), icon: 'settings' },
    { id: PLUGINS, title: tr('settings.pluginManager'), icon: 'puzzle' },
    ...pluginSections,
    { id: DIAGNOSTICS, title: tr('settings.section.diagnostics', undefined, 'Diagnostics'), icon: 'info' },
  ];

  // Search matches a section by its own name and, for the built-in ones, by the
  // names of the settings inside it -- somebody looking for "language" should
  // not have to know it lives under General.
  const searchTerms = {
    [GENERAL]: () => [tr('settings.language'), tr('settings.section.appearance', undefined, 'Appearance')],
    [PLUGINS]: () => [tr('settings.section.pluginsHint', undefined, 'install, enable, disable, permissions')],
    [DIAGNOSTICS]: () => [tr('settings.section.diagnosticsHint', undefined, 'log, crash, report, bug, support')],
  };

  $: normalizedQuery = query.trim().toLowerCase();
  $: visibleSections = normalizedQuery
    ? sections.filter((section) => {
        const extra = searchTerms[section.id] ? searchTerms[section.id]() : [];
        return [section.title, ...extra]
          .join(' ')
          .toLowerCase()
          .includes(normalizedQuery);
      })
    : sections;

  $: activeSectionData = sections.find((section) => section.id === activeSection) || sections[0];

  onMount(async () => {
    await load();
    if (requestedSection) {
      activeSection = requestedSection;
    } else {
      const stored = await App.GetAppSettings().catch(() => ({}));
      const remembered = stored && stored.settingsSection;
      if (remembered && sections.some((section) => section.id === remembered)) {
        activeSection = remembered;
      }
    }
    await ensurePluginInfo();
  });

  onDestroy(unsubscribeLocale);

  $: if (requestedSection && requestedSection !== activeSection) {
    activeSection = requestedSection;
  }

  async function load() {
    const [rawContributions, rawPlugins, rawDiagnostics] = await Promise.all([
      App.GetContributions().catch(() => ({})),
      App.GetPlugins().catch(() => []),
      App.GetDiagnosticsInfo().catch(() => null),
    ]);
    diagnostics = rawDiagnostics;
    await Promise.all((rawPlugins || []).map((plugin) => (
      i18n.loadPlugin(plugin.manifest?.id, plugin.manifest?.localization).catch(() => {})
    )));
    contributions = i18n.localizeContributionSummary(rawContributions || {});
    plugins = (rawPlugins || []).map((plugin) => i18n.localizePlugin(plugin));
  }

  async function ensurePluginInfo() {
    const wanted = pluginSections.map((section) => section.panel.pluginId);
    for (const pluginId of new Set(wanted)) {
      if (pluginInfo[pluginId] !== undefined) continue;
      try {
        pluginInfo[pluginId] = await GetPluginFrontendInfo(pluginId);
      } catch {
        pluginInfo[pluginId] = null;
      }
    }
    pluginInfo = pluginInfo;
  }

  async function selectSection(id) {
    activeSection = id;
    await ensurePluginInfo();
    // Remembered so reopening settings returns where the user was.
    App.UpdateAppSettings({ settingsSection: id }).catch(() => {});
  }

  async function selectLanguage(language) {
    const err = await App.UpdateAppSettings({ language });
    if (err) return;
    await i18n.setLanguagePreference(language);
    selectedLanguage = language;
  }

  function openPluginManager() {
    window.dispatchEvent(new CustomEvent('verstak:nav', { detail: { viewId: 'plugin-manager' } }));
  }

  function close() {
    window.dispatchEvent(new CustomEvent('verstak:close-settings-window'));
  }

  // Up and Down move between sections without leaving the list, Home and End
  // jump to its ends: the vertical tablist pattern, which is what this is.
  async function onListKeydown(event) {
    const index = visibleSections.findIndex((section) => section.id === activeSection);
    let next = -1;
    if (event.key === 'ArrowDown') next = (index + 1) % visibleSections.length;
    else if (event.key === 'ArrowUp') next = (index - 1 + visibleSections.length) % visibleSections.length;
    else if (event.key === 'Home') next = 0;
    else if (event.key === 'End') next = visibleSections.length - 1;
    else return;
    if (next < 0 || !visibleSections[next]) return;
    event.preventDefault();
    await selectSection(visibleSections[next].id);
    await tick();
    const button = listEl?.querySelector(`[data-settings-section="${CSS.escape(visibleSections[next].id)}"]`);
    button?.focus();
  }

  function onRootKeydown(event) {
    if (event.key === 'Escape') {
      event.stopPropagation();
      close();
    }
  }
</script>

<svelte:window on:keydown={onRootKeydown} />

<div class="settings-window vt-page" data-settings-window>
  <header class="settings-window-header">
    <h2>{tr('settings.title')}</h2>
    <button class="settings-window-close" type="button" data-settings-window-close on:click={close} aria-label={tr('common.close', undefined, 'Close')}>
      <Icon name="x" size={16} />
    </button>
  </header>

  <div class="settings-window-body">
    <nav class="settings-nav" aria-label={tr('settings.title')}>
      <input
        class="settings-search vt-input"
        type="search"
        data-settings-search
        bind:value={query}
        placeholder={tr('settings.search', undefined, 'Search settings')}
        aria-label={tr('settings.search', undefined, 'Search settings')}
      />
      <div
        class="settings-section-list"
        role="tablist"
        aria-orientation="vertical"
        bind:this={listEl}
        on:keydown={onListKeydown}
      >
        {#each visibleSections as section (section.id)}
          <button
            class="settings-section-item"
            class:is-active={section.id === activeSection}
            type="button"
            role="tab"
            aria-selected={section.id === activeSection}
            tabindex={section.id === activeSection ? 0 : -1}
            data-settings-section={section.id}
            on:click={() => selectSection(section.id)}
          >
            <Icon name={section.icon} size={14} />
            <span>{section.title}</span>
          </button>
        {/each}
        {#if visibleSections.length === 0}
          <p class="settings-nav-empty" data-settings-no-matches>
            {tr('settings.noMatches', undefined, 'No setting matches that search')}
          </p>
        {/if}
      </div>
    </nav>

    <section class="settings-content" role="tabpanel" aria-label={activeSectionData?.title || ''} data-settings-content>
      {#if activeSection === GENERAL}
        <h3>{tr('settings.section.general', undefined, 'General')}</h3>
        <div class="settings-group">
          <div class="settings-group-title" id="settings-language-label">{tr('settings.language')}</div>
          <div class="settings-choices" role="radiogroup" aria-labelledby="settings-language-label">
            {#each ['system', 'en', 'ru'] as language}
              <button
                class="settings-choice"
                class:is-active={selectedLanguage === language}
                type="button"
                role="radio"
                aria-checked={selectedLanguage === language}
                data-settings-language={language}
                on:click={() => selectLanguage(language)}
              >{tr(`settings.language.${language}`)}</button>
            {/each}
          </div>
        </div>
      {:else if activeSection === PLUGINS}
        <h3>{tr('settings.pluginManager')}</h3>
        <p class="settings-hint">
          {tr('settings.pluginsDescription', undefined, 'Install, enable and disable plugins, and review what each one is allowed to do.')}
        </p>
        <button class="vt-button" type="button" data-settings-open-plugin-manager on:click={openPluginManager}>
          {tr('settings.openPluginManager', undefined, 'Open Plugin Manager')}
        </button>
      {:else if activeSection === DIAGNOSTICS}
        <h3>{tr('settings.section.diagnostics', undefined, 'Diagnostics')}</h3>
        <p class="settings-hint">
          {tr('settings.diagnosticsDescription', undefined, 'A report of what this build is, which plugins loaded and what the log says. It never includes the contents of your vault, your secrets or your sync key.')}
        </p>
        <div class="settings-group">
          <div class="settings-group-title">{tr('settings.diagnosticsLog', undefined, 'Session log')}</div>
          <p class="settings-hint" data-settings-diagnostics-log>{diagnostics?.logPath || tr('settings.diagnosticsNoLog', undefined, 'This run is not writing a log.')}</p>
        </div>
        <button
          class="vt-button"
          type="button"
          data-settings-collect-diagnostics
          disabled={collectingDiagnostics}
          on:click={collectDiagnostics}
        >
          {tr('settings.collectDiagnostics', undefined, 'Save a report')}
        </button>
        {#if diagnosticsReportPath}
          <p class="settings-hint" data-settings-diagnostics-report>
            {tr('settings.diagnosticsSaved', { path: diagnosticsReportPath }, `Saved to ${diagnosticsReportPath}`)}
          </p>
        {/if}
        {#if diagnosticsError}
          <p class="settings-hint settings-error" data-settings-diagnostics-error>{diagnosticsError}</p>
        {/if}
      {:else if activeSectionData?.panel}
        <!-- No heading of our own: the panel is the section, and most panels
             title themselves. The tabpanel's aria-label carries the section
             name for anyone who cannot see which nav item is selected. -->
        {#key activeSectionData.id}
          {#if pluginInfo[activeSectionData.panel.pluginId]?.entry}
            <PluginBundleHost
              pluginId={activeSectionData.panel.pluginId}
              componentId={activeSectionData.panel.component || activeSectionData.panel.id}
            />
          {:else}
            <p class="settings-hint" data-settings-panel-unavailable>
              {tr('pluginManager.settingsBundleUnavailable')}
            </p>
          {/if}
        {/key}
      {/if}
    </section>
  </div>
</div>

<style>
  .settings-window {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 0;
    width: min(100%, 1100px);
    margin: 0 auto;
  }

  .settings-window-header {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.85rem 1rem;
    border-bottom: 1px solid var(--vt-color-border);
  }

  .settings-window-header h2 {
    margin: 0;
    font-size: 1.05rem;
    color: var(--vt-color-text-primary);
  }

  .settings-window-close {
    margin-left: auto;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    /* The global button rule adds horizontal padding and a minimum height,
       which on a square icon button leaves no room for the icon at all. */
    padding: 0;
    min-height: 0;
    width: 1.9rem;
    height: 1.9rem;
    border: 1px solid var(--vt-color-border-strong);
    border-radius: var(--vt-radius-md);
    background: var(--vt-color-surface);
    color: var(--vt-color-text-secondary);
    cursor: pointer;
  }

  .settings-window-close:hover {
    color: var(--vt-color-text-primary);
    border-color: var(--vt-color-accent);
  }

  .settings-window-body {
    flex: 1;
    min-height: 0;
    display: flex;
  }

  .settings-nav {
    flex: 0 0 15rem;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    padding: 0.75rem;
    border-right: 1px solid var(--vt-color-border);
  }

  .settings-search {
    width: 100%;
  }

  .settings-section-list {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
  }

  .settings-section-item {
    display: flex;
    align-items: center;
    /* Buttons are centred by default in the design system; a list of sections
       reads as a list only when its rows start at the same edge. */
    justify-content: flex-start;
    gap: 0.5rem;
    width: 100%;
    min-height: 0;
    padding: 0.4rem 0.55rem;
    border: 1px solid transparent;
    border-radius: var(--vt-radius-md);
    background: transparent;
    color: var(--vt-color-text-secondary);
    font-size: 0.85rem;
    text-align: left;
    cursor: pointer;
  }

  .settings-section-item:hover {
    background: var(--vt-color-surface-hover);
    color: var(--vt-color-text-primary);
  }

  .settings-section-item.is-active {
    background: var(--vt-color-surface-selected);
    border-color: var(--vt-color-accent);
    color: var(--vt-color-text-primary);
  }

  .settings-section-item:focus-visible {
    outline: 0;
    box-shadow: var(--vt-focus-ring);
  }

  .settings-nav-empty {
    margin: 0.5rem 0.55rem;
    font-size: 0.8rem;
    color: var(--vt-color-text-muted);
  }

  .settings-content {
    flex: 1;
    min-width: 0;
    min-height: 0;
    overflow-y: auto;
    padding: 1rem 1.25rem;
  }

  .settings-content h3 {
    margin: 0 0 0.75rem;
    font-size: 0.95rem;
    color: var(--vt-color-text-primary);
  }

  .settings-group {
    margin-bottom: 1.25rem;
  }

  .settings-group-title {
    font-size: 0.8rem;
    color: var(--vt-color-text-secondary);
    margin-bottom: 0.4rem;
  }

  .settings-choices {
    display: flex;
    flex-wrap: wrap;
    gap: 0.35rem;
  }

  .settings-choice {
    padding: 0.32rem 0.7rem;
    border: 1px solid var(--vt-color-border-strong);
    border-radius: var(--vt-radius-md);
    background: var(--vt-color-surface);
    color: var(--vt-color-text-secondary);
    font-size: 0.82rem;
    cursor: pointer;
  }

  .settings-choice:hover {
    border-color: var(--vt-color-accent);
    color: var(--vt-color-text-primary);
  }

  .settings-choice.is-active {
    background: var(--vt-color-surface-selected);
    border-color: var(--vt-color-accent);
    color: var(--vt-color-text-primary);
  }

  .settings-choice:focus-visible {
    outline: 0;
    box-shadow: var(--vt-focus-ring);
  }

  .settings-error {
    color: var(--vt-color-danger, #e94560);
  }

  .settings-hint {
    margin: 0 0 0.75rem;
    font-size: 0.83rem;
    line-height: 1.45;
    color: var(--vt-color-text-secondary);
  }

  @container vt-content (max-width: 720px) {
    .settings-window-body {
      flex-direction: column;
    }

    .settings-nav {
      flex: 0 0 auto;
      max-height: 12rem;
      border-right: 0;
      border-bottom: 1px solid var(--vt-color-border);
    }
  }
</style>
