<script>
  import ProductCard from "./ProductCard.svelte"

  const shop = Shugo.query("shop").get(1)
  // The menu shows how many products every category has.
  const menu = Shugo.query("categories").order("position").limit(8).all().map((category) => ({
    href: `/categories/${category.id}`,
    title: category.title,
    count: Shugo.query("products").where({ category: category.id }).count(),
  }))
  const featured = shop.featured ? Shugo.query("products").get(shop.featured) : null
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
