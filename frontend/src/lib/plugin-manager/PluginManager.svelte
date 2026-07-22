<script>
  import Icon from '../ui/Icon.svelte';
  import PluginCard from './PluginCard.svelte';
  import PluginBundleHost from '../plugin-host/PluginBundleHost.svelte';
  import { onDestroy, onMount, tick } from 'svelte';
  import { GetPlugins, GetCapabilities, GetPermissions, GetContributions, ReloadPlugins, GetVaultStatus, GetVaultPluginState, EnablePlugin, DisablePlugin, ReadPluginSettings, WritePluginSettings, GetPluginFrontendInfo, WriteFrontendLog } from '../../../wailsjs/go/api/App';
  import { debug } from '../log/debug.js';
  import { i18n } from '../i18n/index.js';

  let plugins = [];
  let capabilities = [];
  let permissions = [];
  let contributions = {};
  let loading = true;
  let error = '';
  let vaultStatus = { status: 'unknown', path: '', vaultId: '' };
  let vaultPluginState = { enabledPlugins: [], disabledPlugins: [], desiredPlugins: [] };
  let settingsPanel = null;
  let settingsData = {};
  let settingsPluginId = '';
  let settingsHost = null;
  let settingsError = null;
  let settingsPluginInfo = null;
  let lastOpenedKey = '';
  let locale = i18n.getLocale();
  let unsubscribeLocale = null;
  $: tr = ((activeLocale) => (key, params, fallback) => {
    void activeLocale;
    return i18n.t(key, params, fallback);
  })(locale);

  // Per-action loading state — shows feedback on specific buttons without hiding the whole list
  let actionFeedback = {}; // { [pluginId]: 'enabling' | 'disabling' | null }
  let reloading = false;
  let toastMessage = '';
  let toastType = 'success'; // 'success' | 'error' | 'info'
  let statusFilter = 'all';
  let permissionFilters = [];
  let capabilityFilters = [];
  let settingsFilter = 'all';
  let sourceFilter = 'all';

  export let activeSettingsPluginId = '';
  export let activeSettingsPanelId = '';

  $: {
    if (activeSettingsPluginId) {
      const settingsPanelCount = (contributions.settingsPanels || []).length;
      const key = `${activeSettingsPluginId}:${activeSettingsPanelId || '*'}`;
      if (key !== lastOpenedKey || settingsPanelCount === 0) {
        openSettingsFromProps(activeSettingsPluginId, activeSettingsPanelId);
      }
    } else {
      lastOpenedKey = '';
    }
  }

  function showToast(msg, type = 'success') {
    toastMessage = msg;
    toastType = type;
    setTimeout(() => {
      toastMessage = '';
    }, 4000);
  }

  function reportError(key, fallback, details) {
    debug.log('[PluginManager] ' + key + ':', String(details));
    WriteFrontendLog('PluginManager', key + ': ' + String(details)).catch(() => {});
    return tr(key, undefined, fallback);
  }

  function notifyPluginsChanged() {
    window.dispatchEvent(new CustomEvent('verstak:plugins-changed'));
  }

  function unpackBackendResult(result) {
    if (Array.isArray(result) && result.length === 2 && (typeof result[1] === 'string' || result[1] == null)) {
      return { value: result[0], error: result[1] || '' };
    }
    return { value: result, error: '' };
  }

  function unpackReloadResult(result) {
    if (Array.isArray(result)) {
      return {
        count: Number(result[0] || 0),
        summary: result[1] || `Reloaded ${Number(result[0] || 0)} plugin(s).`,
      };
    }
    const count = Number(result || 0);
    return { count, summary: `Reloaded ${count} plugin(s).` };
  }

  async function openSettingsFromProps(pluginId, panelId) {
    const panel = (contributions.settingsPanels || []).find(sp => sp.pluginId === pluginId && (!panelId || sp.id === panelId));
    if (panel) {
      lastOpenedKey = `${pluginId}:${panelId || '*'}`;
      settingsPanel = panel;
      settingsPluginId = pluginId;
      settingsError = null;
      try {
        const info = await GetPluginFrontendInfo(pluginId);
        settingsPluginInfo = info;
      } catch { settingsPluginInfo = null; }
      ReadPluginSettings(pluginId).then(result => {
        const unpacked = unpackBackendResult(result);
        if (unpacked.error) {
          settingsError = reportError('pluginManager.settingsLoadError', 'Could not load plugin settings. Please try again.', unpacked.error);
          settingsData = {};
          return;
        }
        settingsData = unpacked.value || {};
      }).catch(() => { settingsData = {}; });
    } else {
      settingsError = tr('pluginManager.settingsUnavailable', undefined, 'Plugin settings are unavailable.');
    }
  }

  $: vaultOpen = vaultStatus.status === 'open';
  $: missingInstalled = computeMissingInstalled();

  function computeMissingInstalled() {
    if (!vaultPluginState.desiredPlugins) return [];
    const installedIDs = new Set(plugins.map(p => p.manifest?.id).filter(Boolean));
    return (vaultPluginState.desiredPlugins || []).filter(dp => !installedIDs.has(dp.id));
  }

  async function loadAll() {
    debug.log('[PluginManager] loadAll: START');
    error = '';
    loading = true;
    try {
      debug.log('[PluginManager] loadAll: calling GetPlugins...');
      const p = await GetPlugins();
      await Promise.all((p || []).map((plugin) => (
        i18n.loadPlugin(plugin.manifest?.id, plugin.manifest?.localization).catch(() => {})
      )));
      plugins = (p || []).map((plugin) => i18n.localizePlugin(plugin));
      debug.log('[PluginManager] loadAll: GetPlugins returned', plugins.length, 'plugins');
      for (var i = 0; i < plugins.length; i++) {
        debug.log('[PluginManager] loadAll: plugin[' + i + ']:', plugins[i].manifest?.id, 'status:', plugins[i].status, 'enabled:', plugins[i].enabled);
      }
    } catch (e) {
      debug.log('[PluginManager] loadAll: GetPlugins ERROR:', String(e));
      WriteFrontendLog('PluginManager', 'loadAll: GetPlugins ERROR: ' + String(e));
      error = reportError('pluginManager.loadError', 'Could not load plugins. Please try again.', e);
      loading = false;
      return;
    }
    // Collect all async loads but await them so loading stays true until all are done
    try {
      debug.log('[PluginManager] loadAll: loading vault/capabilities/permissions/contributions...');
      const [v, caps, perms, contribs] = await Promise.all([
        GetVaultStatus().catch(() => ({ status: 'unknown', path: '', vaultId: '' })),
        GetCapabilities().catch(() => []),
        GetPermissions().catch(() => []),
        GetContributions().catch(() => ({})),
      ]);
      vaultStatus = v || { status: 'unknown', path: '', vaultId: '' };
      capabilities = caps || [];
      permissions = perms || [];
      contributions = i18n.localizeContributionSummary(contribs || {});
      debug.log('[PluginManager] loadAll: vault=' + vaultStatus.status + ' caps=' + capabilities.length + ' perms=' + permissions.length);
      WriteFrontendLog('PluginManager', 'loadAll: vault=' + vaultStatus.status + ' caps=' + capabilities.length + ' perms=' + permissions.length);
    } catch (e) {
      debug.log('[PluginManager] loadAll: non-critical load ERROR:', String(e));
      WriteFrontendLog('PluginManager', 'loadAll: non-critical ERROR: ' + String(e));
      console.error('[PluginManager] non-critical load error:', e);
    }
    if (vaultStatus.status === 'open') {
      try {
        debug.log('[PluginManager] loadAll: calling GetVaultPluginState...');
        vaultPluginState = await GetVaultPluginState() || { enabledPlugins: [], disabledPlugins: [], desiredPlugins: [] };
        WriteFrontendLog('PluginManager', 'loadAll: GetVaultPluginState returned');
      } catch (e) {
        WriteFrontendLog('PluginManager', 'loadAll: GetVaultPluginState ERROR: ' + String(e));
      }
    }
    loading = false;
    await tick();
    debug.log('[PluginManager] loadAll: END, loading=false');
    WriteFrontendLog('PluginManager', 'loadAll: END, loading=false');
  }

  onMount(() => {
    window.dispatchEvent(new CustomEvent('verstak:content-title-changed', {
      detail: { title: tr('settings.pluginManager') }
    }));
    unsubscribeLocale = i18n.subscribe((nextLocale) => {
      const changed = locale !== nextLocale;
      locale = nextLocale;
      if (changed) {
        window.dispatchEvent(new CustomEvent('verstak:content-title-changed', {
          detail: { title: tr('settings.pluginManager') }
        }));
        loadAll();
      }
    });
    loadAll();
  });

  onDestroy(() => {
    if (unsubscribeLocale) unsubscribeLocale();
  });

  async function reload() {
    debug.log('[PluginManager] reload: START');
    reloading = true;
    error = '';
    let resultMsg = '';
    try {
      debug.log('[PluginManager] reload: calling ReloadPlugins...');
      const { count, summary } = unpackReloadResult(await ReloadPlugins());
      debug.log('[PluginManager] reload: ReloadPlugins returned count=' + count + ' summary=' + summary);
      resultMsg = `Reloaded ${count} plugin(s). ${summary}`;
    } catch (e) {
      debug.log('[PluginManager] reload: ReloadPlugins ERROR:', String(e));
      error = reportError('pluginManager.reloadError', 'Could not reload plugins. Please try again.', e);
      reloading = false;
      return;
    }
    debug.log('[PluginManager] reload: calling loadAll after reload...');
    await loadAll();
    notifyPluginsChanged();
    reloading = false;
    debug.log('[PluginManager] reload: END');
    showToast(resultMsg, 'success');
  }

  async function enablePlugin(pluginId) {
    debug.log('[PluginManager] enablePlugin:', pluginId);
    actionFeedback = { ...actionFeedback, [pluginId]: 'enabling' };
    error = '';
    const err = await EnablePlugin(pluginId);
    if (err) {
      debug.log('[PluginManager] enablePlugin: ERROR:', err);
      actionFeedback = { ...actionFeedback, [pluginId]: null };
      error = reportError('pluginManager.enableError', 'Could not enable the plugin. Please try again.', err);
      return;
    }
    debug.log('[PluginManager] enablePlugin: success, reloading...');
    // Reload to get updated state
    try { await ReloadPlugins(); } catch (e) { /* ignore */ }
    await loadAll();
    notifyPluginsChanged();
    actionFeedback = { ...actionFeedback, [pluginId]: null };
    debug.log('[PluginManager] enablePlugin: done');
    showToast(`Plugin "${pluginId}" enabled`, 'success');
  }

  async function disablePlugin(pluginId) {
    debug.log('[PluginManager] disablePlugin:', pluginId);
    actionFeedback = { ...actionFeedback, [pluginId]: 'disabling' };
    error = '';
    const err = await DisablePlugin(pluginId);
    if (err) {
      debug.log('[PluginManager] disablePlugin: ERROR:', err);
      actionFeedback = { ...actionFeedback, [pluginId]: null };
      error = reportError('pluginManager.disableError', 'Could not disable the plugin. Please try again.', err);
      return;
    }
    debug.log('[PluginManager] disablePlugin: success, reloading...');
    // Reload to get updated state
    try { await ReloadPlugins(); } catch (e) { /* ignore */ }
    await loadAll();
    notifyPluginsChanged();
    actionFeedback = { ...actionFeedback, [pluginId]: null };
    debug.log('[PluginManager] disablePlugin: done');
    showToast(`Plugin "${pluginId}" disabled`, 'info');
  }

  $: totalPlugins = plugins.length;
  $: totalCaps = capabilities.length;
  $: totalPerms = permissions.length;
  $: statusSummary = computeStatusSummary(plugins);
  $: elevatedPermissionPluginCount = computeElevatedPermissionPluginCount(plugins, permissions);
  $: filterPermissions = collectManifestValues(plugins, 'permissions');
  $: filterCapabilities = collectCapabilities(plugins);
  $: hasReliableSourceMetadata = plugins.some((plugin) => sourceGroup(plugin) !== '');
  $: visiblePlugins = filterPlugins(plugins, statusFilter, permissionFilters, capabilityFilters, settingsFilter, sourceFilter, contributions);
  $: filtersActive = statusFilter !== 'all' || permissionFilters.length > 0 || capabilityFilters.length > 0 || settingsFilter !== 'all' || sourceFilter !== 'all';

  function computeStatusSummary(pluginRows) {
    return pluginRows.reduce((summary, plugin) => {
      const status = plugin?.status || 'unknown';
      if (status === 'loaded') summary.loaded += 1;
      else if (status === 'degraded') summary.degraded += 1;
      else if (status === 'failed' || status === 'incompatible' || status === 'missing-required-capability') summary.failed += 1;
      else if (status === 'disabled' || plugin?.enabled === false) summary.disabled += 1;
      else summary.other += 1;
      return summary;
    }, { loaded: 0, degraded: 0, failed: 0, disabled: 0, other: 0 });
  }

  function computeElevatedPermissionPluginCount(pluginRows, permissionRows) {
    const dangerous = new Set((permissionRows || []).filter(permission => permission.dangerous).map(permission => permission.name));
    return (pluginRows || []).filter(plugin => (plugin?.manifest?.permissions || []).some(permission => dangerous.has(permission))).length;
  }

  function collectManifestValues(pluginRows, key) {
    return Array.from(new Set((pluginRows || []).flatMap((plugin) => plugin?.manifest?.[key] || [])))
      .sort((left, right) => left.localeCompare(right));
  }

  function collectCapabilities(pluginRows) {
    return Array.from(new Set((pluginRows || []).flatMap((plugin) => [
      ...(plugin?.manifest?.provides || []),
      ...(plugin?.manifest?.requires || []),
      ...(plugin?.manifest?.optionalRequires || []),
    ]))).sort((left, right) => left.localeCompare(right));
  }

  function pluginStatusGroup(plugin) {
    const status = plugin?.status || '';
    if (status === 'failed' || status === 'incompatible' || status === 'missing-required-capability') return 'failed';
    if (status === 'degraded') return 'degraded';
    if (status === 'disabled' || plugin?.enabled === false) return 'disabled';
    return 'enabled';
  }

  function sourceGroup(plugin) {
    const source = plugin?.manifest?.source;
    if (source === 'official') return 'official';
    if (source === 'local' || source === 'third-party') return 'third-party';
    return '';
  }

  function hasSettings(plugin, activeContributions) {
    const pluginId = plugin?.manifest?.id;
    return Boolean(pluginId && (activeContributions.settingsPanels || []).some((panel) => panel.pluginId === pluginId));
  }

  function includesAny(values, expected) {
    return expected.length === 0 || expected.some((value) => values.includes(value));
  }

  function filterPlugins(pluginRows, activeStatusFilter, activePermissionFilters, activeCapabilityFilters, activeSettingsFilter, activeSourceFilter, activeContributions) {
    return pluginRows.filter((plugin) => matchesFilters(plugin, activeStatusFilter, activePermissionFilters, activeCapabilityFilters, activeSettingsFilter, activeSourceFilter, activeContributions));
  }

  function matchesFilters(plugin, activeStatusFilter, activePermissionFilters, activeCapabilityFilters, activeSettingsFilter, activeSourceFilter, activeContributions) {
    if (activeStatusFilter !== 'all' && pluginStatusGroup(plugin) !== activeStatusFilter) return false;
    if (!includesAny(plugin?.manifest?.permissions || [], activePermissionFilters)) return false;
    const declaredCapabilities = [
      ...(plugin?.manifest?.provides || []),
      ...(plugin?.manifest?.requires || []),
      ...(plugin?.manifest?.optionalRequires || []),
    ];
    if (!includesAny(declaredCapabilities, activeCapabilityFilters)) return false;
    if (activeSettingsFilter === 'with' && !hasSettings(plugin, activeContributions)) return false;
    if (activeSettingsFilter === 'without' && hasSettings(plugin, activeContributions)) return false;
    return activeSourceFilter === 'all' || sourceGroup(plugin) === activeSourceFilter;
  }

  function resetFilters() {
    statusFilter = 'all';
    permissionFilters = [];
    capabilityFilters = [];
    settingsFilter = 'all';
    sourceFilter = 'all';
  }

  function closeSettings() {
    settingsHost?.dispose?.();
    settingsHost = null;
    settingsPanel = null;
    settingsPluginId = '';
    settingsError = null;
    window.dispatchEvent(new CustomEvent('verstak:close-settings'));
  }

  function saveSettings() {
    WritePluginSettings(settingsPluginId, settingsData).then(err => {
      if (err) console.error('WritePluginSettings:', err);
    }).catch(e => console.error('WritePluginSettings:', e));
  }
