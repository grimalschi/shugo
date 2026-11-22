<script lang="ts">
  type Bar = { label: string; value: number; max: number };
  let { bars }: { bars: Bar[] } = $props();

  const colours = { low: "#e5484d", mid: "#f5a524", high: "#30a46c" };
  const percentOf = ({ value, max }: Bar) => (max === 0 ? 0 : Math.min(100, Math.max(0, Math.round(value / max * 100))));
  const levelOf = (percent: number) => (percent < 30 ? "low" : percent < 70 ? "mid" : "high");
</script>

{#if bars.length === 0}
  <p class="empty">No data</p>
{:else}
  <ul class="bars">
    {#each bars as bar}
      {@const percent = percentOf(bar)}
      {@const level = levelOf(percent)}
      <li class={["bar", level, { done: percent === 100 }]}>
        <span class="label">{bar.label}</span>
        <div class="track" role="progressbar" aria-label={bar.label} aria-valuenow={percent} aria-valuemin="0" aria-valuemax="100">
          <div class="fill" style:width="{percent}%" style:background-color={colours[level]}></div>
        </div>
        <span class="value">{percent}%</span>
      </li>
    {/each}
  </ul>
{/if}
