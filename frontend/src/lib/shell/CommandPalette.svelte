<script>
  import { onDestroy, tick } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import { executePluginCommand } from '../plugin-host/VerstakPluginAPI.js';

  let open = false;
  let query = '';
  let commands = [];
  let selectedIndex = 0;
  let inputEl = null;
  let statusMessage = '';
  let statusType = '';

  const inactiveStatuses = new Set(['disabled', 'failed', 'incompatible', 'missing-required-capability']);
  const shellCommands = [
    { id: 'verstak.shell.open-today', title: 'Open Today', pluginId: 'verstak.shell', pluginName: 'Verstak', priority: 10, shellAction: 'today' },
    { id: 'verstak.shell.open-files', title: 'Open Files', pluginId: 'verstak.shell', pluginName: 'Verstak', priority: 20, shellAction: 'files' },
    { id: 'verstak.shell.open-activity', title: 'Open Activity', pluginId: 'verstak.shell', pluginName: 'Verstak', priority: 30, shellAction: 'activity' },
    { id: 'verstak.shell.open-browser-inbox', title: 'Open Browser Inbox', pluginId: 'verstak.shell', pluginName: 'Verstak', priority: 40, shellAction: 'browser-inbox' },
    { id: 'verstak.shell.create-markdown', title: 'Create Markdown File', pluginId: 'verstak.shell', pluginName: 'Verstak', priority: 50, shellAction: 'create-markdown' },
    { id: 'verstak.shell.create-text', title: 'Create Text File', pluginId: 'verstak.shell', pluginName: 'Verstak', priority: 60, shellAction: 'create-text' },
    { id: 'verstak.shell.sync-now', title: 'Sync Now', pluginId: 'verstak.shell', pluginName: 'Verstak', priority: 70, shellAction: 'sync-now' },
    { id: 'verstak.shell.open-sync-settings', title: 'Open Sync Settings', pluginId: 'verstak.shell', pluginName: 'Verstak', priority: 80, shellAction: 'sync-settings' },
    { id: 'verstak.shell.open-plugin-manager', title: 'Open Plugin Manager', pluginId: 'verstak.shell', pluginName: 'Verstak', priority: 90, shellAction: 'plugin-manager' },
  ];

  $: normalizedQuery = query.trim().toLowerCase();
  $: filteredCommands = commands.filter((command) => {
    if (!normalizedQuery) return true;
    return [
      command.title,
      command.id,
      command.pluginId,
      command.pluginName,
    ].filter(Boolean).join(' ').toLowerCase().includes(normalizedQuery);
  });
  $: if (selectedIndex >= filteredCommands.length) {
    selectedIndex = Math.max(0, filteredCommands.length - 1);
  }

  async function loadCommands() {
    const [plugins, contributions] = await Promise.all([
      App.GetPlugins().catch(() => []),
      App.GetContributions().catch(() => ({})),
    ]);
    const pluginById = new Map((plugins || []).map((plugin) => [plugin.manifest?.id, plugin]));
    const pluginCommands = (contributions.commands || [])
      .filter((command) => {
        const plugin = pluginById.get(command.pluginId);
        if (!plugin) return false;
        return !inactiveStatuses.has(plugin.status);
      })
      .map((command) => {
        const plugin = pluginById.get(command.pluginId);
        return {
          ...command,
          pluginName: plugin?.manifest?.name || command.pluginId,
          priority: 1000,
        };
      });
    commands = [...shellCommands, ...pluginCommands].sort((a, b) => {
      const priority = (a.priority || 1000) - (b.priority || 1000);
      if (priority) return priority;
      const title = String(a.title || a.id).localeCompare(String(b.title || b.id));
      if (title) return title;
      return String(a.pluginId).localeCompare(String(b.pluginId));
    });
  }

  async function openPalette() {
    await loadCommands();
    query = '';
    selectedIndex = 0;
    open = true;
    await tick();
    inputEl?.focus();
  }

  function closePalette() {
    open = false;
    query = '';
    selectedIndex = 0;
  }

  function setStatus(type, message) {
    statusType = type;
    statusMessage = message;
    window.clearTimeout(setStatus.timer);
    setStatus.timer = window.setTimeout(() => {
      statusType = '';
      statusMessage = '';
    }, 4000);
  }

  function clickWorkspaceTool(label) {
    const tabs = Array.from(document.querySelectorAll('.workspace-tabs [role="tab"]'));
    const tab = tabs.find((node) => String(node.textContent || '').trim().toLowerCase() === label);
    if (tab) {
      tab.click();
      return true;
    }
    return false;
  }

  async function openWorkspaceTool(label) {
    if (clickWorkspaceTool(label)) return;
    const selectedWorkspace = document.querySelector('.wt-node.selected .wt-label');
    const firstWorkspace = document.querySelector('.wt-label');
    const workspaceButton = selectedWorkspace || firstWorkspace;
    if (workspaceButton) {
      workspaceButton.click();
      await tick();
      await new Promise((resolve) => requestAnimationFrame(resolve));
    }
    clickWorkspaceTool(label);
  }

  async function startFilesCreate(action) {
    await openWorkspaceTool('files');
    await tick();
    await new Promise((resolve) => requestAnimationFrame(resolve));
    const button = document.querySelector(`[data-files-action="${action}"]`);
    if (!button) {
      throw new Error(`Files action not available: ${action}`);
    }
    button.click();
  }

  async function runShellCommand(command) {
    if (command.shellAction === 'plugin-manager') {
      window.dispatchEvent(new CustomEvent('verstak:open-settings', { detail: {} }));
      return;
    }
    if (command.shellAction === 'sync-settings') {
      window.dispatchEvent(new CustomEvent('verstak:open-settings', {
        detail: { pluginId: 'verstak.sync', panelId: 'verstak.sync.settings' }
      }));
      return;
    }
    if (command.shellAction === 'sync-now') {
      const result = await App.PluginSyncNow('verstak.sync');
      if (typeof result === 'string' && result) {
        throw new Error(result);
      }
      if (Array.isArray(result) && result[1]) {
        throw new Error(result[1]);
      }
      return;
    }
    if (command.shellAction === 'create-markdown') {
      await startFilesCreate('new-markdown');
      return;
    }
    if (command.shellAction === 'create-text') {
      await startFilesCreate('new-text');
      return;
    }
    const actionToTab = {
      today: 'today',
      files: 'files',
      activity: 'activity',
      'browser-inbox': 'browser inbox',
    };
    await openWorkspaceTool(actionToTab[command.shellAction] || command.shellAction);
  }

  async function runCommand(command) {
    if (!command) return;
    try {
      if (command.shellAction) {
        await runShellCommand(command);
        closePalette();
        setStatus('success', `${command.title || command.id} handled`);
        return;
      }
      const result = await executePluginCommand(command.pluginId, command.id, {
        source: 'command-palette',
      });
      closePalette();
      setStatus('success', `${command.title || command.id} ${result.status || 'handled'}`);
    } catch (err) {
      setStatus('error', `${command.title || command.id}: ${err?.message || String(err)}`);
    }
  }

  function moveSelection(delta) {
    if (filteredCommands.length === 0) return;
    selectedIndex = (selectedIndex + delta + filteredCommands.length) % filteredCommands.length;
  }

  function onWindowKeydown(event) {
    const key = event.key || '';
    const comboOpen = (event.ctrlKey || event.metaKey) && (key.toLowerCase() === 'k' || (event.shiftKey && key.toLowerCase() === 'p'));
    if (!open && comboOpen) {
      event.preventDefault();
      openPalette();
      return;
    }
    if (!open) return;

    if (key === 'Escape') {
      event.preventDefault();
      closePalette();
    } else if (key === 'ArrowDown') {
      event.preventDefault();
      moveSelection(1);
    } else if (key === 'ArrowUp') {
      event.preventDefault();
      moveSelection(-1);
    } else if (key === 'Enter') {
      event.preventDefault();
      runCommand(filteredCommands[selectedIndex]);
    }
  }

  function onOverlayMouseDown(event) {
    if (event.target === event.currentTarget) closePalette();
  }

  if (typeof window !== 'undefined') {
    window.addEventListener('keydown', onWindowKeydown);
  }

  onDestroy(() => {
    if (typeof window !== 'undefined') {
      window.removeEventListener('keydown', onWindowKeydown);
      window.clearTimeout(setStatus.timer);
    }
  });
