<script lang="ts">
  type Tag = { name: string; count: number };
  let { tags }: { tags: Tag[] } = $props();

  const shown = $derived(
    tags
      .filter((tag) => tag.count > 0)
      .sort((a, b) => b.count - a.count || (a.name < b.name ? -1 : a.name > b.name ? 1 : 0)),
  );
  const min = $derived(Math.min(...shown.map((tag) => tag.count)));
  const max = $derived(Math.max(...shown.map((tag) => tag.count)));
  const size = (count: number) => (max === min ? 22 : Math.round(12 + ((count - min) * 20) / (max - min)));
</script>

{#if shown.length === 0}
  <p class="empty">No tags yet</p>
{:else}
  <ul class="tag-cloud">
    {#each shown as tag (tag.name)}
      <li><a href="/tags/{encodeURIComponent(tag.name)}" style:font-size="{size(tag.count)}px">{tag.name}</a></li>
    {/each}
  </ul>
{/if}
