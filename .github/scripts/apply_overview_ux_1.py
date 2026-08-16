from pathlib import Path
import re


def replace_exact(path, old, new, count=1):
    p = Path(path)
    text = p.read_text()
    found = text.count(old)
    if found != count:
        raise SystemExit(f"{path}: expected {count} occurrences, found {found}: {old[:80]!r}")
    p.write_text(text.replace(old, new, count))


surface = "frontend/src/lib/shell/TodaySurface.svelte"

replace_exact(
    surface,
    "  $: continueItems = filterAvailableItems(buildContinueItems(activityEvents, unprocessedCaptures, journalEntries, urgentTodos), overviewToolKey);",
    "  $: continueItems = filterAvailableItems(buildContinueItems(activityEvents, journalEntries), overviewToolKey);",
)

replace_exact(
    surface,
    "    hasAttentionTools ? { key: 'attention', label: tr('overview.attention'), count: needsAttention.length, detail: countText('overview.count.pending', needsAttention.length), actionKind: attentionActionKind, actionLabel: tr('overview.reviewPending') } : null,",
    "    hasAttentionTools && needsAttention.length ? { key: 'attention', label: tr('overview.attention'), count: needsAttention.length, detail: countText('overview.count.pending', needsAttention.length), actionKind: attentionActionKind, actionLabel: tr('overview.reviewPending') } : null,",
)

replace_exact(
    surface,
    "    unsubscribeLocale = i18n.subscribe((nextLocale) => locale = nextLocale);",
    "\n".join([
        "    unsubscribeLocale = i18n.subscribe((nextLocale) => {",
        "      const changed = locale !== nextLocale;",
        "      locale = nextLocale;",
        "      if (changed && workspaceRootPath) loadOverview();",
        "    });",
    ]),
)

replace_exact(
    surface,
    "    if (!value) return 'No timestamp';",
    "    if (!value) return tr('overview.time.none');",
)

p = Path(surface)
text = p.read_text()
pattern = re.compile(
    r"  function buildContinueItems\(events, captureRows, journalRows, todoRows\) \{.*?\n  \}\n\n  function buildNeedsAttention",
    re.S,
)
replacement = "\n".join([
    "  function buildContinueItems(events, journalRows) {",
    "    const noteCandidates = sortByTime(events)",
    "      .filter(item => isResumeEvent(item) && activityCategory(item) === 'notes')",
    "      .map(continueItemFromActivity);",
    "    const fileCandidates = sortByTime(events)",
    "      .filter(item => ['file.changed', 'file.created'].includes(String(item?.type || '').toLowerCase()))",
    "      .map(continueItemFromActivity);",
    "    const journalCandidates = sortByTime(journalRows).map(item => ({",
    "      id: item.entryId || `journal:${timeValue(item)}` ,",
    "      category: 'journal',",
    "      title: tr('overview.event.continueJournal', { title: journalTitle(item) }),",
    "      meta: itemTimeLabel(item),",
    "      time: timeValue(item),",
    "      absolute: absoluteTime(timeValue(item)),",
    "      actionKind: 'journal',",
    "      actionLabel: tr('overview.openJournal'),",
    "    }));",
    "    return [...noteCandidates, ...fileCandidates, ...journalCandidates].slice(0, 4);",
    "  }",
    "",
    "  function buildNeedsAttention",
])
text, changed = pattern.subn(replacement, text, count=1)
if changed != 1:
    raise SystemExit(f"{surface}: buildContinueItems replacement count={changed}")
p.write_text(text)

replace_exact(
    "frontend/src/lib/i18n/catalogs/ru.js",
    "  'overview.filter.captures': 'Сохранённое',",
    "  'overview.filter.captures': 'Браузер',",
)
replace_exact(
    "frontend/src/lib/i18n/catalogs/ru.js",
    "  'overview.captures': 'Сохранённое',",
    "  'overview.captures': 'Входящие',",
)
replace_exact(
    "frontend/src/lib/i18n/catalogs/ru.js",
    "  'overview.noResumeHint': 'Здесь появятся недавние заметки, файлы, сохранённое и записи журнала.',",
    "  'overview.noResumeHint': 'Здесь появятся недавние заметки, файлы и записи журнала.',",
)
replace_exact(
    "frontend/src/lib/i18n/catalogs/en.js",
    "  'overview.filter.captures': 'Captures',",
    "  'overview.filter.captures': 'Browser',",
)
replace_exact(
    "frontend/src/lib/i18n/catalogs/en.js",
    "  'overview.captures': 'Captures',",
    "  'overview.captures': 'Inbox',",
)
replace_exact(
    "frontend/src/lib/i18n/catalogs/en.js",
    "  'overview.noResumeHint': 'Recent notes, files, captures, and journal entries will appear here.',",
    "  'overview.noResumeHint': 'Recent notes, files, and journal entries will appear here.',",
)

visual = Path("frontend/e2e/visual/overview.visual.js")
text = visual.read_text()
old = "\n".join([
    "    await expect(page.locator('[data-overview-section=\"continue\"]')).toContainText('Пока неясно, с чего продолжить');",
    "    await shot(page, 'overview-empty-ru.png');",
])
new = "\n".join([
    "    await expect(page.locator('[data-overview-section=\"continue\"]')).toContainText('Пока неясно, с чего продолжить');",
    "    await expect(page.locator('[data-overview-summary=\"attention\"]')).toHaveCount(0);",
    "    await expect(page.locator('[data-overview-section=\"key-resources\"]')).toContainText('Открыть заметки');",
    "    await expect(page.locator('[data-overview-section=\"key-resources\"]')).not.toContainText('Open Notes');",
    "    await shot(page, 'overview-empty-ru.png');",
])
if text.count(old) != 1:
    raise SystemExit("visual empty-state anchor did not match exactly")
text = text.replace(old, new, 1)

old = "\n".join([
    "    await expect(overview).toContainText('План миграции');",
    "    await expect(overview).toContainText('Проверить резервную копию');",
    "    await expect(overview).toContainText('План выпуска');",
    "    await shot(page, 'overview-populated-ru.png');",
])
new = "\n".join([
    "    await expect(overview).toContainText('План миграции');",
    "    await expect(overview).toContainText('Проверить резервную копию');",
    "    await expect(overview).toContainText('План выпуска');",
    "    await expect(overview.locator('[data-overview-section=\"continue\"]')).not.toContainText('Проверить резервную копию');",
    "    await expect(overview.locator('[data-overview-section=\"continue\"]')).not.toContainText('План выпуска');",
    "    await expect(overview.locator('[data-overview-section=\"attention\"]')).toContainText('Проверить резервную копию');",
    "    await expect(overview.locator('[data-overview-section=\"attention\"]')).toContainText('План выпуска');",
    "    await expect(overview.locator('[data-overview-summary=\"attention\"]')).toContainText('4');",
    "    await shot(page, 'overview-populated-ru.png');",
])
if text.count(old) != 1:
    raise SystemExit("visual populated-state anchor did not match exactly")
visual.write_text(text.replace(old, new, 1))
