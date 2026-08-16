from pathlib import Path


def replace_once(text, old, new, label):
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label} anchor count: {count}")
    return text.replace(old, new, 1)


# GlobalSearch publishes fast provider batches immediately instead of waiting
# for every keyboard-layout variant and carries the target Deal with deep links.
path = Path("frontend/src/lib/shell/GlobalSearch.svelte")
text = path.read_text()
old = r'''  async function queryProviders(variants) {
    const calls = [];
    (searchProviders || []).forEach((provider, providerRank) => {
      variants.forEach((variant) => {
        calls.push((async () => {
          try {
            const response = await executePluginCommand(provider.pluginId, provider.handler, {
              query: variant,
              limit: RESULT_LIMIT,
            });
            const value = commandResult(response);
            const list = Array.isArray(value) ? value : (Array.isArray(value?.results) ? value.results : []);
            return {
              rows: list.map(item => normalizeProviderItem(provider, item, providerRank)).filter(Boolean),
              partial: Boolean(value?.partial),
            };
          } catch (error) {
            console.warn(`[GlobalSearch] provider ${provider.pluginId}/${provider.id || provider.handler} failed:`, error);
            return { rows: [], partial: true };
          }
        })());
      });
    });
    return Promise.all(calls);
  }
'''
new = r'''  function queryProviderCalls(variants) {
    const calls = [];
    (searchProviders || []).forEach((provider, providerRank) => {
      variants.forEach((variant) => {
        calls.push((async () => {
          try {
            const response = await executePluginCommand(provider.pluginId, provider.handler, {
              query: variant,
              limit: RESULT_LIMIT,
            });
            const value = commandResult(response);
            const list = Array.isArray(value) ? value : (Array.isArray(value?.results) ? value.results : []);
            return {
              rows: list.map(item => normalizeProviderItem(provider, item, providerRank)).filter(Boolean),
              partial: Boolean(value?.partial),
            };
          } catch (error) {
            console.warn(`[GlobalSearch] provider ${provider.pluginId}/${provider.id || provider.handler} failed:`, error);
            return { rows: [], partial: true };
          }
        })());
      });
    });
    return calls;
  }
'''
text = replace_once(text, old, new, "queryProviders")

old = r'''  async function runSearch(value) {
    const variants = queryVariants(value);
    const seq = ++searchSeq;
    if (!variants.length) {
      searching = false;
      partial = false;
      results = [];
      return;
    }

    searching = true;
    const shellRows = shellIndex
      .map(item => ({ item, score: matchScore(item, variants) }))
      .filter(row => row.score > 0);
    const providerBatches = await queryProviders(variants);
    if (seq !== searchSeq) return;

    const providerRows = providerBatches.flatMap(batch => batch.rows.map(item => ({ item, score: item.score })));
    const combined = dedupeRows(shellRows.concat(providerRows))
      .sort((a, b) => b.score - a.score || a.item.rank - b.item.rank || a.item.title.localeCompare(b.item.title));
    partial = providerBatches.some(batch => batch.partial) || combined.length > RESULT_LIMIT;
    results = combined.slice(0, RESULT_LIMIT).map(row => row.item);
    searching = false;
    revision += 1;
  }
'''
new = r'''  function publishSearchResults(seq, shellRows, providerRows, providerPartial, pending) {
    if (seq !== searchSeq) return;
    const combined = dedupeRows(shellRows.concat(providerRows))
      .sort((a, b) => b.score - a.score || a.item.rank - b.item.rank || a.item.title.localeCompare(b.item.title));
    partial = providerPartial || pending > 0 || combined.length > RESULT_LIMIT;
    results = combined.slice(0, RESULT_LIMIT).map(row => row.item);
    searching = pending > 0;
    revision += 1;
  }

  function runSearch(value) {
    const variants = queryVariants(value);
    const seq = ++searchSeq;
    if (!variants.length) {
      searching = false;
      partial = false;
      results = [];
      return;
    }

    const shellRows = shellIndex
      .map(item => ({ item, score: matchScore(item, variants) }))
      .filter(row => row.score > 0);
    const providerRows = [];
    let providerPartial = false;
    const calls = queryProviderCalls(variants);
    let pending = calls.length;

    publishSearchResults(seq, shellRows, providerRows, providerPartial, pending);
    calls.forEach((call) => {
      call.then((batch) => {
        if (seq !== searchSeq) return;
        pending -= 1;
        providerPartial = providerPartial || batch.partial;
        providerRows.push(...batch.rows.map(item => ({ item, score: item.score })));
        publishSearchResults(seq, shellRows, providerRows, providerPartial, pending);
      });
    });
  }
'''
text = replace_once(text, old, new, "runSearch")
old = r'''      window.dispatchEvent(new CustomEvent('verstak:workspace-open-tool', {
        detail: { workspaceItemId: action.workspaceItemId, toolRequest: action.toolRequest || null }
      }));
'''
new = r'''      window.dispatchEvent(new CustomEvent('verstak:workspace-open-tool', {
        detail: { workspaceRootPath, workspaceItemId: action.workspaceItemId, toolRequest: action.toolRequest || null }
      }));
'''
text = replace_once(text, old, new, "workspace deep-link event")
path.write_text(text)

