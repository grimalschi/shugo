<script lang="ts">
  let { rating, votes }: { rating: number; votes: number } = $props();
  const shown = $derived(votes === 0 ? 0 : Math.round(rating * 2) / 2);
</script>

<div class={["rating", { "rating--empty": votes === 0 }]}>
  <span class="stars" role="img" aria-label="Rating {shown} out of 5">
    {#each [1, 2, 3, 4, 5] as n}
      <span class={["star", n <= shown ? "full" : n - 0.5 === shown ? "half" : "empty"]}></span>
    {/each}
  </span>
  <span class="votes">{votes === 0 ? "no ratings" : `Ratings: ${votes}`}</span>
</div>
