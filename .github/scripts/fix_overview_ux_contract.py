from pathlib import Path

path = Path('frontend/e2e/ux-today.spec.js')
text = path.read_text()


def replace_exact(old, new, count=1):
    global text
    found = text.count(old)
    if found != count:
        raise SystemExit(f'expected {count} occurrences, found {found}: {old[:100]!r}')
    text = text.replace(old, new, count)


replace_exact(
    "    await expect(summaryCards).toHaveCount(6);\n",
    "    await expect(summaryCards).toHaveCount(5);\n    await expect(overview.locator('[data-overview-summary=\"attention\"]')).toHaveCount(0);\n",
)

replace_exact(
    """    await page.getByRole('tab', { name: 'Overview' }).click();
    await overview.locator('[data-overview-summary="attention"]').click();
    await expect(page.getByRole('tab', { name: 'Browser' })).toHaveAttribute('aria-selected', 'true');
""",
    """    await page.getByRole('tab', { name: 'Overview' }).click();
    await expect(overview.locator('[data-overview-summary="attention"]')).toHaveCount(0);
""",
)

replace_exact(
    """    await expect(candidates).toHaveCount(4);
    await expect(candidates.nth(0)).toContainText('Quote to process');
    await expect(candidates.nth(1)).toContainText('Research Report');
    await expect(candidates.nth(2)).toContainText('Edited note "Overview"');
    await expect(candidates.nth(3)).toContainText('Changed file "draft.md"');
    await candidates.nth(0).click();

    await expect(page.getByRole('tab', { name: 'Browser' })).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('.browser-inbox-root')).toBeVisible({ timeout: 10000 });
""",
    """    await expect(candidates).toHaveCount(3);
    await expect(candidates.nth(0)).toContainText('Edited note "Overview"');
    await expect(candidates.nth(1)).toContainText('Changed file "draft.md"');
    await expect(candidates.nth(2)).toContainText('Continue journal entry "Write project summary"');
    await expect(resume).not.toContainText('Quote to process');
    await expect(resume).not.toContainText('Research Report');
    await candidates.nth(0).click();

    await expect(page.getByRole('tab', { name: 'Notes' })).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('.notes-root')).toBeVisible({ timeout: 10000 });
""",
)

path.write_text(text)
