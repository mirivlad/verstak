from pathlib import Path


def replace_once(text, old, new, label):
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label} anchor count: {count}")
    return text.replace(old, new, 1)


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
path.write_text(text)

path = Path("frontend/e2e/ux-followup.spec.js")
text = path.read_text()
old = '''    const folderResult = page.locator('[data-global-search-result-category="folders"][data-global-search-result-path="Project/Notes"]');\n'''
new = '''    const folderResult = page.locator('[data-global-search-result-category="folders"]').filter({ hasText: 'Project/Notes' });\n'''
text = replace_once(text, old, new, "folder result selector")
path.write_text(text)
