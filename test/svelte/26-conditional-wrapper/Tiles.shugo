<script lang="ts">
  type Tile = { title: string; text: string; href?: string; image?: { src: string; alt: string }; wide?: boolean };
  let { tiles, columns = 3 }: { tiles: Tile[]; columns?: number } = $props();
  const isExternal = (href: string) => href.startsWith("http://") || href.startsWith("https://");
</script>

{#snippet body(tile: Tile)}
  {#if tile.image}<img src={tile.image.src} alt={tile.image.alt} loading="lazy" />{/if}
  <h3>{tile.title}</h3>
  {#if tile.text}<p>{tile.text}</p>{/if}
{/snippet}

{#if tiles.length > 0}
  <div class="tiles" style:--columns={columns}>
    {#each tiles as tile}
      {#if tile.href}
        {@const external = isExternal(tile.href)}
        <a class={["tile", tile.wide && "wide"]} href={tile.href} target={external ? "_blank" : undefined} rel={external ? "noopener" : undefined}>
          {@render body(tile)}
        </a>
      {:else}
        <div class={["tile", tile.wide && "wide"]}>{@render body(tile)}</div>
      {/if}
    {/each}
  </div>
{/if}
