<script module>
  import { query, paginate, page } from "shugo"
  // Every category is a list of pages, 24 products on each. The route counts
  // the products to know how many pages there are; every page asks for its
  // own products.
  export function getStaticPaths() {
    return query("categories").all().flatMap(({ id }) =>
      paginate(query("products").where({ category: id }).count(), { pageSize: 24, params: { category: id } }),
    )
  }
</script>

<script>
  import ProductCard from "../../../components/ProductCard.svelte"

  const category = query("categories").get(page.params.category)
  const list = paginate(query("products").where({ category: category.id }).order("position"), { pageSize: 24 })
</script>

<svelte:head><title>{category.title} — Measure</title></svelte:head>
<h1>{category.title}</h1>
<p>Products: {list.total}. Page {list.currentPage} of {list.lastPage}.</p>
<ul class="cards">
  {#each list.data as item}<li><ProductCard id={item.id} /></li>{/each}
</ul>
<nav class="pages">
  {#if list.url.prev}<a href={list.url.prev}>← Previous</a>{/if}
  {#if list.url.next}<a href={list.url.next}>Next →</a>{/if}
</nav>
