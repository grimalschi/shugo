<script>
  import Counter from "../components/Counter.island.svelte"
</script>

<svelte:head><title>Two islands</title></svelte:head>

<h1>Two islands on one page</h1>
<p>These islands use no search. Each has its own state, and the page loads their browser module once.</p>
<Counter title="Left island" start={0} />
<Counter title="Right island" start={10} />
