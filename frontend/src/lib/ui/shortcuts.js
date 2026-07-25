// One place that decides whether a keyboard event is a given shortcut.
//
// `event.key` carries the character the current layout produces, so
// `event.key === 's'` is false on a Russian layout, where the S key yields
// 'ы'. `event.code` names the physical key and does not move with the layout,
// which is what a shortcut means.
//
// It also hides the Ctrl/Cmd split: Linux and Windows use Ctrl, macOS uses
// Cmd, and every call site otherwise repeats `(ctrlKey || metaKey)`.
//
// Spec fields:
//   code       required, a KeyboardEvent.code value such as 'KeyS', 'Enter'
//   ctrlOrMeta Ctrl on Linux/Windows, Cmd on macOS
//   ctrl       Ctrl specifically, when the platform difference matters
//   meta       Cmd/Super specifically
//   shift, alt exact match; omitted means the modifier must be absent

function wants(spec, name) {
  return spec[name] === true;
}

export function matchesShortcut(event, spec) {
  if (!event || !spec || !spec.code) return false;
  if (event.code !== spec.code) return false;

  const ctrl = Boolean(event.ctrlKey);
  const meta = Boolean(event.metaKey);
  const shift = Boolean(event.shiftKey);
  const alt = Boolean(event.altKey);

  if (wants(spec, 'ctrlOrMeta')) {
    if (!ctrl && !meta) return false;
  } else {
    if (wants(spec, 'ctrl') !== ctrl) return false;
    if (wants(spec, 'meta') !== meta) return false;
  }

  if (wants(spec, 'shift') !== shift) return false;
  if (wants(spec, 'alt') !== alt) return false;
  return true;
}

// Convenience for the common case, so call sites read as the shortcut itself.
export function isSaveShortcut(event) {
  return matchesShortcut(event, { code: 'KeyS', ctrlOrMeta: true });
}
