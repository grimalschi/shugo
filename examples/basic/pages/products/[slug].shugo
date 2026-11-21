<script module lang="ts">
  import { products } from "../../lib/shop"

  export function getStaticPaths() {
    return products.map((product) => ({ params: { slug: product.slug }, props: { product } }))
  }
</script>

<script lang="ts">
  import { formatPrice, type Product } from "../../lib/shop"

  let { product }: { product: Product } = $props()
</script>

<svelte:head><title>{product.title}</title></svelte:head>

<h1>{product.title}</h1>
<p class="price">{formatPrice(product.price)}</p>
{#if product.discount}
  <p>Discount {product.discount}%</p>
{:else if product.price > 1000}
  <p>Free shipping</p>
{:else}
  <p>Shipping: $300</p>
{/if}
<ul>
  {#each product.tags as tag}<li>{tag}</li>{/each}
</ul>
<p><a href="/">← Catalog</a> · address: {Shugo.url.pathname}, parameter: {Shugo.params.slug}</p>
