<script>
  import PluginManager from './lib/plugin-manager/PluginManager.svelte';

  let view = 'plugin-manager';
</script>

<main>
  <nav>
    <h1>Verstak</h1>
    <div class="nav-items">
      <button
        class="nav-item"
        class:active={view === 'plugin-manager'}
        on:click={() => view = 'plugin-manager'}
        type="button"
      >
        ⚙ Plugin Manager
      </button>
    </div>
  </nav>

  <section class="content">
    {#if view === 'plugin-manager'}
      <PluginManager />
    {/if}
  </section>
</main>

<style>
  main {
    display: flex;
    height: 100vh;
    background: #1a1a2e;
  }

  nav {
    width: 220px;
    background: #16213e;
    border-right: 1px solid #0f3460;
    padding: 1rem;
    display: flex;
    flex-direction: column;
  }

  nav h1 {
    font-size: 1.2rem;
    color: #e94560;
    margin-bottom: 1.5rem;
  }

  .nav-items {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .nav-item {
    background: transparent;
    border: none;
    color: #a0a0b8;
    padding: 0.5rem 0.75rem;
    border-radius: 6px;
    cursor: pointer;
    text-align: left;
    font-size: 0.9rem;
    transition: all 0.15s;
  }

  .nav-item:hover {
    background: #0f3460;
    color: #e0e0e0;
  }

  .nav-item.active {
    background: #e94560;
    color: #fff;
  }

  .content {
    flex: 1;
    padding: 1.5rem;
    overflow-y: auto;
  }
</style>