# WorkspaceHost queues a cross-Deal tool request until its target workspace is
# the active one. Changing Deal must clear stale active requests, but not the
# pending request whose target is exactly the new Deal.
path = Path("frontend/src/lib/shell/WorkspaceHost.svelte")
text = path.read_text()
old = r'''  let requestedWorkspaceItemId = '';
  let requestedToolRequest = null;
  let activeToolRequest = null;
  let requestedWorkspaceRoot = '';
'''
new = r'''  let requestedWorkspaceItemId = '';
  let requestedToolRequest = null;
  let requestedTargetWorkspaceRoot = '';
  let activeToolRequest = null;
  let requestedWorkspaceRoot = '';
'''
text = replace_once(text, old, new, "pending target state")
old = r'''  $: if (workspaceRootPath !== requestedWorkspaceRoot) {
    requestedWorkspaceRoot = workspaceRootPath;
    requestedToolRequest = null;
    activeToolRequest = null;
  }
'''
new = r'''  $: if (workspaceRootPath !== requestedWorkspaceRoot) {
    requestedWorkspaceRoot = workspaceRootPath;
    activeToolRequest = null;
    if (requestedTargetWorkspaceRoot && requestedTargetWorkspaceRoot !== workspaceRootPath) {
      requestedWorkspaceItemId = '';
      requestedToolRequest = null;
      requestedTargetWorkspaceRoot = '';
    } else if (!requestedTargetWorkspaceRoot) {
      requestedToolRequest = null;
    }
  }
'''
text = replace_once(text, old, new, "workspace root request reset")
old = r'''  $: if (requestedWorkspaceItemId && workspaceTools.length > 0) {
    const match = findWorkspaceItem(requestedWorkspaceItemId);
    if (match) {
      const toolRequest = requestedToolRequest;
      requestedWorkspaceItemId = '';
      requestedToolRequest = null;
      selectTool(match, toolRequest);
    }
  }
'''
new = r'''  $: if (requestedWorkspaceItemId && workspaceTools.length > 0 && (!requestedTargetWorkspaceRoot || requestedTargetWorkspaceRoot === workspaceRootPath)) {
    const match = findWorkspaceItem(requestedWorkspaceItemId);
    if (match) {
      const toolRequest = requestedToolRequest;
      requestedWorkspaceItemId = '';
      requestedToolRequest = null;
      requestedTargetWorkspaceRoot = '';
      selectTool(match, toolRequest);
    }
  }
'''
text = replace_once(text, old, new, "pending item resolution")
old = r'''  function requestWorkspaceItem(workspaceItemId, toolRequest = null) {
    requestedWorkspaceItemId = String(workspaceItemId || '').trim();
    requestedToolRequest = toolRequest;
    const match = findWorkspaceItem(requestedWorkspaceItemId);
    if (match) {
      requestedWorkspaceItemId = '';
      requestedToolRequest = null;
      selectTool(match, toolRequest);
    }
  }

  function openWorkspaceTool(event) {
    requestWorkspaceItem(event?.detail?.workspaceItemId, event?.detail?.toolRequest || null);
  }

  function handleWorkspaceOpenTool(event) {
    requestWorkspaceItem(event?.detail?.workspaceItemId, event?.detail?.toolRequest || null);
  }
'''
new = r'''  function requestWorkspaceItem(workspaceItemId, toolRequest = null, targetWorkspaceRoot = '') {
    requestedWorkspaceItemId = String(workspaceItemId || '').trim();
    requestedToolRequest = toolRequest;
    requestedTargetWorkspaceRoot = String(targetWorkspaceRoot || '').trim();
    if (requestedTargetWorkspaceRoot && requestedTargetWorkspaceRoot !== workspaceRootPath) return;
    const match = findWorkspaceItem(requestedWorkspaceItemId);
    if (match) {
      requestedWorkspaceItemId = '';
      requestedToolRequest = null;
      requestedTargetWorkspaceRoot = '';
      selectTool(match, toolRequest);
    }
  }

  function openWorkspaceTool(event) {
    requestWorkspaceItem(event?.detail?.workspaceItemId, event?.detail?.toolRequest || null, event?.detail?.workspaceRootPath || '');
  }

  function handleWorkspaceOpenTool(event) {
    requestWorkspaceItem(event?.detail?.workspaceItemId, event?.detail?.toolRequest || null, event?.detail?.workspaceRootPath || '');
  }
'''
text = replace_once(text, old, new, "workspace item handoff")
path.write_text(text)

# The result contract exposes the folder path as visible subtitle text; the
# machine field remains the provider-owned category id.
path = Path("frontend/e2e/ux-followup.spec.js")
text = path.read_text()
old = '''    const folderResult = page.locator('[data-global-search-result-category="folders"][data-global-search-result-path="Project/Notes"]');\n'''
new = '''    const folderResult = page.locator('[data-global-search-result-category="folders"]').filter({ hasText: 'Project/Notes' });\n'''
text = replace_once(text, old, new, "folder result selector")
path.write_text(text)