</script>

{#if statusMessage}
  <div
    class="command-palette-toast"
    data-command-palette-status={statusType}
  >
    {statusMessage}
  </div>
{/if}

{#if open}
  <div class="command-palette-overlay" role="presentation" on:mousedown={onOverlayMouseDown}>
    <section class="command-palette" role="dialog" aria-modal="true" aria-label="Command Palette">
      <input
        bind:this={inputEl}
        bind:value={query}
        class="command-palette-input"
        data-command-palette-input
        placeholder="Run command"
        aria-label="Run command"
      />

      <div class="command-palette-list" role="listbox">
        {#if filteredCommands.length === 0}
          <div class="command-palette-empty">No commands</div>
        {:else}
          {#each filteredCommands as command, index}
            <button
              type="button"
              class:selected={index === selectedIndex}
              class="command-palette-item"
              data-command-id={command.id}
              on:mouseenter={() => selectedIndex = index}
              on:click={() => runCommand(command)}
            >
              <span class="command-palette-title">{command.title || command.id}</span>
              <span class="command-palette-meta">{command.pluginName} · {command.id}</span>
            </button>
          {/each}
        {/if}
      </div>
    </section>
  </div>
{/if}

<style>
  .command-palette-overlay {
    position: fixed;
    inset: 0;
    z-index: 10000;
    display: flex;
    align-items: flex-start;
    justify-content: center;
    padding: 10vh 1rem 1rem;
    background: rgba(0, 0, 0, 0.48);
  }

  .command-palette {
    width: min(640px, 100%);
    max-height: min(560px, 80vh);
    display: flex;
    flex-direction: column;
    overflow: hidden;
    border: 1px solid #254466;
    border-radius: 8px;
    background: #101526;
    box-shadow: 0 18px 48px rgba(0, 0, 0, 0.45);
  }

  .command-palette-input {
    width: 100%;
    min-height: 3rem;
    padding: 0.75rem 1rem;
    border: 0;
    border-bottom: 1px solid #213650;
    background: #0b1020;
    color: #f4f7fb;
    font: inherit;
    font-size: 1rem;
    outline: none;
  }

  .command-palette-list {
    min-height: 0;
    overflow: auto;
    padding: 0.35rem;
  }

  .command-palette-item {
    width: 100%;
    min-height: 3rem;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 0.18rem;
    padding: 0.55rem 0.75rem;
    border: 1px solid transparent;
    border-radius: 6px;
    background: transparent;
    color: #e0e0f0;
    text-align: left;
  }

  .command-palette-item:hover,
  .command-palette-item.selected {
    border-color: #4ecca3;
    background: #17243a;
  }

  .command-palette-title {
    font-size: 0.9rem;
    font-weight: 650;
  }

  .command-palette-meta {
    max-width: 100%;
    overflow: hidden;
    color: #8da2bd;
    font-size: 0.74rem;
    font-weight: 500;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .command-palette-empty {
    padding: 1.25rem;
    color: #8da2bd;
    font-size: 0.9rem;
    text-align: center;
  }

  .command-palette-toast {
    position: fixed;
    right: 1rem;
    bottom: 1rem;
    z-index: 10001;
    max-width: min(420px, calc(100vw - 2rem));
    padding: 0.7rem 0.9rem;
    border: 1px solid #254466;
    border-radius: 8px;
    background: #101526;
    color: #e0e0f0;
    font-size: 0.85rem;
    box-shadow: 0 12px 32px rgba(0, 0, 0, 0.35);
  }

  .command-palette-toast[data-command-palette-status="success"] {
    border-color: #4ecca3;
  }

  .command-palette-toast[data-command-palette-status="error"] {
    border-color: #e74c3c;
    color: #ffd6d1;
  }
</style>
