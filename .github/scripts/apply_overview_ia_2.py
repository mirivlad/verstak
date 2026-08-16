from pathlib import Path


def replace_exact(path, old, new, count=1):
    p = Path(path)
    text = p.read_text()
    found = text.count(old)
    if found != count:
        raise SystemExit(f"{path}: expected {count} occurrences, found {found}: {old[:100]!r}")
    p.write_text(text.replace(old, new, count))


surface = "frontend/src/lib/shell/TodaySurface.svelte"

replace_exact(
    surface,
    "  $: attentionActionKind = needsAttention[0]?.actionKind || fallbackAttentionAction();\n",
    "",
)
replace_exact(
    surface,
    "    hasAttentionTools && needsAttention.length ? { key: 'attention', label: tr('overview.attention'), count: needsAttention.length, detail: countText('overview.count.pending', needsAttention.length), actionKind: attentionActionKind, actionLabel: tr('overview.reviewPending') } : null,\n",
    "",
)
replace_exact(
    surface,
    """  function fallbackAttentionAction() {
    if (hasBrowserInbox) return 'browser-inbox';
    if (hasTodos) return 'todo';
    if (hasActivity && hasJournal) return 'journal';
    return '';
  }

""",
    "",
)

summary_block = """  <div class="today-summary overview-summary" aria-label={tr('overview.summary')}>
    {#each summaryItems as item}
      <button
        type="button"
        class="today-summary-item overview-summary-item"
        class:summary-attention={item.key === 'attention'}
        data-overview-summary={item.key}
        data-overview-action={item.actionKind}
        aria-label={`${item.label}: ${item.actionLabel}`}
        on:click={() => openTool(item.actionKind)}
      >
        <strong>{loading ? '...' : item.count}</strong>
        <span>{item.label}</span>
        <small>{loading ? tr('common.loading') : item.detail}</small>
        <em>{item.actionLabel}</em>
      </button>
    {/each}
  </div>

"""
replace_exact(surface, summary_block, "")

replace_exact(
    surface,
    """      </section>

      <section class="today-panel overview-panel overview-recent" data-overview-section="recent">
""",
    """      </section>

      <div class="today-summary overview-summary" data-overview-section="summary" aria-label={tr('overview.summary')}>
        {#each summaryItems as item}
          <button
            type="button"
            class="today-summary-item overview-summary-item"
            data-overview-summary={item.key}
            data-overview-action={item.actionKind}
            aria-label={`${item.label}: ${item.actionLabel}`}
            on:click={() => openTool(item.actionKind)}
          >
            <strong>{loading ? '...' : item.count}</strong>
            <span>{item.label}</span>
            <small>{loading ? tr('common.loading') : item.detail}</small>
          </button>
        {/each}
      </div>

      <section class="today-panel overview-panel overview-recent" data-overview-section="recent">
""",
)

old_css = """  .today-summary {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr));
    gap: 0.5rem;
    padding: 0.75rem 0.75rem 0;
  }

  .today-summary-item {
    min-width: 0;
    display: grid;
    gap: 0.16rem;
    padding: 0.65rem 0.75rem;
    border: 1px solid var(--vt-color-border);
    border-radius: var(--vt-radius-lg);
    background: var(--vt-color-surface);
    color: inherit;
    cursor: pointer;
    font: inherit;
    text-align: left;
  }

  .today-summary-item:hover,
  .today-summary-item:focus-visible {
    border-color: var(--vt-color-accent);
    background: var(--vt-color-surface-hover);
    outline: none;
  }

  .today-summary-item strong {
    color: var(--vt-color-text-primary);
    font-size: 1rem;
    line-height: 1;
  }

  .today-summary-item span,
  .today-summary-item small,
  .today-summary-item em {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--vt-color-text-muted);
    font-size: 0.74rem;
  }

  .today-summary-item span {
    color: var(--vt-color-text-secondary);
    font-weight: 600;
  }

  .today-summary-item em {
    color: var(--vt-color-accent);
    font-size: 0.7rem;
    font-style: normal;
  }

  .today-summary-item.summary-attention {
    border-color: rgba(255, 200, 87, 0.5);
    background: var(--vt-color-warning-muted);
  }

  .today-summary-item.summary-attention em {
    color: var(--vt-color-warning);
  }
"""
new_css = """  .today-summary {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr));
    gap: 0.4rem;
  }

  .today-summary-item {
    min-width: 0;
    display: grid;
    grid-template-columns: auto minmax(0, 1fr);
    grid-template-rows: auto auto;
    align-items: center;
    column-gap: 0.55rem;
    row-gap: 0.12rem;
    padding: 0.5rem 0.65rem;
    border: 1px solid var(--vt-color-border);
    border-radius: var(--vt-radius-lg);
    background: var(--vt-color-surface);
    color: inherit;
    cursor: pointer;
    font: inherit;
    text-align: left;
  }

  .today-summary-item:hover,
  .today-summary-item:focus-visible {
    border-color: var(--vt-color-accent);
    background: var(--vt-color-surface-hover);
    outline: none;
  }

  .today-summary-item strong {
    grid-row: 1 / 3;
    color: var(--vt-color-text-primary);
    font-size: 1.05rem;
    line-height: 1;
  }

  .today-summary-item span,
  .today-summary-item small {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--vt-color-text-muted);
    font-size: 0.72rem;
  }

  .today-summary-item span {
    color: var(--vt-color-text-secondary);
    font-weight: 600;
  }
"""
replace_exact(surface, old_css, new_css)

