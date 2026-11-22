<script lang="ts">
  type Props = {
    name: string;
    sku: string;
    description: string;
    image?: string;
    brand?: string;
    price?: number;
    inStock: boolean;
    ratings: number[];
  };
  let { name, sku, description, image, brand, price, inStock, ratings }: Props = $props();

  const rating = $derived(ratings.length > 0 ? Math.round((ratings.reduce((a, b) => a + b, 0) / ratings.length) * 10) / 10 : undefined);
  const data = $derived({
    "@context": "https://schema.org",
    "@type": "Product",
    name,
    sku,
    description,
    image: image || undefined,
    brand: brand ? { "@type": "Brand", name: brand } : undefined,
    offers: price !== undefined ? {
      "@type": "Offer",
      price: price.toFixed(2),
      priceCurrency: "USD",
      availability: inStock ? "https://schema.org/InStock" : "https://schema.org/OutOfStock",
    } : undefined,
    aggregateRating: rating !== undefined ? { "@type": "AggregateRating", ratingValue: rating, reviewCount: ratings.length } : undefined,
  });
  // Every "<" becomes its JSON escape, so a closing script tag inside a string cannot end the script.
  const json = $derived(JSON.stringify(data).replace(/</g, "\\u003c"));
</script>

<article class="product">
  <h1>{name}</h1>
  {#if image}<img src={image} alt={name} loading="lazy" />{/if}
  {#if brand}<p class="brand">Brand: {brand}</p>{/if}
  <p class="description">{description}</p>
  <p class="sku">SKU: {sku}</p>
  {#if price !== undefined}
    <p class="offer"><span class="price">${price.toFixed(2)}</span> <span class="stock">{inStock ? "In stock" : "Out of stock"}</span></p>
  {:else}
    <p class="offer">Price on request</p>
  {/if}
  {#if rating !== undefined}<p class="rating">Rating {rating} out of 5, reviews: {ratings.length}</p>{/if}
</article>
<!-- A <script> in the markup takes no expressions, so the whole element is inserted as HTML. -->
{@html `<script type="application/ld+json">${json}</script>`}
