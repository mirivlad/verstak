from pathlib import Path

path = Path('frontend/e2e/ux-followup.spec.js')
text = path.read_text()
old = '''    const folderResult = page.locator('[data-global-search-result-category="folders"][data-global-search-result-path="Project/Notes"]');
'''
new = '''    const folderResult = page.locator('[data-global-search-result-category="folders"]').filter({ hasText: 'Project/Notes' });
'''
if text.count(old) != 1:
    raise SystemExit(f'folder result selector anchor count: {text.count(old)}')
path.write_text(text.replace(old, new, 1))
