<script>
  import { search } from "shugo:catalog"
  import Catalog from "../components/Catalog.island.svelte"
</script>

<svelte:head><title>Ceramics catalog — Measure</title></svelte:head>

<h1>Ceramics catalog</h1>
<Catalog request:initial={search()} action="/catalog" endpoint="/_islands/catalog" />