replace_exact(
    surface,
    """  @container vt-content (max-width: 1120px) {
    .today-summary {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
  }

""",
    "",
)

replace_exact(
    "frontend/src/lib/i18n/catalogs/ru.js",
    "  'overview.continueHint': 'Вернитесь к следующему полезному делу в этом Деле.',",
    "  'overview.continueHint': 'Вернитесь к тому, на чём остановились.',",
)
replace_exact(
    "frontend/src/lib/i18n/catalogs/en.js",
    "  'overview.continueHint': 'Pick up the next useful item in this Deal.',",
    "  'overview.continueHint': 'Pick up where you left off.',",
)

visual = "frontend/e2e/visual/overview.visual.js"
replace_exact(
    visual,
    """    await expect(page.locator('[data-overview-section="continue"]')).toContainText('Пока неясно, с чего продолжить');
    await expect(page.locator('[data-overview-summary="attention"]')).toHaveCount(0);
""",
    """    await expect(page.locator('[data-overview-section="continue"]')).toContainText('Пока неясно, с чего продолжить');
    await expect(page.locator('[data-overview-section="continue"]')).toContainText('Вернитесь к тому, на чём остановились.');
    await expect(page.locator('[data-overview-summary="attention"]')).toHaveCount(0);
    await expect(page.locator('[data-overview-section="summary"]')).toBeVisible();
""",
)
replace_exact(
    visual,
    """    await expect(overview.locator('[data-overview-section="attention"]')).toContainText('Проверить резервную копию');
    await expect(overview.locator('[data-overview-section="attention"]')).toContainText('План выпуска');
    await expect(overview.locator('[data-overview-summary="attention"]')).toContainText('4');
    await shot(page, 'overview-populated-ru.png');
""",
    """    await expect(overview.locator('[data-overview-section="attention"]')).toContainText('Проверить резервную копию');
    await expect(overview.locator('[data-overview-section="attention"]')).toContainText('План выпуска');
    await expect(overview.locator('[data-overview-summary="attention"]')).toHaveCount(0);
    const continueBox = await overview.locator('[data-overview-section="continue"]').boundingBox();
    const attentionBox = await overview.locator('[data-overview-section="attention"]').boundingBox();
    const summaryBox = await overview.locator('[data-overview-section="summary"]').boundingBox();
    expect(Math.abs(continueBox.y - attentionBox.y)).toBeLessThan(4);
    expect(summaryBox.y).toBeGreaterThan(continueBox.y + continueBox.height - 1);
    await shot(page, 'overview-populated-ru.png');
""",
)

ux = "frontend/e2e/ux-today.spec.js"
replace_exact(
    ux,
    """    await expect(overview.locator('[data-overview-section="continue"]')).toContainText('Continue working');
    await expect(overview.locator('[data-overview-section="recent"]')).toContainText('Recent changes');
""",
    """    await expect(overview.locator('[data-overview-section="continue"]')).toContainText('Continue working');
    await expect(overview.locator('[data-overview-section="continue"]')).toContainText('Pick up where you left off.');
    await expect(overview.locator('[data-overview-section="summary"]')).toBeVisible();
    await expect(overview.locator('[data-overview-section="recent"]')).toContainText('Recent changes');
""",
)
replace_exact(
    ux,
    "    await expect(overview.locator('[data-overview-summary=\"attention\"]')).toContainText('3');\n",
    "    await expect(overview.locator('[data-overview-summary=\"attention\"]')).toHaveCount(0);\n",
)
