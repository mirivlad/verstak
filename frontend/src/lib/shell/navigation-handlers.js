// Back and forward mean different things depending on what is on screen. In a
// file browser they walk the folder history; in the shell they walk the view
// history. The shell has to ask the thing in front of the user first.
//
// It used to do that by reaching into the DOM for a specific plugin's toolbar
// button — `document.querySelector('[data-files-action="up"]')`. That made the
// core depend on one plugin's markup: rename the attribute and shell navigation
// breaks silently, and no other plugin could ever participate.
//
// A plugin now registers what it can do with the navigation, and the shell asks
// the registry.

const handlers = new Map();

/**
 * Register a plugin's navigation handler.
 *
 * The handler may implement `canGoBack`/`goBack` and `canGoForward`/`goForward`.
 * A missing or falsy `can*` means the shell handles that direction itself.
 * Returns a function that unregisters it.
 */
export function registerNavigationHandler(pluginId, handler) {
	if (!pluginId || !handler || typeof handler !== 'object') {
		throw new Error('a navigation handler needs a plugin id and a handler object');
	}
	handlers.set(pluginId, handler);
	return function unregisterNavigationHandler() {
		if (handlers.get(pluginId) === handler) handlers.delete(pluginId);
	};
}

/**
 * Offer a navigation request to the plugin currently on screen.
 *
 * Returns true when the plugin took it, in which case the shell must not also
 * move its own history — otherwise one gesture would navigate twice.
 */
export function offerNavigation(pluginId, direction) {
	const handler = handlers.get(pluginId);
	if (!handler) return false;
	const canDo = direction === 'back' ? handler.canGoBack : handler.canGoForward;
	const doIt = direction === 'back' ? handler.goBack : handler.goForward;
	if (typeof canDo !== 'function' || typeof doIt !== 'function') return false;
	try {
		if (!canDo()) return false;
		doIt();
		return true;
	} catch (error) {
		// A misbehaving plugin must not take navigation down with it: fall
		// through and let the shell handle the request.
		console.error('[navigation] handler for ' + pluginId + ' failed:', error);
		return false;
	}
}

// Test seam. Nothing in the shell should need this.
export function clearNavigationHandlers() {
	handlers.clear();
}
