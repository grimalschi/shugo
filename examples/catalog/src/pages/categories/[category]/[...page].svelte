<script module>
  // Every category is a list of pages, 24 products on each. The route counts
  // the products to know how many pages there are; every page asks for its
  // own products.
  export function getStaticPaths({ query, paginate }) {
    return query("categories").all().flatMap(({ id }) =>
      paginate(query("products").where({ category: id }).count(), { pageSize: 24, params: { category: id } }),
    )
  }
</script>

<script>
  import ProductCard from "../../../components/ProductCard.svelte"

  const category = Shugo.query("categories").get(Shugo.params.category)
  const page = Shugo.paginate(Shugo.query("products").where({ category: category.id }).order("position"), { pageSize: 24 })
</script>

<svelte:head><title>{category.title} — Measure</title></svelte:head>
<h1>{category.title}</h1>
<p>Products: {page.total}. Page {page.currentPage} of {page.lastPage}.</p>
<ul class="cards">
  {#each page.data as item}<li><ProductCard id={item.id} /></li>{/each}
</ul>
<nav class="pages">
  {#if page.url.prev}<a href={page.url.prev}>← Previous</a>{/if}
  {#if page.url.next}<a href={page.url.next}>Next →</a>{/if}
</nav>
