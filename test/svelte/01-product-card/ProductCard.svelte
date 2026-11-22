<script lang="ts">
  type Props = {
    title: string; url: string; price: string; oldPrice?: string;
    badges: string[]; inStock: boolean; rating?: number;
  };
  let { title, url, price, oldPrice, badges, inStock, rating }: Props = $props();
</script>

<article class={["card", !inStock && "out"]}>
  <a href={url}><h3>{title}</h3></a>
  <p class="price">{#if oldPrice}<s>{oldPrice}</s> <b>{price}</b>{:else}<b>{price}</b>{/if}</p>
  {#if badges.length > 0}<ul class="badges">{#each badges as badge}<li>{badge}</li>{/each}</ul>{/if}
  {#if rating !== undefined}<p class="rating" title="Rating: {rating}">★ {rating}</p>{/if}
  {#if inStock}<button type="button">Add to cart</button>{:else}<button type="button" disabled>Out of stock</button>{/if}
</article>
