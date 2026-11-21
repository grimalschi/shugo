<script lang="ts">
  import { products } from "../lib/shop"

  const links = [
    { href: "/", title: "Home" },
    { href: "/about", title: "About the shop" },
  ]
</script>

<header>
  <nav>
    {#each links as link}
      <a href={link.href}>{link.title}</a>
    {/each}
  </nav>
  <small>Products in the catalog: {products.length}</small>
</header>
