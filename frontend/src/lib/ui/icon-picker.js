import { mount, unmount } from 'svelte';
import IconPickerDialog from './IconPickerDialog.svelte';

export function openIconPicker(current = '') {
  const target = document.createElement('div');
  document.body.appendChild(target);
  let resolve;
  let closed = false;
  const result = new Promise((done) => { resolve = done; });
  function finish(value) {
    if (closed) return;
    closed = true;
    unmount(dialog);
    target.remove();
    resolve(value);
  }
  const dialog = mount(IconPickerDialog, { target, props: { current, finish } });
  return { result, cancel: () => finish(null) };
}
