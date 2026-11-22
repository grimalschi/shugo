<script>
  import ProductCard from "../components/ProductCard.svelte"

  const first = Shugo.query("products").order("position").limit(24).all()
</script>

<svelte:head><title>Measure — handmade tableware</title></svelte:head>
<h1>New in the catalog</h1>
<ul class="cards">
  {#each first as item}<li><ProductCard id={item.id} /></li>{/each}
</ul>
