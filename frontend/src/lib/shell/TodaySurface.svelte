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
  $: summaryItems = [
    { key: 'captures', label: 'Captured', count: captures.length },
    { key: 'activity', label: 'Activity', count: activity.length },
    { key: 'worklog', label: 'Worklog', count: worklogSuggestions.length },
  ];
  $: resumeItem = nextResumeItem(captures, worklogSuggestions, activity, hasActivity);

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
    return item.title || item.summary || humanActivityType(item.type) || 'Activity event';
  }

  function humanActivityType(type) {
    const labels = {
      'workspace.selected': 'Workspace selected',
      'case.selected': 'Workspace selected',
      'file.opened': 'File opened',
      'file.changed': 'File changed',
      'note.saved': 'Note edited',
      'browser.capture.received': 'Browser capture received',
      'browser.capture.page': 'Page captured',
      'browser.capture.selection': 'Selection captured',
      'browser.capture.link': 'Link captured',
      'browser.capture.file': 'File captured',
      'browser.capture.converted': 'Capture converted',
      'action.started': 'Work session detected'
    };
    return labels[String(type || '').toLowerCase()] || '';
  }

  function worklogTitle(item) {
    return item.title || item.summary || item.date || item.entryId || 'Worklog item';
  }

  function nextResumeItem(captureRows, worklogRows, activityRows, activityAvailable) {
    if (captureRows.length > 0) {
      const capture = captureRows[0];
      return {
        title: captureTitle(capture),
        meta: capture.url || capture.domain || capture.kind || 'Browser capture',
        kind: 'browser-inbox',
        action: 'Open Inbox',
      };
    }
    if (worklogRows.length > 0) {
      const item = worklogRows[0];
      return {
        title: worklogTitle(item),
        meta: item.minutes ? item.minutes + ' min suggested' : item.date || 'Suggested worklog',
        kind: 'activity',
        action: activityAvailable ? 'Review Activity' : 'Refresh',
      };
    }
    if (activityRows.length > 0) {
      const item = activityRows[0];
      return {
        title: activityTitle(item),
        meta: item.occurredAt || item.receivedAt || item.type || 'Activity event',
        kind: 'activity',
        action: 'Open Activity',
      };
    }
    return null;
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

  <div class="today-summary" aria-label="Today summary">
    {#each summaryItems as item}
      <div class="today-summary-item" data-today-summary={item.key}>
        <strong>{loading ? '...' : item.count}</strong>
        <span>{item.label}</span>
      </div>
    {/each}
  </div>

  <section class="today-resume" data-today-section="resume">
    <div class="today-resume-copy">
      <span>Resume next</span>
      {#if loading}
        <strong>Loading workspace signals...</strong>
        <p>Recent captures, activity, and worklog suggestions will appear here.</p>
      {:else if resumeItem}
        <strong>{resumeItem.title}</strong>
        <p>{resumeItem.meta}</p>
      {:else}
        <strong>No pending workspace signals</strong>
        <p>Start with files, capture something from the browser, or review plugin activity.</p>
      {/if}
    </div>
    {#if resumeItem}
      <button type="button" data-today-action="resume-primary" on:click={() => openTool(resumeItem.kind)}>{resumeItem.action}</button>
    {:else if hasFiles}
      <button type="button" data-today-action="resume-primary" on:click={() => openTool('files')}>Open Files</button>
    {/if}
  </section>

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
    background: var(--vt-color-background);
    color: var(--vt-color-text-primary);
    overflow: auto;
  }

  .today-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 1rem;
    border-bottom: 1px solid var(--vt-color-border);
    background: var(--vt-color-surface-muted);
  }

  .today-header h2 {
    margin: 0;
    font-size: 1.05rem;
  }

  .today-header p {
    margin: 0.25rem 0 0;
    color: var(--vt-color-text-muted);
    font-size: 0.8rem;
  }

  .today-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.75rem;
    padding: 0.75rem;
  }

  .today-summary {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 0.5rem;
    padding: 0.75rem 0.75rem 0;
  }

  .today-summary-item {
    min-width: 0;
    display: grid;
    gap: 0.15rem;
    padding: 0.65rem 0.75rem;
    border: 1px solid var(--vt-color-border);
    border-radius: var(--vt-radius-lg);
    background: var(--vt-color-surface);
  }

  .today-summary-item strong {
    color: var(--vt-color-text-primary);
    font-size: 1rem;
    line-height: 1;
  }

  .today-summary-item span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--vt-color-text-muted);
    font-size: 0.74rem;
  }

  .today-resume {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    margin: 0.75rem 0.75rem 0;
    padding: 0.85rem 1rem;
    border: 1px solid rgba(78, 204, 163, 0.24);
    border-radius: var(--vt-radius-lg);
    background: linear-gradient(135deg, rgba(78, 204, 163, 0.11), rgba(27, 36, 64, 0.6));
  }

  .today-resume-copy {
    min-width: 0;
    display: grid;
    gap: 0.22rem;
  }

  .today-resume-copy span {
    color: var(--vt-color-accent);
    font-size: 0.72rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .today-resume-copy strong {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--vt-color-text-primary);
    font-size: 0.98rem;
  }

  .today-resume-copy p {
    min-width: 0;
    margin: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--vt-color-text-secondary);
    font-size: 0.8rem;
  }

  .today-panel {
    min-width: 0;
    min-height: 10rem;
    display: flex;
    flex-direction: column;
    border: 1px solid var(--vt-color-border);
    border-radius: var(--vt-radius-lg);
    background: var(--vt-color-surface);
  }

  .today-panel-head {
    min-height: 2.8rem;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    padding: 0.65rem 0.75rem;
    border-bottom: 1px solid var(--vt-color-border);
  }

  .today-panel h3 {
    margin: 0;
    color: var(--vt-color-text-primary);
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
    color: var(--vt-color-text-muted);
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
    border: 1px solid var(--vt-color-border);
    border-radius: var(--vt-radius-md);
    background: var(--vt-color-surface-muted);
  }

  .today-row strong {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--vt-color-text-primary);
    font-size: 0.85rem;
  }

  .today-row span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--vt-color-text-muted);
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

    .today-summary {
      grid-template-columns: 1fr;
    }

    .today-resume {
      align-items: stretch;
      flex-direction: column;
    }

    .today-resume button {
      width: 100%;
    }
  }
</style>
