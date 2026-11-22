<script>
  import { query } from "shugo"
  import ProductCard from "./ProductCard.svelte"

  const shop = query("shop").get(1)
  // The menu shows how many products every category has.
  const menu = query("categories").order("position").limit(8).all().map((category) => ({
    href: `/categories/${category.id}`,
    title: category.title,
    count: query("products").where({ category: category.id }).count(),
  }))
  const featured = shop.featured ? query("products").get(shop.featured) : null
</script>

<header>
  <a class="brand" href="/">{shop.name}</a>
  <nav>
    {#each menu as item}
      <a href={item.href}>{item.title} <small>{item.count}</small></a>
    {/each}
  </nav>
  {#if featured}
    <p>Product of the month: {featured.title}</p>
    <ProductCard id={featured.id} />
  {/if}
</header>
