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
    {...$$restProps}
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
  .vt-select-wrap.disabled { opacity: 0.55; cursor: not-allowed; }
  .vt-select {
    appearance: none;
    -webkit-appearance: none;
    width: 100%;
    padding-right: 1.7rem;
    cursor: pointer;
  }
  .vt-select:disabled { cursor: not-allowed; }
  .vt-select option { background: var(--vt-color-input); color: var(--vt-color-text-primary); }
  .vt-select-arrow {
    position: absolute;
    right: var(--vt-space-2);
    top: 50%;
    transform: translateY(-50%);
    pointer-events: none;
    color: var(--vt-color-text-muted);
    display: flex;
    align-items: center;
  }
</style>
