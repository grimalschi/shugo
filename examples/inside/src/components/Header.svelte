<script>
  import { query } from "shugo"
  // The header stands on every page. It is called with no parameters, so it is
  // a shared block: computed once for the whole site.
  const shop = query("shop").get(1)
  const menu = query("categories").order("position").all()
</script>

<header>
  <a href="/">{shop.name}</a>
  <nav>
    {#each menu as category}
      <a href={`/categories/${category.id}`}>{category.title} ({query("products").where({ category: category.id }).count()})</a>
    {/each}
  </nav>
</header>
