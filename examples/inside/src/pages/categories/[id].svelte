<script module>
  import { query, page } from "shugo"
  export function getStaticPaths() {
    return query("categories").select("id").all().map((row) => ({ params: { id: row.id } }))
  }
</script>

<script>
  import ProductCard from "../../components/ProductCard.svelte"

  const category = query("categories").get(page.params.id)
  const products = query("products").select("id").where({ category: category.id }).order("title").all()
</script>

<svelte:head><title>{category.title} — Pottery Shop</title></svelte:head>
<h1>{category.title}</h1>
{#each products as product}<ProductCard id={product.id} />{/each}
