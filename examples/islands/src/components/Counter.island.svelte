<script>
  import { onMount } from "svelte"
  let { title, start } = $props()
  let clicks = $state(0)
  let count = $derived(Number(start) + clicks)
  let ready = $state(false)
  onMount(() => { ready = true })
</script>

<section data-counter-ready={ready ? "" : undefined}>
  <h2>{title}</h2>
  <button type="button" onclick={() => clicks++} aria-label={title + ": increment"}>Count {count}</button>
</section>

<style>
  section { border: 1px solid var(--border); border-radius: 8px; padding: 1rem; }
  h2 { margin-top: 0; }
</style>
