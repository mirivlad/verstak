<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';

  export let workspaceRootPath = '';
  export let workspaceTitle = '';
  export let availableTools = [];

  const dispatch = createEventDispatcher();

  let loading = true;
  let captures = [];
  let activity = [];
  let worklogSuggestions = [];

  $: hasBrowserInbox = hasTool('browser inbox') || hasTool('inbox');
  $: hasActivity = hasTool('activity');
  $: hasFiles = hasTool('files');

  onMount(() => {
    loadToday();
  });

  function hasTool(name) {
    name = String(name || '').toLowerCase();
    return (availableTools || []).some(tool => {
      const label = String(tool?.title || tool?.id || tool?.pluginId || '').toLowerCase();
      return label.includes(name);
    });
  }

  function decodeTuple(response, fallback) {
    if (Array.isArray(response) && response.length === 2) return response[1] ? fallback : (response[0] || fallback);
    return response || fallback;
  }

  async function readPluginSettings(pluginId) {
    try {
      return decodeTuple(await App.ReadPluginSettings(pluginId), {});
    } catch (_) {
      return {};
    }
  }

  function workspaceKey(prefix) {
    return prefix + encodeURIComponent(String(workspaceRootPath || '').trim());
  }

  function normalizeRows(value) {
    return Array.isArray(value) ? value.filter(item => item && typeof item === 'object') : [];
  }

  function rowsFor(settings, keys) {
    return keys.flatMap(key => normalizeRows(settings?.[key]));
  }

  function captureTitle(capture) {
    return capture.title || capture.fileName || capture.url || capture.captureId || 'Untitled capture';
  }

  function activityTitle(item) {
    return item.title || item.summary || item.type || item.activityId || 'Activity event';
  }

  function worklogTitle(item) {
    return item.title || item.summary || item.date || item.entryId || 'Worklog item';
  }

  async function loadToday() {
    loading = true;
    const [browserSettings, activitySettings, journalSettings] = await Promise.all([
      readPluginSettings('verstak.browser-inbox'),
      readPluginSettings('verstak.activity'),
      readPluginSettings('verstak.journal'),
    ]);

    captures = rowsFor(browserSettings, [
      workspaceKey('captures:workspace:'),
      'captures:global',
      'captures',
    ]).slice(0, 4);

    activity = rowsFor(activitySettings, [
      workspaceKey('events:workspace:'),
      'events:global',
      'events',
    ]).slice(0, 4);

    worklogSuggestions = rowsFor(journalSettings, [
      workspaceKey('suggestions:workspace:'),
      workspaceKey('worklog:workspace:'),
      'suggestions',
      'worklog',
    ]).slice(0, 4);

    loading = false;
  }

  function openTool(kind) {
    dispatch('openTool', { kind });
  }
</script>

