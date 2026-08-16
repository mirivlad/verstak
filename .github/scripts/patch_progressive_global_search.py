from pathlib import Path

path = Path('frontend/src/lib/shell/GlobalSearch.svelte')
text = path.read_text()

old = """  async function queryProviders(variants) {
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
"""
new = """  async function queryProviders(variants, onBatch) {
    const calls = [];
    (searchProviders || []).forEach((provider, providerRank) => {
      variants.forEach((variant) => {
        const call = (async () => {
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
        })();
        if (typeof onBatch === 'function') {
          call.then(onBatch);
        }
        calls.push(call);
      });
    });
    return Promise.all(calls);
  }
"""
if text.count(old) != 1:
    raise SystemExit(f'queryProviders anchor count: {text.count(old)}')
text = text.replace(old, new, 1)

old = """    searching = true;
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
"""
new = """    searching = true;
    const shellRows = shellIndex
      .map(item => ({ item, score: matchScore(item, variants) }))
      .filter(row => row.score > 0);
    const providerRows = [];
    let providerPartial = false;

    function publishResults() {
      if (seq !== searchSeq) return;
      const combined = dedupeRows(shellRows.concat(providerRows))
        .sort((a, b) => b.score - a.score || a.item.rank - b.item.rank || a.item.title.localeCompare(b.item.title));
      partial = providerPartial || combined.length > RESULT_LIMIT;
      results = combined.slice(0, RESULT_LIMIT).map(row => row.item);
      revision += 1;
    }

    // Providers and keyboard-layout variants are independent. Publish each
    // completed batch immediately so a fast filename/folder hit is never held
    // behind a slow full-text index or an irrelevant alternate-layout query.
    publishResults();
    await queryProviders(variants, (batch) => {
      if (seq !== searchSeq) return;
      providerRows.push(...batch.rows.map(item => ({ item, score: item.score })));
      providerPartial = providerPartial || batch.partial;
      publishResults();
    });
    if (seq !== searchSeq) return;
    publishResults();
    searching = false;
"""
if text.count(old) != 1:
    raise SystemExit(f'runSearch anchor count: {text.count(old)}')
path.write_text(text.replace(old, new, 1))

path = Path('frontend/tests/shell-source-contract-test.mjs')
text = path.read_text()
marker = """assertIncludes(
  globalSearch,
  'enabledPluginIds.has(provider?.pluginId)',
  'GlobalSearch should query only providers from enabled loaded/degraded plugins',
);
"""
addition = marker + """assertIncludes(
  globalSearch,
  'call.then(onBatch)',
  'GlobalSearch should publish completed provider batches without waiting for every provider/layout variant',
);
"""
if text.count(marker) != 1:
    raise SystemExit(f'progressive source assertion anchor count: {text.count(marker)}')
path.write_text(text.replace(marker, addition, 1))
