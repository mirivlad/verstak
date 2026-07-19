<script>
  import { createEventDispatcher } from 'svelte';
  import Icon from '../ui/Icon.svelte';

  export let value = '';
  export let options = [];
  export let disabled = false;
  export let placeholder = '';
  export let labelKey = 'label';
  export let valueKey = 'value';
  export let id = '';

  const dispatch = createEventDispatcher();

  function handleChange(e) {
    value = e.target.value;
    dispatch('change', { value });
  }
</script>

<div class="vt-select-wrap" class:disabled>
  <select
    {id}
    class="vt-select"
    bind:value
    {disabled}
    on:change={handleChange}
    on:blur
  >
    {#if placeholder}
      <option value="" disabled>{placeholder}</option>
    {/if}
    {#each options as opt}
      {@const v = typeof opt === 'object' ? opt[valueKey] : opt}
      {@const l = typeof opt === 'object' ? opt[labelKey] : String(opt)}
      <option value={v}>{l}</option>
    {/each}
  </select>
  <span class="vt-select-arrow"><Icon name="chevron-down" size={14} /></span>
</div>

<style>
  .vt-select-wrap {
    position: relative;
    display: inline-flex;
    align-items: center;
    width: 100%;
  }
  .vt-select-wrap.disabled { opacity: 0.5; cursor: not-allowed; }
  .vt-select {
    appearance: none;
    -webkit-appearance: none;
    width: 100%;
    min-height: 2rem;
    box-sizing: border-box;
    border: 1px solid var(--vt-color-border-strong);
    border-radius: var(--vt-radius-sm);
    background: #0f1424;
    color: var(--vt-color-text-primary);
    padding: 0.35rem 1.7rem 0.35rem 0.5rem;
    font: inherit;
    font-size: 0.84rem;
    cursor: pointer;
  }
  .vt-select:disabled { cursor: not-allowed; }
  .vt-select:focus { outline: none; border-color: var(--vt-color-accent); box-shadow: var(--vt-focus-ring); }
  .vt-select option { background: #0f1424; color: var(--vt-color-text-primary); }
  .vt-select-arrow {
    position: absolute;
    right: 0.4rem;
    top: 50%;
    transform: translateY(-50%);
    pointer-events: none;
    color: var(--vt-color-text-muted);
    display: flex;
    align-items: center;
  }
</style>