<div class="today-root" aria-label="Today">
  <div class="today-header">
    <div>
      <h2>Today</h2>
      <p>{workspaceTitle || workspaceRootPath || 'Workspace'} overview</p>
    </div>
    <button type="button" on:click={loadToday}>Refresh</button>
  </div>

  <div class="today-grid">
    <section class="today-panel" data-today-section="captured">
      <div class="today-panel-head">
        <h3>Captured</h3>
        <button type="button" data-today-action="browser-inbox" on:click={() => openTool('browser-inbox')}>Open Inbox</button>
      </div>
      {#if loading}
        <p class="today-empty">Loading captures...</p>
      {:else if captures.length}
        <div class="today-list">
          {#each captures as capture}
            <div class="today-row">
              <strong>{captureTitle(capture)}</strong>
              <span>{capture.url || capture.domain || capture.kind || 'Browser capture'}</span>
            </div>
          {/each}
        </div>
      {:else}
        <p class="today-empty">No browser captures yet. Send a page, selection, link, or file from the browser extension.</p>
      {/if}
    </section>

    <section class="today-panel" data-today-section="activity">
      <div class="today-panel-head">
        <h3>Recent Activity</h3>
        {#if hasActivity}
          <button type="button" data-today-action="activity" on:click={() => openTool('activity')}>Open Activity</button>
        {/if}
      </div>
      {#if loading}
        <p class="today-empty">Loading activity...</p>
      {:else if activity.length}
        <div class="today-list">
          {#each activity as item}
            <div class="today-row">
              <strong>{activityTitle(item)}</strong>
              <span>{item.occurredAt || item.receivedAt || item.type || 'Activity event'}</span>
            </div>
          {/each}
        </div>
      {:else}
        <p class="today-empty">No activity events yet. File changes, captures, and conversions will appear here.</p>
      {/if}
    </section>

    <section class="today-panel" data-today-section="worklog">
      <div class="today-panel-head">
        <h3>Worklog Suggestions</h3>
      </div>
      {#if loading}
        <p class="today-empty">Loading worklog suggestions...</p>
      {:else if worklogSuggestions.length}
        <div class="today-list">
          {#each worklogSuggestions as item}
            <div class="today-row">
              <strong>{worklogTitle(item)}</strong>
              <span>{item.minutes ? item.minutes + ' min' : item.date || 'Suggested worklog'}</span>
            </div>
          {/each}
        </div>
      {:else}
        <p class="today-empty">No worklog suggestions yet. Activity will be grouped into suggestions once work starts.</p>
      {/if}
    </section>

    <section class="today-panel" data-today-section="quick-actions">
      <div class="today-panel-head">
        <h3>Quick Actions</h3>
      </div>
      <div class="today-actions">
        {#if hasFiles}
          <button type="button" data-today-action="files" on:click={() => openTool('files')}>Open Files</button>
        {/if}
        <button type="button" on:click={() => openTool('browser-inbox')}>Process Captures</button>
        {#if hasActivity}
          <button type="button" on:click={() => openTool('activity')}>Review Activity</button>
        {/if}
      </div>
    </section>
  </div>
</div>

<style>
  .today-root {
    height: 100%;
    min-height: 0;
    display: flex;
    flex-direction: column;
    background: #101020;
    color: #e0e0f0;
    overflow: auto;
  }

  .today-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 1rem;
    border-bottom: 1px solid #16213e;
    background: #12122a;
  }

  .today-header h2 {
    margin: 0;
    font-size: 1.05rem;
  }

  .today-header p {
    margin: 0.25rem 0 0;
    color: #8b8ba8;
    font-size: 0.8rem;
  }

  .today-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.75rem;
    padding: 0.75rem;
  }

  .today-panel {
    min-width: 0;
    min-height: 10rem;
    display: flex;
    flex-direction: column;
    border: 1px solid #16213e;
    border-radius: 8px;
    background: #15152c;
  }

  .today-panel-head {
    min-height: 2.8rem;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    padding: 0.65rem 0.75rem;
    border-bottom: 1px solid rgba(22, 33, 62, 0.8);
  }

  .today-panel h3 {
    margin: 0;
    color: #f0f0ff;
    font-size: 0.9rem;
  }

  .today-panel-head button,
  .today-actions button,
  .today-header button {
    min-height: 1.85rem;
    padding: 0.3rem 0.65rem;
    font-size: 0.76rem;
  }

  .today-empty {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    margin: 0;
    padding: 1rem;
    color: #8b8ba8;
    font-size: 0.82rem;
    line-height: 1.45;
    text-align: center;
  }

  .today-list {
    display: grid;
    gap: 0.45rem;
    padding: 0.65rem;
  }

  .today-row {
    min-width: 0;
    display: grid;
    gap: 0.2rem;
    padding: 0.55rem;
    border: 1px solid rgba(78, 204, 163, 0.16);
    border-radius: 6px;
    background: #101827;
  }

  .today-row strong {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: #f4f7fb;
    font-size: 0.85rem;
  }

  .today-row span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: #8b8ba8;
    font-size: 0.75rem;
  }

  .today-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    padding: 0.75rem;
  }

  @media (max-width: 860px) {
    .today-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
