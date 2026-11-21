<script>
  let { item } = $props()
</script>

<article class="product-card" data-product-id={item.id}>
  <p class="product-category">{item.categoryTitle}</p>
  <h3>{item.title}</h3>
  <p class="product-price">
    <strong>{item.priceLabel}</strong>
    {#if item.discounted}<del>{item.originalPriceLabel}</del>{/if}
  </p>
  <p class={item.stockClass}>{item.stockLabel}</p>
  {#if item.hasSizes}<p class="product-sizes">{item.sizesLabel}</p>{/if}
</article>
