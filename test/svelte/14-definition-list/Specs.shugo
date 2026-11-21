<script lang="ts">
  type Entry = { term: string; values: (string | number)[]; unit?: string };
  let { title, entries }: { title: string; entries: Entry[] } = $props();
  const shown = $derived(
    entries
      .map((entry) => ({ ...entry, values: entry.values.filter((value) => value !== "") }))
      .filter((entry) => entry.values.length > 0),
  );
</script>

<section class="specs">
  <h2>{title}</h2>
  {#if shown.length > 0}
    <dl>
      {#each shown as entry}
        <dt>{entry.term}</dt>
        {#each entry.values as value}<dd>{entry.unit ? `${value} ${entry.unit}` : value}</dd>{/each}
      {/each}
    </dl>
  {:else}
    <p class="empty">No specifications provided</p>
  {/if}
</section>
