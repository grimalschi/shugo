<script>
  import ProductCard from "../components/ProductCard.svelte"

  const products = Shugo.query("products").select("id").order("title").all()
</script>

<svelte:head><title>Pottery Shop</title></svelte:head>
<h1>All products</h1>
{#each products as product}<ProductCard id={product.id} />{/each}
