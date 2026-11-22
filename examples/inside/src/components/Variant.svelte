<script>
  // A variant is given as a row, an object: so this is an ordinary component,
  // computed inside the page that shows it.
  let { variant } = $props()
</script>

<li>
  {variant.title}:
  {#if variant.stock > 0}{variant.stock} in stock{:else}out of stock{/if}
</li>
