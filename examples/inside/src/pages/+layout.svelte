<script>
  import Header from "../components/Header.svelte"

  let { children } = $props()
</script>

<Header />
<main>{@render children()}</main>
