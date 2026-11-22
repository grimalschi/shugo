<script lang="ts">
  import Header from "../components/Header.svelte"
  import "../styles/main.css"

  let { children } = $props()
</script>

<Header />
<main>{@render children()}</main>
<footer>© 2026</footer>
