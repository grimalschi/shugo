<script lang="ts">
  type Bar = { label: string; value: number };
  let { title, data, width, height }: { title: string; data: Bar[]; width: number; height: number } = $props();
  const plot = $derived(height - 20);
  const step = $derived(data.length > 0 ? Math.floor(width / data.length) : 0);
  const max = $derived(Math.max(0, ...data.map((bar) => bar.value)));
  const top = $derived(data.findIndex((bar) => bar.value === max));
</script>

<svg class="chart" viewBox="0 0 {width} {height}" {width} {height} role="img" aria-label={title}>
  {#each data as bar, i}
    {@const h = max > 0 ? Math.round(bar.value / max * plot) : 0}
    <rect class={["bar", { max: i === top }]} x={i * step + 4} y={plot - h} width={step - 8} height={h} data-value={bar.value} />
    <text x={i * step + Math.floor(step / 2)} y={height - 6} text-anchor="middle">{bar.label}</text>
  {:else}
    <text class="empty" x={Math.round(width / 2)} y={Math.round(height / 2)} text-anchor="middle">no data</text>
  {/each}
</svg>