</script>

<div class="plugin-manager">
  <!-- Toast notification -->
  {#if toastMessage}
    <div class="toast" class:toast-success={toastType === 'success'} class:toast-error={toastType === 'error'} class:toast-info={toastType === 'info'}>
      {toastMessage}
    </div>
  {/if}

  <header>
    <div class="header-left">
      {#if vaultStatus.status !== 'unknown'}
        <span class="vault-badge" class:vault-open={vaultStatus.status === 'open'} class:vault-not-created={vaultStatus.status === 'not-created'} class:vault-closed={vaultStatus.status === 'closed'} class:vault-error={vaultStatus.status === 'error'}>
          {tr('vault.label', { status: tr(`vault.status.${vaultStatus.status}`, undefined, vaultStatus.status) })}{#if vaultStatus.path} ({vaultStatus.path}){/if}
        </span>
      {/if}
    </div>
    <button class="reload-btn" on:click={reload} type="button" disabled={loading || reloading}>
      {reloading ? tr('pluginManager.reloading') : tr('pluginManager.reload')}
    </button>
  </header>

  {#if loading}
    <div class="loading">{tr('pluginManager.scanning')}</div>
  {:else if error}
    <div class="error">
      <Icon name="warning" size={24} class="error-icon" />
      <div class="error-message">{error}</div>
      <button class="retry-btn" on:click={loadAll} type="button">{tr('common.retry')}</button>
    </div>
  {:else}
    <div class="summary">
      <span class="badge">{tr('pluginManager.summary.plugins', { count: totalPlugins })}</span>
      <span class="badge">{tr('pluginManager.summary.capabilities', { count: totalCaps })}</span>
      <span class="badge">{tr('pluginManager.summary.permissions', { count: totalPerms })}</span>
    </div>

    <div class="scan-summary" aria-label={tr('pluginManager.scanSummary')}>
      <section class="scan-card" data-plugin-manager-summary="health">
        <div class="scan-card-title">{tr('pluginManager.health')}</div>
        <div class="scan-metrics">
          <span data-plugin-status-summary="loaded"><strong>{statusSummary.loaded}</strong> {tr('status.loaded')}</span>
          <span data-plugin-status-summary="degraded"><strong>{statusSummary.degraded}</strong> {tr('status.degraded')}</span>
          <span data-plugin-status-summary="failed"><strong>{statusSummary.failed}</strong> {tr('status.failed')}</span>
          <span data-plugin-status-summary="disabled"><strong>{statusSummary.disabled}</strong> {tr('status.disabled')}</span>
        </div>
      </section>

      <section class="scan-card" data-plugin-manager-summary="risk">
        <div class="scan-card-title">{tr('pluginManager.permissionRisk')}</div>
        <div class="scan-metrics">
          <span data-plugin-risk-summary="elevated-permissions">{tr('pluginManager.elevatedPermissions', { count: elevatedPermissionPluginCount })}</span>
        </div>
      </section>
    </div>

    <section class="plugin-filters" aria-label={tr('pluginManager.filters')}>
      <div class="filter-header">
        <div>
          <h3>{tr('pluginManager.filters')}</h3>
          <p data-plugin-filter-results>{tr('pluginManager.filterResults', { visible: visiblePlugins.length, total: totalPlugins })}</p>
        </div>
        <button class="filter-reset" data-plugin-filter-reset type="button" on:click={resetFilters} disabled={!filtersActive}>{tr('pluginManager.filterReset')}</button>
      </div>
      <div class="filter-grid">
        <label class="filter-select-label">
          <span>{tr('pluginManager.filterState')}</span>
          <select class="filter-select" data-plugin-filter="status" bind:value={statusFilter}>
            <option value="all">{tr('pluginManager.filterAll')}</option>
            <option value="enabled">{tr('pluginManager.filterEnabled')}</option>
            <option value="disabled">{tr('pluginManager.filterDisabled')}</option>
            <option value="failed">{tr('pluginManager.filterFailed')}</option>
            <option value="degraded">{tr('pluginManager.filterDegraded')}</option>
          </select>
        </label>

        <label class="filter-select-label">
          <span>{tr('pluginManager.filterSettings')}</span>
          <select class="filter-select" data-plugin-filter="settings" bind:value={settingsFilter}>
            <option value="all">{tr('pluginManager.filterAll')}</option>
            <option value="with">{tr('pluginManager.filterWithSettings')}</option>
            <option value="without">{tr('pluginManager.filterWithoutSettings')}</option>
          </select>
        </label>

        {#if hasReliableSourceMetadata}
          <label class="filter-select-label">
            <span>{tr('pluginManager.filterSource')}</span>
            <select class="filter-select" data-plugin-filter="source" bind:value={sourceFilter}>
              <option value="all">{tr('pluginManager.filterAll')}</option>
              <option value="official">{tr('pluginManager.filterOfficial')}</option>
              <option value="third-party">{tr('pluginManager.filterThirdParty')}</option>
            </select>
          </label>
        {/if}
      </div>

      {#if filterPermissions.length > 0}
        <fieldset class="filter-options">
          <legend>{tr('pluginManager.filterPermissions')}</legend>
          <div class="filter-option-list">
            {#each filterPermissions as permission}
              <label><input data-plugin-filter-permission={permission} type="checkbox" bind:group={permissionFilters} value={permission} /> <code>{permission}</code></label>
            {/each}
          </div>
        </fieldset>
      {/if}

      {#if filterCapabilities.length > 0}
        <fieldset class="filter-options">
          <legend>{tr('pluginManager.filterCapabilities')}</legend>
          <div class="filter-option-list">
            {#each filterCapabilities as capability}
              <label><input data-plugin-filter-capability={capability} type="checkbox" bind:group={capabilityFilters} value={capability} /> <code>{capability}</code></label>
            {/each}
          </div>
        </fieldset>
      {/if}
    </section>

    {#if plugins.length === 0 && missingInstalled.length === 0}
      <div class="empty">
        <div class="empty-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
          </svg>
        </div>
        <p>{tr('pluginManager.none')}</p>
        <p class="hint">{tr('pluginManager.scannedDirs')}</p>
        <ul class="hint-list">
          <li><code>~/.config/verstak/plugins/</code> — {tr('pluginManager.userPlugins')}</li>
          <li><code>./plugins/</code> — {tr('pluginManager.bundledPlugins')}</li>
        </ul>
        <p class="hint">{tr('pluginManager.installHint')}</p>
      </div>
    {:else if visiblePlugins.length === 0}
      <div class="empty filter-empty" data-plugin-filter-empty>
        <p>{tr('pluginManager.filterEmpty')}</p>
      </div>
    {:else}
      <div class="plugin-list">
        {#each visiblePlugins as p}
          <PluginCard {p} {capabilities} {permissions} {contributions} {vaultOpen} {actionFeedback} settingsPanels={(contributions.settingsPanels || []).filter(sp => sp.pluginId === p.manifest?.id)} onEnable={enablePlugin} onDisable={disablePlugin} />
        {/each}
      </div>
    {/if}

    {#if missingInstalled.length > 0}
      <div class="missing-section">
        <h3>{tr('pluginManager.missingTitle')}</h3>
        <p class="missing-hint">{tr('pluginManager.missingHint')}</p>
        <div class="plugin-list">
          {#each missingInstalled as mp}
            <div class="plugin-card missing-card">
              <div class="card-header">
                <div class="plugin-id">
                  <span class="status-dot" style="background: #e94560"></span>
                  <strong>{mp.id}</strong>
                  {#if mp.version}<span class="version">v{mp.version}</span>{/if}
                </div>
                <span class="status-badge" style="color: #e94560">{tr('status.missing')}</span>
              </div>
              <p class="missing-text">
                {tr('pluginManager.missingPackage')}
                {#if mp.source && mp.source !== 'unknown'}
                  <span class="source-hint">{tr('common.source')}: {mp.source}</span>
                {/if}
              </p>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    {#if capabilities.length > 0}
      <details class="registry-section">
        <summary>{tr('pluginManager.capabilityRegistry', { count: totalCaps })}</summary>
        <table>
          <thead>
            <tr><th>{tr('common.capability')}</th><th>{tr('common.provider')}</th><th>{tr('common.source')}</th><th>{tr('common.status')}</th></tr>
          </thead>
          <tbody>
            {#each capabilities as cap}
              <tr>
                <td><code>{cap.name}</code></td>
                <td>{cap.pluginId}</td>
                <td><span class="source-badge" class:source-core={cap.pluginId === 'verstak-desktop'} class:source-plugin={cap.pluginId !== 'verstak-desktop'}>{cap.pluginId === 'verstak-desktop' ? 'core' : 'plugin'}</span></td>
                <td><span class="status-{cap.status}">{cap.status}</span></td>
              </tr>
            {/each}
          </tbody>
        </table>
      </details>
    {/if}
  {/if}

  <!-- Settings Panel Modal -->
  {#key `settings-${settingsPluginId}`}
  {#if settingsError}
  <div class="modal-overlay" on:click|self={closeSettings} on:keydown|self={(e) => e.key === 'Escape' && closeSettings()} role="presentation">
  <div class="modal" role="dialog" aria-modal="true" aria-label={tr('pluginManager.settingsError')}>
    <div class="modal-header">
      <h3>{tr('pluginManager.settingsError')}</h3>
      <button class="modal-close" on:click={closeSettings} type="button">✕</button>
    </div>
    <div class="modal-body">
      <p class="error" style="color: #e94560;">{settingsError}</p>
    </div>
  </div>
  </div>
  {:else if settingsPanel}
  <div class="modal-overlay" on:click|self={closeSettings} on:keydown|self={(e) => e.key === 'Escape' && closeSettings()} role="presentation">
  <div class="modal" role="dialog" aria-modal="true" aria-label={tr('pluginManager.pluginSettings')}>
    <div class="modal-header">
      <h3>{settingsPanel.title}</h3>
      <button class="modal-close" on:click={closeSettings} type="button">✕</button>
    </div>
    <div class="modal-body settings-modal-body">
      <div class="plugin-settings-surface">
        <p class="settings-hint">{tr('common.plugin')}: <code>{settingsPluginId}</code></p>
        {#if settingsPluginInfo && settingsPluginInfo.entry}
          <PluginBundleHost
            bind:this={settingsHost}
            pluginId={settingsPluginId}
            componentId={settingsPanel.component || settingsPanel.id}
          />
        {:else}
          <p class="settings-hint">{tr('common.component')}: <code>{settingsPanel.component}</code></p>
          <p class="placeholder">{tr('pluginManager.settingsBundleUnavailable')}</p>
        {/if}
      </div>
    </div>
  </div>
  </div>
  {/if}
  {/key}
</div>

<style>
  .plugin-manager {
    flex: 1;
    width: min(100%, 1100px);
    min-height: 0;
    padding: 0.5rem 0.5rem 1.5rem 0;
    position: relative;
  }
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    flex-wrap: wrap;
    margin-bottom: 1.25rem;
    padding-bottom: 0.75rem;
    border-bottom: 1px solid #0f3460;
  }
  .header-left {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    min-width: 0;
    flex-wrap: wrap;
  }
  .vault-badge {
    max-width: 100%;
    font-size: 0.75rem;
    padding: 0.2rem 0.6rem;
    border-radius: 12px;
    font-weight: 600;
    border: 1px solid;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .vault-open { background: rgba(78, 204, 163, 0.15); color: #4ecca3; border-color: #4ecca3; }
  .vault-not-created { background: rgba(255, 200, 87, 0.15); color: #ffc857; border-color: #ffc857; }
  .vault-closed { background: rgba(160, 160, 184, 0.15); color: #a0a0b8; border-color: #a0a0b8; }
  .vault-error { background: rgba(233, 69, 96, 0.15); color: #e94560; border-color: #e94560; }
  .reload-btn {
    background: #0f3460; color: #e0e0e0; border: 1px solid #533483;
    padding: 0.4rem 1rem; border-radius: 6px; cursor: pointer; font-size: 0.85rem;
  }
  .reload-btn:hover:not(:disabled) { background: #533483; }
  .reload-btn:disabled { opacity: 0.5; cursor: not-allowed; }

  /* Toast */
  .toast {
    position: fixed; top: 1rem; right: 1rem; z-index: 2000;
    padding: 0.6rem 1.2rem; border-radius: 6px; font-size: 0.85rem;
    max-width: 400px; word-break: break-word;
    animation: toastIn 0.25s ease-out;
  }
  .toast-success { background: #1a3a2e; color: #4ecca3; border: 1px solid #4ecca3; }
  .toast-error { background: #3a1a1a; color: #e94560; border: 1px solid #e94560; }
  .toast-info { background: #1a1a3a; color: #a78bfa; border: 1px solid #a78bfa; }
  @keyframes toastIn { from { opacity: 0; transform: translateY(-10px); } to { opacity: 1; transform: translateY(0); } }

  .loading, .error {
    padding: 2rem; text-align: center; color: #a0a0b8;
  }
  .error { color: #e94560; }
  :global(.error-icon) { color: #e94560; margin-bottom: 0.5rem; }
  .error-message {
    font-family: monospace; font-size: 0.85rem; margin-bottom: 1rem; word-break: break-word;
  }
  .retry-btn {
    background: #0f3460; color: #e0e0e0; border: 1px solid #533483;
    padding: 0.4rem 1rem; border-radius: 6px; cursor: pointer; font-size: 0.85rem;
  }
  .retry-btn:hover { background: #533483; }
  .summary {
    display: flex; gap: 0.5rem; margin-bottom: 1rem; flex-wrap: wrap;
  }
  .badge {
    background: #16213e; padding: 0.25rem 0.75rem; border-radius: 12px;
    font-size: 0.8rem; color: #a0a0b8; border: 1px solid #0f3460;
  }
  .scan-summary {
    display: grid;
    grid-template-columns: minmax(0, 1.35fr) minmax(0, 1fr);
    gap: 0.75rem;
    margin-bottom: 1rem;
  }
  .scan-card {
    min-width: 0;
    padding: 0.75rem;
    border: 1px solid #0f3460;
    border-radius: 8px;
    background: #121a2c;
  }
  .scan-card-title {
    color: #e0e0f0;
    font-size: 0.82rem;
    font-weight: 700;
    margin-bottom: 0.5rem;
  }
  .scan-metrics {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 0.45rem;
  }
  .scan-metrics span {
    min-height: 1.55rem;
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    padding: 0.18rem 0.5rem;
    border: 1px solid rgba(78, 204, 163, 0.16);
    border-radius: 5px;
    color: #a0a0b8;
    background: #101626;
    font-size: 0.76rem;
  }
  .scan-metrics strong {
    color: #f4f7fb;
    font-size: 0.88rem;
  }
  .plugin-filters {
    margin-bottom: 1rem;
    padding: 0.75rem;
    border: 1px solid #0f3460;
    border-radius: 8px;
    background: #121a2c;
  }
  .filter-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 0.75rem;
    margin-bottom: 0.75rem;
  }
  .filter-header h3 {
    margin: 0;
    color: #e0e0f0;
    font-size: 0.9rem;
  }
  .filter-header p {
    margin: 0.2rem 0 0;
    color: #a0a0b8;
    font-size: 0.78rem;
  }
  .filter-reset {
    flex: 0 0 auto;
    padding: 0.35rem 0.65rem;
    border: 1px solid #533483;
    border-radius: 5px;
    background: #16213e;
    color: #e0e0e0;
    cursor: pointer;
    font-size: 0.78rem;
  }
  .filter-reset:hover:not(:disabled) { background: #0f3460; }
  .filter-reset:disabled { cursor: not-allowed; opacity: 0.5; }
  .filter-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 0.65rem;
  }
  .filter-select-label {
    display: grid;
    gap: 0.28rem;
    min-width: 10rem;
    color: #a0a0b8;
    font-size: 0.76rem;
    font-weight: 600;
  }
  .filter-select {
    min-height: 2rem;
    padding: 0.3rem 1.9rem 0.3rem 0.55rem;
    border: 1px solid #0f3460;
    border-radius: 5px;
    background: #16213e;
    background-image: linear-gradient(45deg, transparent 50%, #a0a0b8 50%), linear-gradient(135deg, #a0a0b8 50%, transparent 50%);
    background-position: calc(100% - 14px) 50%, calc(100% - 9px) 50%;
    background-size: 5px 5px, 5px 5px;
    background-repeat: no-repeat;
    color: #e0e0e0;
    font: inherit;
    appearance: none;
  }
  .filter-select option { background: #16213e; color: #e0e0e0; }
  .filter-select:focus { outline: 2px solid #533483; outline-offset: 1px; }
  .filter-options {
    min-width: 0;
    margin: 0.75rem 0 0;
    padding: 0.55rem 0.65rem 0.65rem;
    border: 1px solid rgba(15, 52, 96, 0.85);
    border-radius: 6px;
  }
  .filter-options legend {
    padding: 0 0.25rem;
    color: #a0a0b8;
    font-size: 0.76rem;
    font-weight: 600;
  }
  .filter-option-list {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem 0.75rem;
  }
  .filter-option-list label {
    display: inline-flex;
    align-items: center;
    gap: 0.28rem;
    min-width: 0;
    color: #c6c6d8;
    font-size: 0.75rem;
  }
  .filter-option-list input { accent-color: #6c4fa3; }
  .filter-option-list code {
    max-width: min(31rem, 72vw);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .filter-empty { margin-bottom: 1.5rem; }
  .empty {
    padding: 2rem; text-align: center; color: #a0a0b8;
    background: #16213e; border-radius: 8px; border: 1px dashed #0f3460;
  }
  .empty-icon { margin-bottom: 0.5rem; color: #0f3460; }
  .hint { font-size: 0.85rem; margin-top: 0.5rem; opacity: 0.7; }
  .hint-list { list-style: none; padding: 0; margin: 0.5rem 0; font-size: 0.8rem; opacity: 0.7; }
  .hint-list li { margin: 0.25rem 0; }
  .plugin-list { display: flex; flex-direction: column; gap: 0.75rem; margin-bottom: 1.5rem; min-width: 0; }

  .missing-section { margin-bottom: 1.5rem; }
  .missing-section h3 { color: #e94560; font-size: 1rem; margin: 0 0 0.25rem; }
  .missing-hint { color: #a0a0b8; font-size: 0.8rem; margin: 0 0 0.75rem; }
  .missing-card { border-color: #e94560; opacity: 0.8; }
  .missing-text { color: #a0a0b8; font-size: 0.85rem; margin: 0.5rem 0 0; }
  .source-hint { display: block; margin-top: 0.25rem; font-size: 0.75rem; color: #666; }

  .registry-section {
    background: #16213e; border: 1px solid #0f3460;
    border-radius: 8px; padding: 0.75rem; margin-top: 1rem;
  }
  .registry-section summary { cursor: pointer; color: #a0a0b8; font-size: 0.9rem; font-weight: 600; }
  table { width: 100%; margin-top: 0.5rem; border-collapse: collapse; font-size: 0.85rem; }
  th { text-align: left; padding: 0.4rem 0.5rem; color: #a0a0b8; border-bottom: 1px solid #0f3460; }
  td { padding: 0.3rem 0.5rem; border-bottom: 1px solid #0f3460; }
  td code { color: #e0e0e0; }
  :global(.status-stable) { color: #4ecca3; }
  :global(.status-draft) { color: #ffc857; }
  :global(.status-deprecated) { color: #e94560; }
  .source-badge { font-size: 0.75rem; padding: 0.1rem 0.4rem; border-radius: 4px; font-weight: 600; }
  .source-core { background: #1a3a5c; color: #4ecca3; border: 1px solid #4ecca3; }
  .source-plugin { background: #0f3460; color: #a0a0b8; border: 1px solid #533483; }

  /* Modal */
  .modal-overlay {
    position: fixed; inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex; align-items: center; justify-content: center;
    z-index: 1000;
  }
  .modal {
    background: #16213e; border: 1px solid #0f3460; border-radius: 8px;
    width: min(880px, calc(100vw - 4rem)); max-width: calc(100vw - 4rem); height: min(680px, calc(100vh - 4rem)); max-height: calc(100vh - 4rem); display: flex; flex-direction: column;
  }
  .modal-header {
    display: flex; align-items: center; justify-content: space-between;
    padding: 1rem; border-bottom: 1px solid #0f3460;
  }
  .modal-header h3 { margin: 0; color: #e0e0f0; font-size: 1.1rem; }
  .modal-close { background: none; border: none; color: #a0a0b8; font-size: 1.2rem; cursor: pointer; padding: 0.2rem 0.5rem; }
  .modal-close:hover { color: #e94560; }
  .modal-body { padding: 1rem; overflow: auto; min-height: 0; flex: 1; display: flex; flex-direction: column; }
  .settings-modal-body { padding: 0; }
  .plugin-settings-surface {
    --verstak-plugin-surface: #16213e;
    --verstak-plugin-border: #0f3460;
    --verstak-plugin-text: #e0e0f0;
    --verstak-plugin-text-muted: #a0a0b8;
    --verstak-plugin-accent: #4ecca3;
    --verstak-plugin-danger: #e94560;
    --verstak-plugin-radius: 8px;
    --verstak-plugin-control-height: 2.25rem;
    width: 90%; min-height: 100%; margin: 0 auto; box-sizing: border-box; display: flex; flex-direction: column; padding: clamp(0.75rem, 1.5vw, 1.25rem);
  }
  .plugin-settings-surface :global(.plugin-bundle-host) { flex: 1; min-height: 0; display: flex; flex-direction: column; }
  .settings-hint { color: #666; font-size: 0.8rem; margin: 0.25rem 0; }
  .settings-hint code { color: #4ecca3; }

  @media (max-width: 760px) {
    .plugin-manager {
      width: 100%;
      padding-right: 0;
    }

    header {
      align-items: flex-start;
    }

    .reload-btn {
      width: 100%;
    }

    .scan-summary {
      grid-template-columns: 1fr;
    }

    .filter-header {
      align-items: stretch;
      flex-direction: column;
    }

    .filter-reset {
      width: 100%;
    }

    .filter-select-label {
      flex: 1 1 10rem;
    }

    .modal {
      width: min(880px, calc(100vw - 2rem));
      height: min(680px, calc(100vh - 2rem));
      max-height: calc(100vh - 2rem);
    }
  }
</style>
