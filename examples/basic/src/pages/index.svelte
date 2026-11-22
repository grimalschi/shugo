<script lang="ts">
  import Card from "../components/Card.svelte"
  import { products } from "../lib/shop"
</script>

<svelte:head><title>Catalog</title></svelte:head>

<h1>Catalog</h1>
{#each products as product, i}
  <Card {product} featured={i === 0}>
    {#if product.tags.length > 0}
      <small>{product.tags.join(", ")}</small>
    {/if}
  </Card>
{/each}
