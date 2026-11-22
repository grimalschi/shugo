<script lang="ts">
  let { current, total, base }: { current: number; total: number; base: string } = $props();

  const url = (n: number) => (n === 1 ? base : `${base}/page/${n}`);
  const pages = $derived(
    [...new Set([1, current - 1, current, current + 1, total])]
      .filter((n) => n >= 1 && n <= total)
      .sort((a, b) => a - b),
  );
  // A gap stands wherever the next page shown is not the neighbour of the one before.
  const items = $derived(pages.flatMap((n, i) => (i > 0 && n - pages[i - 1] > 1 ? ["gap" as const, n] : [n])));
</script>

{#if total > 1}
  <nav class="pagination" aria-label="Pages">
    <ul>
      <li>
        {#if current > 1}
          <a href={url(current - 1)} rel="prev">← Previous</a>
        {:else}
          <span class="disabled" aria-disabled="true">← Previous</span>
        {/if}
      </li>
      {#each items as item}
        <li>
          {#if item === "gap"}
            <span class="gap">…</span>
          {:else if item === current}
            <span aria-current="page">{item}</span>
          {:else}
            <a href={url(item)}>{item}</a>
          {/if}
        </li>
      {/each}
      <li>
        {#if current < total}
          <a href={url(current + 1)} rel="next">Next →</a>
        {:else}
          <span class="disabled" aria-disabled="true">Next →</span>
        {/if}
      </li>
    </ul>
  </nav>
{/if}
