<script lang="ts">
  type Item = { title: string; url?: string };
  let { items, columns }: { items: Item[]; columns: number } = $props();

  const rows = $derived(
    Array.from({ length: Math.ceil(items.length / columns) }, (_, r) => items.slice(r * columns, (r + 1) * columns)),
  );
</script>

{#if items.length === 0}
  <p class="empty">Nothing found</p>
{:else}
  <div class="grid" style:--columns={columns}>
    {#each rows as row, r}
      <div class="row" data-row={r + 1}>
        {#each row as item}
          <div class="cell">{#if item.url}<a href={item.url}>{item.title}</a>{:else}{item.title}{/if}</div>
        {/each}
        {#each { length: columns - row.length }}
          <div class="pad" aria-hidden="true"></div>
        {/each}
      </div>
    {/each}
  </div>
{/if}
