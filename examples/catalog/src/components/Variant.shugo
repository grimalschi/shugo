<script lang="ts">
  // The page gives the whole row of a variant. What the page then depends on
  // is what is read of the row here: the title and the stock, not the shelf.
  let { variant }: { variant: { title: string; stock: number } } = $props()
</script>

<li class={variant.stock > 0 ? "variant" : "variant out"}>
  {variant.title}:
  {#if variant.stock > 0}{variant.stock} in stock{:else}out of stock{/if}
</li>
