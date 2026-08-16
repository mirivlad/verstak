from pathlib import Path

# Temporary test-only trace: record the exact props used to mount plugin UI.
path = Path('frontend/src/lib/plugin-host/PluginBundleHost.svelte')
text = path.read_text()
old = """          comp.mount(mountContainer, Object.assign({ componentId: compId }, componentProps || {}), api);
"""
new = """          window.__verstakPluginMountTrace = window.__verstakPluginMountTrace || [];
          window.__verstakPluginMountTrace.push({
            pluginId: pId,
            componentId: compId,
            props: JSON.parse(JSON.stringify(componentProps || {})),
          });
          comp.mount(mountContainer, Object.assign({ componentId: compId }, componentProps || {}), api);
"""
if text.count(old) != 1:
    raise SystemExit(f'PluginBundleHost mount anchor count: {text.count(old)}')
path.write_text(text.replace(old, new, 1))

path = Path('frontend/e2e/ux-followup.spec.js')
text = path.read_text()
old = """    await folderResult.click();
    await expect(page.locator('[data-workspace-current=\"Project\"]')).toBeVisible();
"""
new = """    await page.evaluate(() => {
      window.__verstakWorkspaceOpenTrace = [];
      window.addEventListener('verstak:workspace-open-tool', (event) => {
        window.__verstakWorkspaceOpenTrace.push(JSON.parse(JSON.stringify(event.detail || {})));
      });
    });
    await folderResult.click();
    await expect(page.locator('[data-workspace-current=\"Project\"]')).toBeVisible();
    await page.waitForTimeout(500);
    const navTrace = await page.evaluate(() => ({
      openTool: window.__verstakWorkspaceOpenTrace || [],
      mounts: window.__verstakPluginMountTrace || [],
    }));
    console.log('SEARCH_FOLDER_NAV_TRACE=' + JSON.stringify(navTrace));
"""
if text.count(old) != 1:
    raise SystemExit(f'folder click trace anchor count: {text.count(old)}')
path.write_text(text.replace(old, new, 1))
