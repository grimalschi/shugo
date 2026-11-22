<script lang="ts">
  type Tile = { title: string; href?: string; wide?: boolean; featured?: boolean; color?: string };
  let { tiles, columns, gap, accent }: { tiles: Tile[]; columns: number; gap: number; accent?: string } = $props();
</script>

<div class="grid" style:--accent={accent || "#e85d04"} style:--cols={columns} style:--gap="{gap}px">
  {#each tiles as tile, i}
    <div class={["tile", { "tile--wide": tile.wide, "tile--featured": tile.featured }]} style:--i={i} style:--tile-color={tile.color || undefined}>
      {#if tile.href}<a href={tile.href}>{tile.title}</a>{:else}<span>{tile.title}</span>{/if}
    </div>
  {:else}
    <p class="grid-empty">Nothing here yet</p>
  {/each}
</div>
