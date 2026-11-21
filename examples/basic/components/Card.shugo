<script lang="ts">
  import { formatPrice, type Product } from "../lib/shop"

  interface Props {
    product: Product
    featured?: boolean
    children?: unknown
  }
  let { product, featured = false, children }: Props = $props()
  const price = product.discount ? product.price * (100 - product.discount) / 100 : product.price
</script>

<a class={featured ? "card featured" : "card"} href={`/products/${product.slug}`}>
  <b>{product.title}</b>
  <span>{formatPrice(price)}</span>
  {#if product.discount}
    <s>{formatPrice(product.price)}</s>
  {/if}
  {@render children?.()}
</a>

<style>
  .card.featured { border-color: #c60; }
</style>
