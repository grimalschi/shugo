<script lang="ts">
  type Product = { name: string; price: number; stock: number; url?: string };
  let { products }: { products: Product[] } = $props();

  const shown = 3;
  const inStock = $derived(
    products
      .filter((p) => p.stock > 0)
      .sort((a, b) => a.price - b.price || (a.name < b.name ? -1 : a.name > b.name ? 1 : 0)),
  );
  const rest = $derived(inStock.length - shown);

  function plural(n: number): string {
    return n === 1 ? "item" : "items";
  }
</script>

<section class="products">
  {#if inStock.length === 0}
    <p class="empty">Out of stock</p>
  {:else}
    <ul>
      {#each inStock.slice(0, shown) as p (p.name)}
        <li class={{ few: p.stock < 3 }}>
          {#if p.url}<a href={p.url}>{p.name}</a>{:else}<span>{p.name}</span>{/if}
          <b>${p.price}</b>
        </li>
      {/each}
    </ul>
    {#if rest > 0}<p class="more">and {rest} more {plural(rest)}</p>{/if}
  {/if}
</section>
