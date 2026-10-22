// Keyed views can replace the entire host. Keep asynchronous teardown barriers
// here so a new host cannot read state before the previous component flushed it.
const pendingCleanups = new Map();

export function unmountPluginComponent(pluginId, component, container, api) {
  const dispose = () => {
    try { api?.dispose?.(); }
    catch (error) { console.error('[PluginBundleHost] API dispose error:', error); }
  };
  let result;
  try { result = component?.unmount?.(container); }
  catch (error) { console.error('[PluginBundleHost] unmount error:', error); }
  if (!result || typeof result.then !== 'function') {
    dispose();
    return;
  }

  const teardown = Promise.resolve(result)
    .catch(error => console.error('[PluginBundleHost] unmount error:', error))
    .then(dispose);
  const pending = Promise.all([pendingCleanups.get(pluginId), teardown]);
  pendingCleanups.set(pluginId, pending);
  pending.then(() => {
    if (pendingCleanups.get(pluginId) === pending) pendingCleanups.delete(pluginId);
  });
}

export async function waitForPluginCleanup(pluginId) {
  while (pendingCleanups.has(pluginId)) await pendingCleanups.get(pluginId);
}
