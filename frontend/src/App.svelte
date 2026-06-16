<script>
  import PluginManager from './lib/plugin-manager/PluginManager.svelte';
  import Sidebar from './lib/shell/Sidebar.svelte';
  import ViewContainer from './lib/shell/ViewContainer.svelte';

  let currentView = 'plugin-manager';
</script>

<main>
  <Sidebar />

  <section class="content">
    {#if currentView === 'plugin-manager'}
      <PluginManager />
    {:else}
      <ViewContainer />
    {/if}
  </section>
</main>

<style>
  main {
    display: flex;
    height: 100vh;
    background: #1a1a2e;
  }
  .content {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }
</style>
